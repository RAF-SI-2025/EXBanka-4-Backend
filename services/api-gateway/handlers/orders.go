package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

// CreateOrder handles POST /orders
func CreateOrder(orderClient pb.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			AssetId    int64   `json:"assetId"    binding:"required"`
			Quantity   int32   `json:"quantity"   binding:"required"`
			LimitValue float64 `json:"limitValue"`
			StopValue  float64 `json:"stopValue"`
			IsAon      bool    `json:"isAon"`
			IsMargin   bool    `json:"isMargin"`
			AccountId  int64   `json:"accountId"  binding:"required"`
			Direction  string  `json:"direction"  binding:"required"`
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
func ListOrders(orderClient pb.OrderServiceClient) gin.HandlerFunc {
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

		c.JSON(http.StatusOK, gin.H{"orders": resp.Orders})
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
