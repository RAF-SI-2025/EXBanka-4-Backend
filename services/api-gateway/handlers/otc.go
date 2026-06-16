package handlers

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func negotiationToJSON(n *pb.NegotiationResponse) gin.H {
	return gin.H{
		"id":             n.Id,
		"ticker":         n.Ticker,
		"sellerId":       n.SellerId,
		"sellerType":     n.SellerType,
		"sellerName":     n.SellerName,
		"buyerId":        n.BuyerId,
		"buyerType":      n.BuyerType,
		"buyerName":      n.BuyerName,
		"amount":         n.Amount,
		"pricePerStock":  n.PricePerStock,
		"settlementDate": n.SettlementDate,
		"premium":        n.Premium,
		"currency":       n.Currency,
		"lastModified":   n.LastModified,
		"modifiedById":   n.ModifiedById,
		"modifiedByType": n.ModifiedByType,
		"modifiedByName": n.ModifiedByName,
		"status":         n.Status,
	}
}

func mapOtcError(c *gin.Context, err error) {
	switch status.Code(err) {
	case codes.PermissionDenied:
		c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
	case codes.AlreadyExists:
		c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
	case codes.FailedPrecondition:
		c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
	case codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": status.Convert(err).Message()})
	}
}

// CreateNegotiation godoc
// @Summary      Create an OTC negotiation
// @Tags         otc
// @Accept       json
// @Produce      json
// @Success      201  {object}  map[string]interface{}
// @Router       /otc/negotiations [post]
func CreateNegotiation(client pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		callerType := middleware.GetCallerRoleFromToken(c)

		var req struct {
			SellerId            *int64  `json:"sellerId"            binding:"required"`
			SellerType          string  `json:"sellerType"          binding:"required"`
			SellerRoutingNumber int32   `json:"sellerRoutingNumber"`
			SellerExternalId    string  `json:"sellerExternalId"`
			Ticker              string  `json:"ticker"              binding:"required"`
			Amount              int32   `json:"amount"              binding:"required"`
			PricePerStock       float64 `json:"pricePerStock"       binding:"required"`
			SettlementDate      string  `json:"settlementDate"      binding:"required"`
			Premium             float64 `json:"premium"`
			Currency            string  `json:"currency"            binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.CreateNegotiation(ctx, &pb.CreateNegotiationRequest{
			BuyerId:             callerID,
			BuyerType:           callerType,
			SellerId:            *req.SellerId,
			SellerType:          req.SellerType,
			SellerRoutingNumber: req.SellerRoutingNumber,
			SellerExternalId:    req.SellerExternalId,
			Ticker:              req.Ticker,
			Amount:              req.Amount,
			PricePerStock:       req.PricePerStock,
			SettlementDate:      req.SettlementDate,
			Premium:             req.Premium,
			Currency:            req.Currency,
		})
		if err != nil {
			mapOtcError(c, err)
			return
		}

		c.JSON(http.StatusCreated, negotiationToJSON(resp))
	}
}

// ListNegotiations godoc
// @Summary      List OTC negotiations for the caller
// @Tags         otc
// @Produce      json
// @Success      200  {array}  map[string]interface{}
// @Router       /otc/negotiations [get]
func ListNegotiations(client pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		callerType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.ListNegotiations(ctx, &pb.ListNegotiationsRequest{
			CallerId:   callerID,
			CallerType: callerType,
		})
		if err != nil {
			mapOtcError(c, err)
			return
		}

		result := make([]gin.H, 0, len(resp.Negotiations))
		for _, n := range resp.Negotiations {
			result = append(result, negotiationToJSON(n))
		}
		c.JSON(http.StatusOK, result)
	}
}

// GetNegotiation godoc
// @Summary      Get a single OTC negotiation
// @Tags         otc
// @Produce      json
// @Param        id  path  int  true  "Negotiation ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /otc/negotiations/{id} [get]
func GetNegotiation(client pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		negID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid negotiation id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetNegotiation(ctx, &pb.GetNegotiationRequest{NegotiationId: negID})
		if err != nil {
			mapOtcError(c, err)
			return
		}
		c.JSON(http.StatusOK, negotiationToJSON(resp))
	}
}

// CounterOffer godoc
// @Summary      Submit a counter-offer on an OTC negotiation
// @Tags         otc
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "Negotiation ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /otc/negotiations/{id}/counter [put]
func CounterOffer(client pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		callerType := middleware.GetCallerRoleFromToken(c)

		negID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid negotiation id"})
			return
		}

		var req struct {
			Amount         int32   `json:"amount"         binding:"required"`
			PricePerStock  float64 `json:"pricePerStock"  binding:"required"`
			SettlementDate string  `json:"settlementDate" binding:"required"`
			Premium        float64 `json:"premium"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.CounterOffer(ctx, &pb.CounterOfferRequest{
			NegotiationId:  negID,
			CallerId:       callerID,
			CallerType:     callerType,
			Amount:         req.Amount,
			PricePerStock:  req.PricePerStock,
			SettlementDate: req.SettlementDate,
			Premium:        req.Premium,
		})
		if err != nil {
			mapOtcError(c, err)
			return
		}
		c.JSON(http.StatusOK, negotiationToJSON(resp))
	}
}

// AcceptNegotiation godoc
// @Summary      Accept an OTC negotiation
// @Tags         otc
// @Produce      json
// @Param        id  path  int  true  "Negotiation ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /otc/negotiations/{id}/accept [put]
func AcceptNegotiation(client pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		callerType := middleware.GetCallerRoleFromToken(c)

		negID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid negotiation id"})
			return
		}

		var body struct {
			BuyerAccountId int64 `json:"buyerAccountId"`
		}
		_ = c.ShouldBindJSON(&body)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.AcceptNegotiation(ctx, &pb.AcceptNegotiationRequest{
			NegotiationId:  negID,
			CallerId:       callerID,
			CallerType:     callerType,
			BuyerAccountId: body.BuyerAccountId,
		})
		if err != nil {
			mapOtcError(c, err)
			return
		}
		c.JSON(http.StatusOK, negotiationToJSON(resp))
	}
}

// RejectNegotiation godoc
// @Summary      Reject an OTC negotiation
// @Tags         otc
// @Produce      json
// @Param        id  path  int  true  "Negotiation ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /otc/negotiations/{id}/reject [put]
func RejectNegotiation(client pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		callerType := middleware.GetCallerRoleFromToken(c)

		negID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid negotiation id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.RejectNegotiation(ctx, &pb.RejectNegotiationRequest{
			NegotiationId: negID,
			CallerId:      callerID,
			CallerType:    callerType,
		})
		if err != nil {
			mapOtcError(c, err)
			return
		}
		c.JSON(http.StatusOK, negotiationToJSON(resp))
	}
}

func contractToJSON(ct *pb.ContractResponse) gin.H {
	return gin.H{
		"id":             ct.Id,
		"negotiationId":  ct.NegotiationId,
		"sellerId":       ct.SellerId,
		"sellerType":     ct.SellerType,
		"sellerName":     ct.SellerName,
		"buyerId":        ct.BuyerId,
		"buyerType":      ct.BuyerType,
		"buyerName":      ct.BuyerName,
		"ticker":         ct.Ticker,
		"amount":         ct.Amount,
		"strikePrice":    ct.StrikePrice,
		"premium":        ct.Premium,
		"currency":       ct.Currency,
		"settlementDate": ct.SettlementDate,
		"status":         ct.Status,
		"createdAt":      ct.CreatedAt,
		"profit":         ct.Profit,
	}
}

func ListContracts(client pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		callerType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.ListContracts(ctx, &pb.ListContractsRequest{
			CallerId:     callerID,
			CallerType:   callerType,
			StatusFilter: c.Query("status"),
		})
		if err != nil {
			mapOtcError(c, err)
			return
		}

		result := make([]gin.H, 0, len(resp.Contracts))
		for _, contract := range resp.Contracts {
			result = append(result, contractToJSON(contract))
		}
		c.JSON(http.StatusOK, result)
	}
}

func ExerciseContract(client pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		callerType := middleware.GetCallerRoleFromToken(c)

		contractID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contract id"})
			return
		}

		var req struct {
			BuyerAccountId int64 `json:"buyerAccountId"`
		}
		_ = c.ShouldBindJSON(&req)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		// Forward X-Saga-* fault-injection headers as gRPC metadata (test builds only).
		if os.Getenv("OTC_SAGA_TEST_HOOKS") == "true" {
			md := metadata.New(nil)
			for _, h := range []string{
				"X-Saga-Force-Fail", "X-Saga-Force-Fail-Kind",
				"X-Saga-Compensate-Fail", "X-Saga-Compensate-Fail-Times",
				"X-Saga-Inject-Delay",
			} {
				if v := c.GetHeader(h); v != "" {
					md.Set(strings.ToLower(h), v)
				}
			}
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		resp, err := client.ExerciseContract(ctx, &pb.ExerciseContractRequest{
			ContractId:     contractID,
			CallerId:       callerID,
			CallerType:     callerType,
			BuyerAccountId: req.BuyerAccountId,
		})
		if err != nil {
			mapOtcError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":     resp.Status,
			"executedAt": resp.ExecutedAt,
			"sagaId":     resp.SagaId,
		})
	}
}

func GetMarket(client pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		callerType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetMarket(ctx, &pb.GetMarketRequest{
			CallerId:   callerID,
			CallerType: callerType,
		})
		if err != nil {
			mapOtcError(c, err)
			return
		}

		result := make([]gin.H, 0, len(resp.Items))
		for _, item := range resp.Items {
			result = append(result, gin.H{
				"ticker":        item.Ticker,
				"name":          item.Name,
				"amount":        item.Amount,
				"pricePerStock": item.PricePerStock,
				"currency":      item.Currency,
				"lastUpdated":   item.LastUpdated,
				"ownerName":     item.OwnerName,
				"ownerBank":     item.OwnerBank,
				"ownerId":       item.OwnerId,
				"ownerType":     item.OwnerType,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

func SetPublicMode(portfolioClient pb_portfolio.PortfolioServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		callerType := middleware.GetCallerRoleFromToken(c)

		ticker := c.Param("ticker")

		var body struct {
			IsPublic bool `json:"isPublic"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := portfolioClient.SetPublicMode(ctx, &pb_portfolio.SetPublicModeRequest{
			UserId:   callerID,
			UserType: callerType,
			Ticker:   ticker,
			IsPublic: body.IsPublic,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": status.Convert(err).Message()})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"ticker":   resp.Ticker,
			"isPublic": resp.IsPublic,
		})
	}
}

func GetPublicStock(client pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validateOtcInterbankKey(c) {
			c.Status(http.StatusUnauthorized)
			return
		}

		routingNumber, _ := strconv.Atoi(os.Getenv("OWN_ROUTING_NUMBER"))

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetMarket(ctx, &pb.GetMarketRequest{
			CallerId:   0,
			CallerType: "CLIENT",
		})
		if err != nil {
			mapOtcError(c, err)
			return
		}

		type seller struct {
			RoutingNumber int    `json:"routingNumber"`
			ID            string `json:"id"`
		}
		type sellerEntry struct {
			Seller seller `json:"seller"`
			Amount int32  `json:"amount"`
		}
		type stock struct {
			Ticker   string  `json:"ticker"`
			Name     string  `json:"name"`
			Price    float64 `json:"price"`
			Currency string  `json:"currency"`
		}
		type stockEntry struct {
			Stock   stock         `json:"stock"`
			Sellers []sellerEntry `json:"sellers"`
		}

		byTicker := make(map[string]*stockEntry)
		for _, item := range resp.Items {
			entry, ok := byTicker[item.Ticker]
			if !ok {
				entry = &stockEntry{Stock: stock{
					Ticker:   item.Ticker,
					Name:     item.Name,
					Price:    item.PricePerStock,
					Currency: item.Currency,
				}}
				byTicker[item.Ticker] = entry
			}
			entry.Sellers = append(entry.Sellers, sellerEntry{
				Seller: seller{
					RoutingNumber: routingNumber,
					ID:            strconv.FormatInt(item.OwnerId, 10),
				},
				Amount: item.Amount,
			})
		}

		result := make([]*stockEntry, 0, len(byTicker))
		for _, v := range byTicker {
			result = append(result, v)
		}
		c.JSON(http.StatusOK, result)
	}
}

func historyEntryToJSON(e *pb.NegotiationHistoryEntry) gin.H {
	return gin.H{
		"id":                e.Id,
		"action":            e.Action,
		"actorId":           e.ActorId,
		"actorType":         e.ActorType,
		"actorName":         e.ActorName,
		"oldAmount":         e.OldAmount,
		"newAmount":         e.NewAmount,
		"oldPricePerStock":  e.OldPricePerStock,
		"newPricePerStock":  e.NewPricePerStock,
		"oldSettlementDate": e.OldSettlementDate,
		"newSettlementDate": e.NewSettlementDate,
		"oldPremium":        e.OldPremium,
		"newPremium":        e.NewPremium,
		"timestamp":         e.Timestamp,
	}
}

func GetNegotiationHistory(client pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		callerType := middleware.GetCallerRoleFromToken(c)

		negID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid negotiation id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetNegotiationHistory(ctx, &pb.GetNegotiationHistoryRequest{
			NegotiationId: negID,
			CallerId:      callerID,
			CallerType:    callerType,
		})
		if err != nil {
			mapOtcError(c, err)
			return
		}

		entries := make([]gin.H, len(resp.Entries))
		for i, e := range resp.Entries {
			entries[i] = historyEntryToJSON(e)
		}
		c.JSON(http.StatusOK, entries)
	}
}
