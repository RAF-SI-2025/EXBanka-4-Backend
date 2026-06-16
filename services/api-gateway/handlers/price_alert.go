package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func priceAlertError(c *gin.Context, err error) {
	switch status.Code(err) {
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
	case codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func CreatePriceAlert(securitiesClient pb.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			ListingID        int64   `json:"listing_id" binding:"required"`
			Condition        string  `json:"condition" binding:"required"`
			Threshold        float64 `json:"threshold" binding:"required"`
			NotificationType string  `json:"notification_type"`
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
		resp, err := securitiesClient.CreatePriceAlert(context.Background(), &pb.CreatePriceAlertRequest{
			UserId:           userID,
			UserType:         userType,
			ListingId:        body.ListingID,
			Condition:        body.Condition,
			Threshold:        body.Threshold,
			NotificationType: body.NotificationType,
		})
		if err != nil {
			priceAlertError(c, err)
			return
		}
		c.JSON(http.StatusCreated, resp.Alert)
	}
}

func ListPriceAlerts(securitiesClient pb.SecuritiesServiceClient) gin.HandlerFunc {
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
		resp, err := securitiesClient.ListPriceAlerts(context.Background(), &pb.ListPriceAlertsRequest{
			UserId:   userID,
			UserType: userType,
		})
		if err != nil {
			priceAlertError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp.Alerts)
	}
}

func DeletePriceAlert(securitiesClient pb.SecuritiesServiceClient) gin.HandlerFunc {
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
		_, err = securitiesClient.DeletePriceAlert(context.Background(), &pb.DeletePriceAlertRequest{
			Id:       id,
			UserId:   userID,
			UserType: userType,
		})
		if err != nil {
			priceAlertError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
