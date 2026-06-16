package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	"github.com/gin-gonic/gin"
)

func ListNotifications(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		unreadOnly := c.Query("unread_only") == "true"
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := client.ListNotifications(ctx, &pb.ListNotificationsRequest{
			UserId:     userID,
			UserType:   userType,
			UnreadOnly: unreadOnly,
			Page:       int32(page),
			PageSize:   int32(pageSize),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch notifications"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"notifications": resp.Notifications, "total": resp.Total})
	}
}

func GetUnreadCount(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := client.GetUnreadCount(ctx, &pb.GetUnreadCountRequest{
			UserId:   userID,
			UserType: userType,
		})
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"count": int64(0)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"count": resp.Count})
	}
}

func MarkNotificationRead(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		_, err = client.MarkNotificationRead(ctx, &pb.MarkNotificationReadRequest{Id: id, UserId: userID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark as read"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

func MarkAllNotificationsRead(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		_, err = client.MarkAllRead(ctx, &pb.MarkAllReadRequest{UserId: userID, UserType: userType})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark all as read"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}
