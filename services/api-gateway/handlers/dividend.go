package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	"github.com/gin-gonic/gin"
)

func GetDividendHistory(portfolioClient pb.PortfolioServiceClient, userType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
			return
		}

		var listingID int64
		if lid := c.Param("listing_id"); lid != "" {
			listingID, _ = strconv.ParseInt(lid, 10, 64)
		}

		page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 32)
		pageSize, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 32)

		resp, err := portfolioClient.GetDividendHistory(context.Background(), &pb.GetDividendHistoryRequest{
			UserId:    userID,
			UserType:  userType,
			ListingId: listingID,
			FromDate:  c.Query("from_date"),
			ToDate:    c.Query("to_date"),
			Page:      int32(page),
			PageSize:  int32(pageSize),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp.Payouts)
	}
}
