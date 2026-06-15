package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_fund "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type orderResponse struct {
	Id                int64   `json:"id"`
	UserId            int64   `json:"user_id"`
	AgentName         string  `json:"agent_name"`
	AssetId           int64   `json:"asset_id"`
	AssetTicker       string  `json:"asset_ticker"`
	OrderType         string  `json:"order_type"`
	Quantity          int32   `json:"quantity"`
	ContractSize      int32   `json:"contract_size"`
	PricePerUnit      float64 `json:"price_per_unit"`
	LimitValue        float64 `json:"limit_value"`
	StopValue         float64 `json:"stop_value"`
	Direction         string  `json:"direction"`
	Status            string  `json:"status"`
	ApprovedBy        int64   `json:"approved_by"`
	IsDone            bool    `json:"is_done"`
	LastModification  string  `json:"last_modification"`
	RemainingPortions int32   `json:"remaining_portions"`
	AfterHours        bool    `json:"after_hours"`
	IsAon             bool    `json:"is_aon"`
	IsMargin          bool    `json:"is_margin"`
	AccountId         int64   `json:"account_id"`
	CommissionPaid    float64 `json:"commission_paid"`
	Ticker            string  `json:"ticker"`
	AssetType         string  `json:"asset_type"`
}

func orderError(c *gin.Context, err error) {
	switch status.Code(err) {
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
	case codes.PermissionDenied:
		c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
	case codes.FailedPrecondition, codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// CreateOrder handles POST /orders and POST /client/orders.
// fundClient is used when purchaseFor="FUND" to validate fund ownership and liquidity.
func CreateOrder(orderClient pb.OrderServiceClient, fundClient pb_fund.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			AssetId     int64   `json:"assetId"    binding:"required"`
			Quantity    int32   `json:"quantity"   binding:"required"`
			LimitValue  float64 `json:"limitValue"`
			StopValue   float64 `json:"stopValue"`
			IsAon       bool    `json:"isAon"`
			IsMargin    bool    `json:"isMargin"`
			AccountId   int64   `json:"accountId"  binding:"required"`
			Direction   string  `json:"direction"  binding:"required"`
			PurchaseFor string  `json:"purchaseFor"` // "BANK" | "FUND" | ""
			FundId      int64   `json:"fundId"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		fundID := int64(0)
		if body.PurchaseFor == "FUND" && body.FundId != 0 && middleware.CallerHasPermission(c, "SUPERVISOR") {
			// Estimate required liquidity from limit value (0 for market orders → any liquidity passes)
			requiredAmount := float64(body.Quantity) * body.LimitValue

			vResp, vErr := fundClient.ValidateFundAccount(ctx, &pb_fund.ValidateFundAccountRequest{
				FundId:         body.FundId,
				ManagerId:      userID,
				RequiredAmount: requiredAmount,
			})
			if vErr != nil {
				switch status.Code(vErr) {
				case codes.NotFound:
					c.JSON(http.StatusNotFound, gin.H{"error": "fund not found"})
				case codes.PermissionDenied:
					c.JSON(http.StatusForbidden, gin.H{"error": "you are not the manager of this fund"})
				default:
					c.JSON(http.StatusInternalServerError, gin.H{"error": vErr.Error()})
				}
				return
			}
			if !vResp.IsLiquid {
				c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient fund liquidity"})
				return
			}
			body.AccountId = vResp.AccountId
			fundID = body.FundId
		}
		// Non-supervisors providing purchaseFor=FUND: field is silently ignored, standard flow applies.

		resp, err := orderClient.CreateOrder(ctx, &pb.CreateOrderRequest{
			UserId:     userID,
			UserType:   userType,
			AssetId:    body.AssetId,
			Quantity:   body.Quantity,
			LimitValue: body.LimitValue,
			StopValue:  body.StopValue,
			IsAon:      body.IsAon,
			IsMargin:   body.IsMargin,
			AccountId:  body.AccountId,
			Direction:  body.Direction,
			FundId:     fundID,
		})
		if err != nil {
			orderError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"orderId":          resp.OrderId,
			"orderType":        resp.OrderType,
			"status":           resp.Status,
			"approximatePrice": resp.ApproximatePrice,
		})
	}
}

// ListOrders handles GET /orders
func ListOrders(orderClient pb.OrderServiceClient, employeeClient pb_emp.EmployeeServiceClient, securitiesClient pb_sec.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		statusFilter := c.DefaultQuery("status", "ALL")
		agentID, _ := strconv.ParseInt(c.Query("agentId"), 10, 64)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := orderClient.ListOrders(ctx, &pb.ListOrdersRequest{
			Status:  statusFilter,
			AgentId: agentID,
		})
		if err != nil {
			orderError(c, err)
			return
		}

		// Collect unique user IDs to look up agent names
		seen := make(map[int64]bool)
		for _, o := range resp.Orders {
			if o.UserId != 0 {
				seen[o.UserId] = true
			}
		}
		names := make(map[int64]string, len(seen))
		for uid := range seen {
			empCtx, empCancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
			empResp, empErr := employeeClient.GetEmployeeById(empCtx, &pb_emp.GetEmployeeByIdRequest{Id: uid})
			empCancel()
			if empErr == nil && empResp.Employee != nil {
				names[uid] = fmt.Sprintf("%s %s", empResp.Employee.FirstName, empResp.Employee.LastName)
			}
		}

		// Collect unique asset IDs to look up tickers
		seenAssets := make(map[int64]bool)
		for _, o := range resp.Orders {
			if o.AssetId != 0 {
				seenAssets[o.AssetId] = true
			}
		}
		tickers := make(map[int64]string, len(seenAssets))
		for aid := range seenAssets {
			secCtx, secCancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
			secResp, secErr := securitiesClient.GetListingById(secCtx, &pb_sec.GetListingByIdRequest{Id: aid})
			secCancel()
			if secErr == nil && secResp.Summary != nil {
				tickers[aid] = secResp.Summary.Ticker
			}
		}

		enriched := make([]orderResponse, 0, len(resp.Orders))
		for _, o := range resp.Orders {
			enriched = append(enriched, orderResponse{
				Id:                o.Id,
				UserId:            o.UserId,
				AgentName:         names[o.UserId],
				AssetId:           o.AssetId,
				AssetTicker:       tickers[o.AssetId],
				OrderType:         o.OrderType,
				Quantity:          o.Quantity,
				ContractSize:      o.ContractSize,
				PricePerUnit:      o.PricePerUnit,
				LimitValue:        o.LimitValue,
				StopValue:         o.StopValue,
				Direction:         o.Direction,
				Status:            o.Status,
				ApprovedBy:        o.ApprovedBy,
				IsDone:            o.IsDone,
				LastModification:  o.LastModification,
				RemainingPortions: o.RemainingPortions,
				AfterHours:        o.AfterHours,
				IsAon:             o.IsAon,
				IsMargin:          o.IsMargin,
				AccountId:         o.AccountId,
				CommissionPaid:    o.CommissionPaid,
				Ticker:            o.Ticker,
				AssetType:         o.AssetType,
			})
		}

		c.JSON(http.StatusOK, gin.H{"orders": enriched})
	}
}

// GetMyOrders handles GET /orders/my (employee) and GET /client/orders (client).
// userType must be "EMPLOYEE" or "CLIENT".
func GetMyOrders(orderClient pb.OrderServiceClient, userType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 32)
		pageSize, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 32)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := orderClient.ListOrders(ctx, &pb.ListOrdersRequest{
			MyOrders:   true,
			CallerId:   callerID,
			CallerType: userType,
			Status:     c.DefaultQuery("status", ""),
			AssetType:  c.Query("asset_type"),
			Direction:  c.Query("direction"),
			FromDate:   c.Query("from_date"),
			ToDate:     c.Query("to_date"),
			Page:       int32(page),
			PageSize:   int32(pageSize),
		})
		if err != nil {
			orderError(c, err)
			return
		}

		enriched := make([]orderResponse, 0, len(resp.Orders))
		for _, o := range resp.Orders {
			enriched = append(enriched, orderResponse{
				Id:                o.Id,
				UserId:            o.UserId,
				AssetId:           o.AssetId,
				AssetTicker:       o.Ticker,
				OrderType:         o.OrderType,
				Quantity:          o.Quantity,
				ContractSize:      o.ContractSize,
				PricePerUnit:      o.PricePerUnit,
				LimitValue:        o.LimitValue,
				StopValue:         o.StopValue,
				Direction:         o.Direction,
				Status:            o.Status,
				ApprovedBy:        o.ApprovedBy,
				IsDone:            o.IsDone,
				LastModification:  o.LastModification,
				RemainingPortions: o.RemainingPortions,
				AfterHours:        o.AfterHours,
				IsAon:             o.IsAon,
				IsMargin:          o.IsMargin,
				AccountId:         o.AccountId,
				CommissionPaid:    o.CommissionPaid,
				Ticker:            o.Ticker,
				AssetType:         o.AssetType,
			})
		}

		c.JSON(http.StatusOK, gin.H{"orders": enriched})
	}
}

// GetOrderById handles GET /orders/:id
func GetOrderById(orderClient pb.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := orderClient.GetOrderById(ctx, &pb.GetOrderByIdRequest{Id: id})
		if err != nil {
			orderError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"order": resp.Order})
	}
}

// ApproveOrder handles PUT /orders/:id/approve
func ApproveOrder(orderClient pb.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
			return
		}

		supervisorID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err = orderClient.ApproveOrder(ctx, &pb.ApproveOrderRequest{
			OrderId:      orderID,
			SupervisorId: supervisorID,
		})
		if err != nil {
			orderError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "order approved"})
	}
}

// DeclineOrder handles PUT /orders/:id/decline
func DeclineOrder(orderClient pb.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
			return
		}

		supervisorID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err = orderClient.DeclineOrder(ctx, &pb.DeclineOrderRequest{
			OrderId:      orderID,
			SupervisorId: supervisorID,
		})
		if err != nil {
			orderError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "order declined"})
	}
}

// CancelOrder handles DELETE /orders/:id
func CancelOrder(orderClient pb.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
			return
		}

		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err = orderClient.CancelOrder(ctx, &pb.CancelOrderRequest{
			OrderId: orderID,
			UserId:  userID,
		})
		if err != nil {
			orderError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
	}
}

// CancelOrderPortions handles DELETE /orders/:id/portions
func CancelOrderPortions(orderClient pb.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
			return
		}

		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err = orderClient.CancelOrderPortions(ctx, &pb.CancelOrderPortionsRequest{
			OrderId: orderID,
			UserId:  userID,
		})
		if err != nil {
			orderError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "remaining portions cancelled"})
	}
}
