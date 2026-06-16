package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/approval"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/execution"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/models"
	orderqueue "github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/queue"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/repository"
	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_loan "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
)

type OrderServer struct {
	pb.UnimplementedOrderServiceServer
	DB               *sql.DB
	AccountDB        *sql.DB
	SecuritiesDB     *sql.DB
	ExchangeDB       *sql.DB
	EmployeeDB       *sql.DB
	SecuritiesClient pb_sec.SecuritiesServiceClient
	LoanClient       pb_loan.LoanServiceClient
	EmployeeClient   pb_emp.EmployeeServiceClient
	PortfolioClient  pb_portfolio.PortfolioServiceClient
	ClientClient     pb_client.ClientServiceClient
	AmqpChannel      *amqp.Channel
}

// sendOrderNotif fetches user email and publishes an order status email notification.
// Runs in a goroutine — errors are logged and swallowed.
func (s *OrderServer) sendOrderNotif(ctx context.Context, orderID int64, userID int64, userType, ticker, direction, status string, quantity, filledQty int32, pricePerUnit float64) {
	if s.AmqpChannel == nil {
		return
	}
	var email, firstName string
	if userType == "CLIENT" && s.ClientClient != nil {
		if resp, err := s.ClientClient.GetClientById(ctx, &pb_client.GetClientByIdRequest{Id: userID}); err == nil && resp.Client != nil {
			email = resp.Client.Email
			firstName = resp.Client.FirstName
		}
	} else if s.EmployeeClient != nil {
		if resp, err := s.EmployeeClient.GetEmployeeById(ctx, &pb_emp.GetEmployeeByIdRequest{Id: userID}); err == nil && resp.Employee != nil {
			email = resp.Employee.Email
			firstName = resp.Employee.FirstName
		}
	}
	if email == "" {
		return
	}
	_ = orderqueue.PublishOrderStatus(s.AmqpChannel, orderqueue.OrderStatusMessage{
		Email:        email,
		FirstName:    firstName,
		OrderID:      orderID,
		Ticker:       ticker,
		Direction:    direction,
		Status:       status,
		Quantity:     quantity,
		FilledQty:    filledQty,
		PricePerUnit: pricePerUnit,
	})
}

func (s *OrderServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "order-service OK"}, nil
}

// CreateOrder creates a new order, determines its type, approval status, and approximate price.
func (s *OrderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	// 1. Determine order type from limit/stop values
	orderType := determineOrderType(req.LimitValue, req.StopValue)

	// 2. Fetch listing for current prices and contract size
	listingResp, err := s.SecuritiesClient.GetListingById(ctx, &pb_sec.GetListingByIdRequest{Id: req.AssetId})
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to fetch listing: %v", err)
	}
	listing := listingResp.Summary

	// 3. Exchange open check — reject if market is closed; determine after-hours flag.
	isOpen, afterHours := s.checkExchangeStatus(ctx, listingResp)
	if !isOpen {
		return nil, grpcstatus.Errorf(codes.FailedPrecondition, "exchange is currently closed")
	}

	// 4. Calculate price per unit and approximate price
	pricePerUnit, _ := execution.CalculatePrice(orderType, req.Direction, listing.Ask, listing.Bid, req.LimitValue, req.StopValue)
	contractSize := int32(1)
	if futures := listingResp.GetFutures(); futures != nil {
		contractSize = int32(futures.ContractSize)
	}
	approxPrice := execution.ApproximatePrice(contractSize, pricePerUnit, req.Quantity)

	// 4b. For CLIENT BUY orders, reject if account has insufficient funds.
	if req.Direction == "BUY" && req.UserType == "CLIENT" {
		var availBalance float64
		if err := s.AccountDB.QueryRowContext(ctx,
			`SELECT available_balance FROM accounts WHERE id = $1`, req.AccountId,
		).Scan(&availBalance); err != nil {
			return nil, grpcstatus.Errorf(codes.Internal, "failed to check balance: %v", err)
		}
		if availBalance < approxPrice {
			return nil, grpcstatus.Errorf(codes.FailedPrecondition, "insufficient funds")
		}
	}

	// 3c. For SELL orders, reject if the user holds fewer securities than requested.
	if req.Direction == "SELL" {
		portfolioCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("user-type", req.UserType))
		portfolioResp, err := s.PortfolioClient.GetPortfolio(portfolioCtx, &pb_portfolio.GetPortfolioRequest{
			UserId:   req.UserId,
			UserType: req.UserType,
		})
		if err != nil {
			return nil, grpcstatus.Errorf(codes.Internal, "failed to fetch portfolio: %v", err)
		}
		var held int32
		for _, entry := range portfolioResp.Entries {
			if entry.ListingId == req.AssetId {
				held = entry.Amount
				break
			}
		}
		if held < req.Quantity {
			return nil, grpcstatus.Errorf(codes.FailedPrecondition, "insufficient holdings: have %d, need %d", held, req.Quantity)
		}
	}

	// 5. Determine initial approval status
	// Convert approxPrice to RSD so it can be compared against the RSD-denominated actuary limit.
	approxPriceRSD := approxPrice
	if req.UserType == "EMPLOYEE" {
		var listingCurrency string
		_ = s.SecuritiesDB.QueryRowContext(ctx, `
			SELECT e.currency FROM listing l
			JOIN stock_exchanges e ON l.exchange_id = e.id
			WHERE l.id = $1`, req.AssetId,
		).Scan(&listingCurrency)
		if listingCurrency != "" && listingCurrency != "RSD" {
			var sellingRate float64
			err := s.ExchangeDB.QueryRowContext(ctx,
				`SELECT selling_rate FROM daily_exchange_rates WHERE currency_code = $1 AND date = CURRENT_DATE`,
				listingCurrency,
			).Scan(&sellingRate)
			if err == nil && sellingRate > 0 {
				approxPriceRSD = approxPrice * sellingRate
			}
		}
	}

	isActuary := false
	needsApproval := false
	if req.UserType == "EMPLOYEE" {
		limitAmount, usedLimit, needApprovalFlag, err := repository.GetActuaryInfo(ctx, s.EmployeeDB, req.UserId)
		if err == nil {
			isActuary = true
			needsApproval = approval.NeedsApproval(needApprovalFlag, usedLimit, limitAmount, approxPriceRSD, req.Direction)
		}
		// sql.ErrNoRows → supervisor, isActuary stays false
	}
	initialStatus := approval.DetermineInitialStatus(req.UserType, isActuary, needsApproval)

	// 6. Build and insert order
	var limitVal, stopVal *float64
	if req.LimitValue != 0 {
		v := req.LimitValue
		limitVal = &v
	}
	if req.StopValue != 0 {
		v := req.StopValue
		stopVal = &v
	}

	o := &models.Order{
		UserID:            req.UserId,
		UserType:          req.UserType,
		AssetID:           req.AssetId,
		OrderType:         orderType,
		Quantity:          req.Quantity,
		ContractSize:      contractSize,
		PricePerUnit:      pricePerUnit,
		LimitValue:        limitVal,
		StopValue:         stopVal,
		Direction:         req.Direction,
		Status:            initialStatus,
		RemainingPortions: req.Quantity,
		AfterHours:        afterHours,
		IsAON:             req.IsAon,
		IsMargin:          req.IsMargin,
		AccountID:         req.AccountId,
		FundID:            req.FundId,
	}

	id, err := repository.InsertOrder(ctx, s.DB, o)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to insert order: %v", err)
	}

	// Deduct from agent's used limit for auto-approved BUY orders.
	// SELL orders do not count against the limit.
	// PENDING orders are deducted in ApproveOrder when the supervisor approves.
	if isActuary && initialStatus == "APPROVED" && req.Direction == "BUY" {
		if err := repository.DeductActuaryUsedLimit(ctx, s.EmployeeDB, req.UserId, approxPriceRSD); err != nil {
			log.Printf("order: DeductActuaryUsedLimit user=%d: %v", req.UserId, err)
		}
	}

	// Non-blocking notification
	go func() {
		ticker := ""
		if listingResp.Summary != nil {
			ticker = listingResp.Summary.Ticker
		}
		s.sendOrderNotif(context.Background(), id, req.UserId, req.UserType, ticker, req.Direction, initialStatus, req.Quantity, 0, approxPrice)
	}()

	return &pb.CreateOrderResponse{
		OrderId:          id,
		OrderType:        orderType,
		Status:           initialStatus,
		ApproximatePrice: approxPrice,
	}, nil
}

// ListOrders returns orders. When req.MyOrders is true, returns only the caller's own orders
// with enriched ticker/asset_type data. Otherwise returns all EMPLOYEE orders for supervisors.
func (s *OrderServer) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	var orders []models.Order
	var err error
	if req.MyOrders {
		orders, err = repository.GetMyOrders(ctx, s.DB, s.SecuritiesDB, req.CallerId, req.CallerType,
			req.Status, req.AssetType, req.Direction, req.FromDate, req.ToDate, req.Page, req.PageSize)
	} else {
		orders, err = repository.ListOrders(ctx, s.DB, req.Status, req.AgentId)
	}
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to list orders: %v", err)
	}
	return &pb.ListOrdersResponse{Orders: ordersToProto(orders)}, nil
}

// GetOrderById returns a single order by ID.
func (s *OrderServer) GetOrderById(ctx context.Context, req *pb.GetOrderByIdRequest) (*pb.GetOrderByIdResponse, error) {
	o, err := repository.GetOrderByID(ctx, s.DB, req.Id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, grpcstatus.Errorf(codes.NotFound, "order %d not found", req.Id)
	}
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to get order: %v", err)
	}
	return &pb.GetOrderByIdResponse{Order: orderToProto(o)}, nil
}

// ApproveOrder approves a PENDING order (supervisor only).
func (s *OrderServer) ApproveOrder(ctx context.Context, req *pb.ApproveOrderRequest) (*pb.ApproveOrderResponse, error) {
	if err := approval.ApproveOrder(ctx, s.DB, s.EmployeeDB, req.OrderId, req.SupervisorId); err != nil {
		if errors.Is(err, approval.ErrNotPending) {
			return nil, grpcstatus.Errorf(codes.FailedPrecondition, "order is not in PENDING status")
		}
		if errors.Is(err, sql.ErrNoRows) {
			return nil, grpcstatus.Errorf(codes.NotFound, "order not found")
		}
		return nil, grpcstatus.Errorf(codes.Internal, "failed to approve order: %v", err)
	}
	if s.EmployeeClient != nil {
		_, _ = s.EmployeeClient.LogAuditEvent(ctx, &pb_emp.AuditLogRequest{
			ActorId: req.SupervisorId, ActorType: "EMPLOYEE",
			Action: "ORDER_APPROVED", TargetId: req.OrderId, TargetType: "ORDER",
		})
	}
	go s.notifyOrderByID(req.OrderId, "APPROVED")
	return &pb.ApproveOrderResponse{}, nil
}

// DeclineOrder declines a PENDING order (supervisor only).
func (s *OrderServer) DeclineOrder(ctx context.Context, req *pb.DeclineOrderRequest) (*pb.DeclineOrderResponse, error) {
	if err := approval.DeclineOrder(ctx, s.DB, req.OrderId, req.SupervisorId); err != nil {
		if errors.Is(err, approval.ErrNotPending) {
			return nil, grpcstatus.Errorf(codes.FailedPrecondition, "order is not in PENDING status")
		}
		if errors.Is(err, sql.ErrNoRows) {
			return nil, grpcstatus.Errorf(codes.NotFound, "order not found")
		}
		return nil, grpcstatus.Errorf(codes.Internal, "failed to decline order: %v", err)
	}
	if s.EmployeeClient != nil {
		_, _ = s.EmployeeClient.LogAuditEvent(ctx, &pb_emp.AuditLogRequest{
			ActorId: req.SupervisorId, ActorType: "EMPLOYEE",
			Action: "ORDER_DECLINED", TargetId: req.OrderId, TargetType: "ORDER",
		})
	}
	go s.notifyOrderByID(req.OrderId, "DECLINED")
	return &pb.DeclineOrderResponse{}, nil
}

// CancelOrder cancels an entire unfulfilled order.
func (s *OrderServer) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	if err := s.cancelOrderChecked(ctx, req.OrderId, req.UserId); err != nil {
		return nil, err
	}
	return &pb.CancelOrderResponse{}, nil
}

// CancelOrderPortions cancels remaining unfilled portions of an order.
func (s *OrderServer) CancelOrderPortions(ctx context.Context, req *pb.CancelOrderPortionsRequest) (*pb.CancelOrderPortionsResponse, error) {
	if err := s.cancelOrderChecked(ctx, req.OrderId, req.UserId); err != nil {
		return nil, err
	}
	return &pb.CancelOrderPortionsResponse{}, nil
}

// cancelOrderChecked validates and cancels an order.
func (s *OrderServer) cancelOrderChecked(ctx context.Context, orderID, userID int64) error {
	o, err := repository.GetOrderByID(ctx, s.DB, orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return grpcstatus.Errorf(codes.NotFound, "order %d not found", orderID)
	}
	if err != nil {
		return grpcstatus.Errorf(codes.Internal, "failed to fetch order: %v", err)
	}
	if o.IsDone || o.RemainingPortions == 0 {
		return grpcstatus.Errorf(codes.FailedPrecondition, "order has no remaining portions to cancel")
	}
	if o.UserID != userID {
		return grpcstatus.Errorf(codes.PermissionDenied, "order does not belong to this user")
	}
	if err := repository.CancelOrder(ctx, s.DB, orderID); err != nil {
		return grpcstatus.Errorf(codes.Internal, "failed to cancel order: %v", err)
	}
	go s.notifyOrderByID(orderID, "CANCELLED")
	return nil
}

// notifyOrderByID fetches the order and sends an email notification with the given status.
func (s *OrderServer) notifyOrderByID(orderID int64, status string) {
	ctx := context.Background()
	o, err := repository.GetOrderByID(ctx, s.DB, orderID)
	if err != nil {
		return
	}
	var ticker string
	_ = s.SecuritiesDB.QueryRowContext(ctx, `SELECT ticker FROM listing WHERE id = $1`, o.AssetID).Scan(&ticker)
	s.sendOrderNotif(ctx, orderID, o.UserID, o.UserType, ticker, o.Direction, status, o.Quantity, o.Quantity-o.RemainingPortions, o.PricePerUnit)
}

// determineOrderType infers MARKET/LIMIT/STOP/STOP_LIMIT from the presence of limit/stop values.
func determineOrderType(limitValue, stopValue float64) string {
	hasLimit := limitValue != 0
	hasStop := stopValue != 0
	switch {
	case !hasLimit && !hasStop:
		return "MARKET"
	case hasLimit && !hasStop:
		return "LIMIT"
	case !hasLimit && hasStop:
		return "STOP"
	default:
		return "STOP_LIMIT"
	}
}

// checkExchangeStatus returns (isOpen, afterHours).
// isOpen=false means the exchange is closed and the order should be rejected.
// afterHours=true means the order is placed during pre/post-market hours.
// On any error the call fails open (isOpen=true, afterHours=false) so a
// temporary securities-service hiccup does not block all trading.
func (s *OrderServer) checkExchangeStatus(ctx context.Context, listingResp *pb_sec.GetListingByIdResponse) (isOpen bool, afterHours bool) {
	var micCode string
	if err := s.SecuritiesDB.QueryRowContext(ctx,
		`SELECT mic_code FROM stock_exchanges WHERE acronym = $1`,
		listingResp.Summary.ExchangeAcronym,
	).Scan(&micCode); err != nil || micCode == "" {
		log.Printf("order: checkExchangeStatus: MIC lookup failed for acronym %q, failing open", listingResp.Summary.ExchangeAcronym)
		return true, false
	}

	resp, err := s.SecuritiesClient.IsExchangeOpen(ctx, &pb_sec.IsExchangeOpenRequest{MicCode: micCode})
	if err != nil {
		log.Printf("order: checkExchangeStatus: IsExchangeOpen(%s) error: %v, failing open", micCode, err)
		return true, false
	}

	if !resp.IsOpen {
		return false, false
	}
	afterHours = resp.Segment == "pre_market" || resp.Segment == "post_market"
	return true, afterHours
}

// orderToProto converts a models.Order to its proto representation.
func orderToProto(o models.Order) *pb.Order {
	var approvedBy int64
	if o.ApprovedBy != nil {
		approvedBy = *o.ApprovedBy
	}
	var limitVal, stopVal float64
	if o.LimitValue != nil {
		limitVal = *o.LimitValue
	}
	if o.StopValue != nil {
		stopVal = *o.StopValue
	}
	return &pb.Order{
		Id:                o.ID,
		UserId:            o.UserID,
		AssetId:           o.AssetID,
		OrderType:         o.OrderType,
		Quantity:          o.Quantity,
		ContractSize:      o.ContractSize,
		PricePerUnit:      o.PricePerUnit,
		LimitValue:        limitVal,
		StopValue:         stopVal,
		Direction:         o.Direction,
		Status:            o.Status,
		ApprovedBy:        approvedBy,
		IsDone:            o.IsDone,
		LastModification:  o.LastModification.Format(time.RFC3339),
		RemainingPortions: o.RemainingPortions,
		AfterHours:        o.AfterHours,
		IsAon:             o.IsAON,
		IsMargin:          o.IsMargin,
		AccountId:         o.AccountID,
		FundId:            o.FundID,
		CommissionPaid:    o.CommissionPaid,
		Ticker:            o.Ticker,
		AssetType:         o.AssetType,
	}
}

func ordersToProto(orders []models.Order) []*pb.Order {
	result := make([]*pb.Order, len(orders))
	for i, o := range orders {
		result[i] = orderToProto(o)
	}
	return result
}

// GetActuaryProfits returns realized P&L per actuary from completed EMPLOYEE orders (#226).
// Profit = SUM(fill_price * qty * sign) where sign = +1 for SELL, -1 for BUY, converted to RSD.
// Two-step: (1) aggregate per (user_id, asset_id) from order DB; (2) look up currency from
// SecuritiesDB (listing + stock_exchanges live there, not in the order DB).
func (s *OrderServer) GetActuaryProfits(ctx context.Context, req *pb.GetActuaryProfitsRequest) (*pb.GetActuaryProfitsResponse, error) {
	// Step 1: net P&L per (user_id, asset_id) from order-service DB.
	rows, err := s.DB.QueryContext(ctx, `
		SELECT o.user_id, o.asset_id,
		       SUM(op.price * op.quantity * CASE WHEN o.direction = 'SELL' THEN 1 ELSE -1 END) AS net_pnl
		FROM orders o
		JOIN order_portions op ON op.order_id = o.id
		WHERE o.user_type = 'EMPLOYEE'
		  AND o.is_done   = TRUE
		  AND (cardinality($1::bigint[]) = 0 OR o.user_id = ANY($1::bigint[]))
		GROUP BY o.user_id, o.asset_id`,
		pq.Array(req.UserIds),
	)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "query actuary profits: %v", err)
	}
	defer func() { _ = rows.Close() }()

	type pnlRow struct {
		userID  int64
		assetID int64
		netPnl  float64
	}
	var pnlRows []pnlRow
	assetIDs := make(map[int64]struct{})
	for rows.Next() {
		var r pnlRow
		if err := rows.Scan(&r.userID, &r.assetID, &r.netPnl); err != nil {
			return nil, grpcstatus.Errorf(codes.Internal, "scan actuary profit row: %v", err)
		}
		pnlRows = append(pnlRows, r)
		assetIDs[r.assetID] = struct{}{}
	}

	// Step 2: resolve currency per asset from SecuritiesDB (same pattern as CreateOrder).
	assetCurrency := make(map[int64]string, len(assetIDs))
	for assetID := range assetIDs {
		var currency string
		if err := s.SecuritiesDB.QueryRowContext(ctx,
			`SELECT e.currency FROM listing l JOIN stock_exchanges e ON e.id = l.exchange_id WHERE l.id = $1`,
			assetID,
		).Scan(&currency); err == nil {
			assetCurrency[assetID] = currency
		}
	}

	// Step 3: fetch exchange rates for non-RSD currencies (cached to avoid duplicate queries).
	rateCache := make(map[string]float64)
	for _, currency := range assetCurrency {
		if currency == "" || currency == "RSD" {
			continue
		}
		if _, cached := rateCache[currency]; cached {
			continue
		}
		var rate float64
		if err := s.ExchangeDB.QueryRowContext(ctx,
			`SELECT selling_rate FROM daily_exchange_rates WHERE currency_code = $1 AND date = CURRENT_DATE`,
			currency,
		).Scan(&rate); err == nil && rate > 0 {
			rateCache[currency] = rate
		}
	}

	// Step 4: accumulate profit in RSD per user.
	profitMap := make(map[int64]float64)
	for _, r := range pnlRows {
		currency := assetCurrency[r.assetID]
		amountRSD := r.netPnl
		if currency != "" && currency != "RSD" {
			if rate, ok := rateCache[currency]; ok {
				amountRSD = r.netPnl * rate
			}
		}
		profitMap[r.userID] += amountRSD
	}

	profits := make([]*pb.ActuaryProfit, 0, len(profitMap))
	for userID, profitRSD := range profitMap {
		profits = append(profits, &pb.ActuaryProfit{UserId: userID, ProfitRsd: profitRSD})
	}
	return &pb.GetActuaryProfitsResponse{Profits: profits}, nil
}
