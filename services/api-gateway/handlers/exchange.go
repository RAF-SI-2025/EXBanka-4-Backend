package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/exchange"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetExchangeRates godoc
// @Summary      Get today's exchange rates
// @Tags         exchange
// @Produce      json
// @Success      200  {array}   map[string]interface{}
// @Router       /exchange/rates [get]
func GetExchangeRates(client pb.ExchangeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetExchangeRates(ctx, &pb.GetExchangeRatesRequest{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch exchange rates"})
			return
		}

		type rateJSON struct {
			CurrencyCode string  `json:"currencyCode"`
			BuyingRate   float64 `json:"buyingRate"`
			SellingRate  float64 `json:"sellingRate"`
			MiddleRate   float64 `json:"middleRate"`
			Date         string  `json:"date"`
		}
		result := make([]rateJSON, 0, len(resp.Rates))
		for _, r := range resp.Rates {
			result = append(result, rateJSON{
				CurrencyCode: r.CurrencyCode,
				BuyingRate:   r.BuyingRate,
				SellingRate:  r.SellingRate,
				MiddleRate:   r.MiddleRate,
				Date:         r.Date,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

// GetExchangeRate godoc
// @Summary      Get exchange rate between two currencies
// @Tags         exchange
// @Produce      json
// @Param        from  query  string  true  "From currency (e.g. EUR)"
// @Param        to    query  string  true  "To currency (e.g. USD)"
// @Success      200   {object}  map[string]interface{}
// @Router       /exchange/rate [get]
func GetExchangeRate(client pb.ExchangeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		from := strings.ToUpper(c.Query("from"))
		to := strings.ToUpper(c.Query("to"))
		if from == "" || to == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from and to query params are required"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetExchangeRates(ctx, &pb.GetExchangeRatesRequest{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch rates"})
			return
		}

		rates := make(map[string]*pb.ExchangeRate)
		for _, r := range resp.Rates {
			rates[r.CurrencyCode] = r
		}

		// If either is RSD, return the other currency's rate directly
		if from == "RSD" {
			r, ok := rates[to]
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "rate not found for " + to})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"from":        from,
				"to":          to,
				"sellingRate": 1.0 / r.SellingRate,
				"buyingRate":  1.0 / r.BuyingRate,
				"middleRate":  1.0 / r.MiddleRate,
			})
			return
		}
		if to == "RSD" {
			r, ok := rates[from]
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "rate not found for " + from})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"from":        from,
				"to":          to,
				"sellingRate": r.SellingRate,
				"buyingRate":  r.BuyingRate,
				"middleRate":  r.MiddleRate,
			})
			return
		}

		rFrom, okFrom := rates[from]
		rTo, okTo := rates[to]
		if !okFrom || !okTo {
			c.JSON(http.StatusNotFound, gin.H{"error": "rate not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"from":        from,
			"to":          to,
			"sellingRate": rFrom.SellingRate / rTo.SellingRate,
			"middleRate":  rFrom.MiddleRate / rTo.MiddleRate,
		})
	}
}

// ConvertAmount godoc
// @Summary      Convert currency between two accounts
// @Tags         exchange
// @Accept       json
// @Produce      json
// @Param        body  body  object  true  "Conversion request"
// @Success      200   {object}  map[string]interface{}
// @Router       /exchange/convert [post]
func ConvertAmount(client pb.ExchangeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			FromAccount string  `json:"fromAccount" binding:"required"`
			ToAccount   string  `json:"toAccount"   binding:"required"`
			Amount      float64 `json:"amount"      binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		clientID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.ConvertAmount(ctx, &pb.ConvertAmountRequest{
			ClientId:    clientID,
			FromAccount: req.FromAccount,
			ToAccount:   req.ToAccount,
			Amount:      req.Amount,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.PermissionDenied:
				c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
			case codes.FailedPrecondition:
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": status.Convert(err).Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": status.Convert(err).Message()})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"fromCurrency":  resp.FromCurrency,
			"toCurrency":    resp.ToCurrency,
			"fromAmount":    resp.FromAmount,
			"toAmount":      resp.ToAmount,
			"rate":          resp.Rate,
			"commission":    resp.Commission,
			"transactionId": resp.TransactionId,
		})
	}
}

// GetExchangeHistory godoc
// @Summary      Get client's exchange transaction history
// @Tags         exchange
// @Produce      json
// @Success      200  {array}  map[string]interface{}
// @Router       /exchange/history [get]
func GetExchangeHistory(client pb.ExchangeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := client.GetExchangeHistory(ctx, &pb.GetExchangeHistoryRequest{ClientId: clientID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch exchange history"})
			return
		}

		type txJSON struct {
			ID           int64   `json:"id"`
			FromAccount  string  `json:"fromAccount"`
			ToAccount    string  `json:"toAccount"`
			FromCurrency string  `json:"fromCurrency"`
			ToCurrency   string  `json:"toCurrency"`
			FromAmount   float64 `json:"fromAmount"`
			ToAmount     float64 `json:"toAmount"`
			Rate         float64 `json:"rate"`
			Commission   float64 `json:"commission"`
			Timestamp    string  `json:"timestamp"`
			Status       string  `json:"status"`
		}
		result := make([]txJSON, 0, len(resp.Transactions))
		for _, t := range resp.Transactions {
			result = append(result, txJSON{
				ID:           t.Id,
				FromAccount:  t.FromAccount,
				ToAccount:    t.ToAccount,
				FromCurrency: t.FromCurrency,
				ToCurrency:   t.ToCurrency,
				FromAmount:   t.FromAmount,
				ToAmount:     t.ToAmount,
				Rate:         t.Rate,
				Commission:   t.Commission,
				Timestamp:    t.Timestamp,
				Status:       t.Status,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}
