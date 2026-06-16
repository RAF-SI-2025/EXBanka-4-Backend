package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func watchlistError(c *gin.Context, err error) {
	switch status.Code(err) {
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
	case codes.PermissionDenied:
		c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
	case codes.AlreadyExists:
		c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// CreateWatchlist handles POST /watchlists
func CreateWatchlist(portfolioClient pb.PortfolioServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := portfolioClient.CreateWatchlist(ctx, &pb.CreateWatchlistRequest{
			UserId: userID, UserType: userType, Name: body.Name,
		})
		if err != nil {
			watchlistError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"watchlist": resp.Watchlist})
	}
}

// ListWatchlists handles GET /watchlists
func ListWatchlists(portfolioClient pb.PortfolioServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := portfolioClient.ListWatchlists(ctx, &pb.ListWatchlistsRequest{
			UserId: userID, UserType: userType,
		})
		if err != nil {
			watchlistError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"watchlists": resp.Watchlists})
	}
}

// DeleteWatchlist handles DELETE /watchlists/:id
func DeleteWatchlist(portfolioClient pb.PortfolioServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid watchlist id"})
			return
		}
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err = portfolioClient.DeleteWatchlist(ctx, &pb.DeleteWatchlistRequest{
			WatchlistId: id, UserId: userID, UserType: userType,
		})
		if err != nil {
			watchlistError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "watchlist deleted"})
	}
}

// AddWatchlistItem handles POST /watchlists/:id/items
func AddWatchlistItem(portfolioClient pb.PortfolioServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		watchlistID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid watchlist id"})
			return
		}
		var body struct {
			ListingId int64 `json:"listing_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := portfolioClient.AddWatchlistItem(ctx, &pb.AddWatchlistItemRequest{
			WatchlistId: watchlistID, ListingId: body.ListingId, UserId: userID, UserType: userType,
		})
		if err != nil {
			watchlistError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"item": resp.Item})
	}
}

// RemoveWatchlistItem handles DELETE /watchlists/:id/items/:listing_id
func RemoveWatchlistItem(portfolioClient pb.PortfolioServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		watchlistID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid watchlist id"})
			return
		}
		listingID, err := strconv.ParseInt(c.Param("listing_id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid listing id"})
			return
		}
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err = portfolioClient.RemoveWatchlistItem(ctx, &pb.RemoveWatchlistItemRequest{
			WatchlistId: watchlistID, ListingId: listingID, UserId: userID, UserType: userType,
		})
		if err != nil {
			watchlistError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "item removed"})
	}
}

// GetWatchlistItems handles GET /watchlists/:id/items
func GetWatchlistItems(portfolioClient pb.PortfolioServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		watchlistID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid watchlist id"})
			return
		}
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := portfolioClient.GetWatchlistItems(ctx, &pb.GetWatchlistItemsRequest{
			WatchlistId: watchlistID, UserId: userID, UserType: userType,
		})
		if err != nil {
			watchlistError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": resp.Items})
	}
}

// GetQuickWatchlist handles GET /watchlists/quick — returns items from user's first watchlist
func GetQuickWatchlist(portfolioClient pb.PortfolioServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		listResp, err := portfolioClient.ListWatchlists(ctx, &pb.ListWatchlistsRequest{
			UserId: userID, UserType: userType,
		})
		if err != nil {
			watchlistError(c, err)
			return
		}
		if len(listResp.Watchlists) == 0 {
			c.JSON(http.StatusOK, gin.H{"items": []interface{}{}})
			return
		}

		firstID := listResp.Watchlists[0].Id
		itemsResp, err := portfolioClient.GetWatchlistItems(ctx, &pb.GetWatchlistItemsRequest{
			WatchlistId: firstID, UserId: userID, UserType: userType,
		})
		if err != nil {
			watchlistError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": itemsResp.Items})
	}
}
