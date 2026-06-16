package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func recurringError(c *gin.Context, err error) {
	switch status.Code(err) {
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
	case codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func CreateRecurringOrder(orderClient pb.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			AssetID   int64   `json:"asset_id" binding:"required"`
			Direction string  `json:"direction" binding:"required"`
			Mode      string  `json:"mode" binding:"required"`
			Value     float64 `json:"value" binding:"required"`
			AccountID int64   `json:"account_id" binding:"required"`
			Cadence   string  `json:"cadence" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
			return
		}
		userType := "CLIENT"
		if role := middleware.GetCallerRoleFromToken(c); role == "EMPLOYEE" {
			userType = "EMPLOYEE"
		}
		resp, err := orderClient.CreateRecurringOrder(context.Background(), &pb.CreateRecurringOrderRequest{
			UserId:    userID,
			UserType:  userType,
			AssetId:   body.AssetID,
			Direction: body.Direction,
			Mode:      body.Mode,
			Value:     body.Value,
			AccountId: body.AccountID,
			Cadence:   body.Cadence,
		})
		if err != nil {
			recurringError(c, err)
			return
		}
		c.JSON(http.StatusCreated, resp.Order)
	}
}

func ListRecurringOrders(orderClient pb.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
			return
		}
		userType := "CLIENT"
		if role := middleware.GetCallerRoleFromToken(c); role == "EMPLOYEE" {
			userType = "EMPLOYEE"
		}
		resp, err := orderClient.ListRecurringOrders(context.Background(), &pb.ListRecurringOrdersRequest{
			UserId:   userID,
			UserType: userType,
		})
		if err != nil {
			recurringError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp.Orders)
	}
}

func PauseRecurringOrder(orderClient pb.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		userID, _ := middleware.GetUserIDFromToken(c)
		userType := "CLIENT"
		if role := middleware.GetCallerRoleFromToken(c); role == "EMPLOYEE" {
			userType = "EMPLOYEE"
		}
		_, err = orderClient.PauseRecurringOrder(context.Background(), &pb.RecurringOrderIdRequest{
			Id:       id,
			UserId:   userID,
			UserType: userType,
		})
		if err != nil {
			recurringError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func ResumeRecurringOrder(orderClient pb.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		userID, _ := middleware.GetUserIDFromToken(c)
		userType := "CLIENT"
		if role := middleware.GetCallerRoleFromToken(c); role == "EMPLOYEE" {
			userType = "EMPLOYEE"
		}
		_, err = orderClient.ResumeRecurringOrder(context.Background(), &pb.RecurringOrderIdRequest{
			Id:       id,
			UserId:   userID,
			UserType: userType,
		})
		if err != nil {
			recurringError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func CancelRecurringOrder(orderClient pb.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		userID, _ := middleware.GetUserIDFromToken(c)
		userType := "CLIENT"
		if role := middleware.GetCallerRoleFromToken(c); role == "EMPLOYEE" {
			userType = "EMPLOYEE"
		}
		_, err = orderClient.CancelRecurringOrder(context.Background(), &pb.RecurringOrderIdRequest{
			Id:       id,
			UserId:   userID,
			UserType: userType,
		})
		if err != nil {
			recurringError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
