package handlers

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/approval"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/execution"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/models"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/repository"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_loan "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc/codes"
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

	// 3. Calculate price per unit and approximate price
	pricePerUnit, _ := execution.CalculatePrice(orderType, req.Direction, listing.Ask, listing.Bid, req.LimitValue, req.StopValue)
	contractSize := int32(1)
	if futures := listingResp.GetFutures(); futures != nil {
		contractSize = int32(futures.ContractSize)
	}
	approxPrice := execution.ApproximatePrice(contractSize, pricePerUnit, req.Quantity)

	// 4. After-hours check via working hours
	afterHours := s.checkAfterHours(ctx, listingResp)

	// 5. Determine initial approval status
	isActuary := false
	needsApproval := false
	if req.UserType == "EMPLOYEE" {
		limitAmount, usedLimit, needApprovalFlag, err := repository.GetActuaryInfo(ctx, s.EmployeeDB, req.UserId)
		if err == nil {
			isActuary = true
			needsApproval = approval.NeedsApproval(needApprovalFlag, usedLimit, limitAmount, approxPrice)
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
	}

	id, err := repository.InsertOrder(ctx, s.DB, o)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to insert order: %v", err)
	}

	return &pb.CreateOrderResponse{
		OrderId:          id,
		OrderType:        orderType,
		Status:           initialStatus,
		ApproximatePrice: approxPrice,
	}, nil
}

// ListOrders returns all orders visible to a supervisor, with optional filters.
func (s *OrderServer) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	orders, err := repository.ListOrders(ctx, s.DB, req.Status, req.AgentId)
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
	return nil
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

// checkAfterHours fetches working hours for the listing's exchange and checks after-hours status.
func (s *OrderServer) checkAfterHours(ctx context.Context, listingResp *pb_sec.GetListingByIdResponse) bool {
	micCode := "" // resolve MIC from exchange_acronym via SecuritiesDB
	if err := s.SecuritiesDB.QueryRowContext(ctx,
		`SELECT mic_code FROM stock_exchanges WHERE acronym = $1`,
		listingResp.Summary.ExchangeAcronym,
	).Scan(&micCode); err != nil || micCode == "" {
		return false
	}

	hoursResp, err := s.SecuritiesClient.GetWorkingHours(ctx, &pb_sec.GetWorkingHoursRequest{MicCode: micCode})
	if err != nil {
		return false
	}

	// Use the "regular" session's close time
	for _, h := range hoursResp.Hours {
		if h.Segment == "regular" {
			exchResp, err := s.SecuritiesClient.GetStockExchangeByMIC(ctx, &pb_sec.GetStockExchangeByMICRequest{MicCode: micCode})
			if err != nil {
				return false
			}
			return execution.IsAfterHours(h.CloseTime, exchResp.Exchange.Timezone, time.Now())
		}
	}
	return false
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
	}
}

func ordersToProto(orders []models.Order) []*pb.Order {
	result := make([]*pb.Order, len(orders))
	for i, o := range orders {
		result[i] = orderToProto(o)
	}
	return result
}
