package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	gwinterbank "github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/interbank"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func gatewayOwnRoutingInt() int32 {
	var n int32
	fmt.Sscanf(os.Getenv("OWN_ROUTING_NUMBER"), "%d", &n)
	return n
}

// validateOtcInterbankKey rejects requests whose X-Api-Key does not match OWN_INTERBANK_API_KEY.
func validateOtcInterbankKey(c *gin.Context) bool {
	key := os.Getenv("OWN_INTERBANK_API_KEY")
	return key != "" && c.GetHeader("X-Api-Key") == key
}

// ── JSON shapes expected/returned by the /otc/interbank/* endpoints ──────────

type otcPartyID struct {
	RoutingNumber int32  `json:"routingNumber"`
	ID            string `json:"id"`
}

type otcMoneyAmount struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

type otcNegotiationBody struct {
	Stock struct {
		Ticker string `json:"ticker"`
	} `json:"stock"`
	SettlementDate string         `json:"settlementDate"`
	PricePerUnit   otcMoneyAmount `json:"pricePerUnit"`
	Premium        otcMoneyAmount `json:"premium"`
	BuyerID        otcPartyID     `json:"buyerId"`
	SellerID       otcPartyID     `json:"sellerId"`
	Amount         int32          `json:"amount"`
	SellerType     string         `json:"sellerType"`
}

type otcNegotiationResponse struct {
	Stock struct {
		Ticker string `json:"ticker"`
	} `json:"stock"`
	SettlementDate string         `json:"settlementDate"`
	PricePerUnit   otcMoneyAmount `json:"pricePerUnit"`
	Premium        otcMoneyAmount `json:"premium"`
	BuyerID        otcPartyID     `json:"buyerId"`
	SellerID       otcPartyID     `json:"sellerId"`
	Amount         int32          `json:"amount"`
	IsOngoing      bool           `json:"isOngoing"`
}

func interbankNegToJSON(n *pb.InterbankNegotiationResponse) otcNegotiationResponse {
	resp := otcNegotiationResponse{
		SettlementDate: n.SettlementDate,
		Amount:         n.Amount,
		IsOngoing:      n.IsOngoing,
		PricePerUnit:   otcMoneyAmount{Currency: n.PriceCurrency, Amount: n.PricePerUnit},
		Premium:        otcMoneyAmount{Currency: n.PriceCurrency, Amount: n.Premium},
		BuyerID:        otcPartyID{RoutingNumber: n.BuyerRoutingNumber, ID: n.BuyerExternalId},
		SellerID:       otcPartyID{RoutingNumber: n.SellerRoutingNumber, ID: n.SellerExternalId},
	}
	resp.Stock.Ticker = n.Ticker
	return resp
}

func mapOtcInterbankError(c *gin.Context, err error) {
	switch status.Code(err) {
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
	case codes.FailedPrecondition, codes.AlreadyExists:
		c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
	case codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": status.Convert(err).Message()})
	}
}

// IncomingCreateNegotiation handles POST /otc/interbank/negotiations from a partner bank.
// The partner (buyer) creates a new negotiation for one of our listed stocks (our user = seller).
func IncomingCreateNegotiation(otcClient pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validateOtcInterbankKey(c) {
			c.Status(http.StatusUnauthorized)
			return
		}

		var body otcNegotiationBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := otcClient.CreateInterbankNegotiation(ctx, &pb.CreateInterbankNegotiationRequest{
			Ticker:               body.Stock.Ticker,
			SettlementDate:       body.SettlementDate,
			PricePerUnit:         body.PricePerUnit.Amount,
			PriceCurrency:        body.PricePerUnit.Currency,
			Premium:              body.Premium.Amount,
			BuyerRoutingNumber:   body.BuyerID.RoutingNumber,
			BuyerExternalId:      body.BuyerID.ID,
			SellerRoutingNumber:  body.SellerID.RoutingNumber,
			SellerExternalId:     body.SellerID.ID,
			CreatorRoutingNumber: body.BuyerID.RoutingNumber,
			CreatorExternalId:    body.BuyerID.ID,
			Amount:               body.Amount,
			SellerType:           body.SellerType,
		})
		if err != nil {
			mapOtcInterbankError(c, err)
			return
		}

		// Return the globally-unique negotiation ID (our routing + our local id)
		ownRouting := gatewayOwnRoutingInt()
		c.JSON(http.StatusCreated, gin.H{
			"routingNumber": ownRouting,
			"id":            fmt.Sprintf("%d", resp.LocalId),
		})
	}
}

// IncomingCounterOffer handles PUT /otc/interbank/negotiations/:routingNumber/:id from a partner bank.
func IncomingCounterOffer(otcClient pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validateOtcInterbankKey(c) {
			c.Status(http.StatusUnauthorized)
			return
		}

		routingNumber, extID, ok := parseInterbankNegPath(c)
		if !ok {
			return
		}

		var body struct {
			PricePerUnit   otcMoneyAmount `json:"pricePerUnit"`
			Premium        otcMoneyAmount `json:"premium"`
			Amount         int32          `json:"amount"`
			SettlementDate string         `json:"settlementDate"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := otcClient.InterbankCounterOffer(ctx, &pb.InterbankCounterOfferRequest{
			RoutingNumber:  routingNumber,
			ExternalId:     extID,
			PricePerUnit:   body.PricePerUnit.Amount,
			Premium:        body.Premium.Amount,
			Amount:         body.Amount,
			SettlementDate: body.SettlementDate,
		})
		if err != nil {
			mapOtcInterbankError(c, err)
			return
		}
		c.JSON(http.StatusOK, interbankNegToJSON(resp))
	}
}

// IncomingGetNegotiation handles GET /otc/interbank/negotiations/:routingNumber/:id from a partner bank.
func IncomingGetNegotiation(otcClient pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validateOtcInterbankKey(c) {
			c.Status(http.StatusUnauthorized)
			return
		}

		routingNumber, extID, ok := parseInterbankNegPath(c)
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := otcClient.InterbankGetNegotiation(ctx, &pb.InterbankNegotiationIdRequest{
			RoutingNumber: routingNumber,
			ExternalId:    extID,
		})
		if err != nil {
			mapOtcInterbankError(c, err)
			return
		}
		c.JSON(http.StatusOK, interbankNegToJSON(resp))
	}
}

// IncomingDeleteNegotiation handles DELETE /otc/interbank/negotiations/:routingNumber/:id from a partner bank.
func IncomingDeleteNegotiation(otcClient pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validateOtcInterbankKey(c) {
			c.Status(http.StatusUnauthorized)
			return
		}

		routingNumber, extID, ok := parseInterbankNegPath(c)
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err := otcClient.InterbankDeleteNegotiation(ctx, &pb.InterbankNegotiationIdRequest{
			RoutingNumber: routingNumber,
			ExternalId:    extID,
		})
		if err != nil {
			mapOtcInterbankError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// IncomingAcceptNegotiation handles GET /otc/interbank/negotiations/:routingNumber/:id/accept from a partner bank.
func IncomingAcceptNegotiation(otcClient pb.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validateOtcInterbankKey(c) {
			c.Status(http.StatusUnauthorized)
			return
		}

		routingNumber, extID, ok := parseInterbankNegPath(c)
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err := otcClient.InterbankAcceptNegotiation(ctx, &pb.InterbankNegotiationIdRequest{
			RoutingNumber: routingNumber,
			ExternalId:    extID,
		})
		if err != nil {
			mapOtcInterbankError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// parseInterbankNegPath extracts (routingNumber, externalId) from path params.
func parseInterbankNegPath(c *gin.Context) (int32, string, bool) {
	var routing int32
	if _, err := fmt.Sscanf(c.Param("routingNumber"), "%d", &routing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid routingNumber"})
		return 0, "", false
	}
	extID := c.Param("id")
	if extID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return 0, "", false
	}
	return routing, extID, true
}

// ── Outgoing HTTP calls to partner bank (helper used by existing OTC handlers) ─

const otcInterbankTimeout = 10 * time.Second

// sendOtcRequest POSTs or PUTs body to the partner bank's OTC endpoint.
func sendOtcRequest(method, url, apiKey string, body any) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", apiKey)
	client := &http.Client{Timeout: otcInterbankTimeout}
	return client.Do(req)
}

// ForwardNegotiationToPartner calls POST <PARTNER_BANK_URL>/otc/interbank/negotiations.
// Returns the partner's (routingNumber, id) tuple, or an error.
func ForwardNegotiationToPartner(body otcNegotiationBody, partnerRoutingNumber string) (int32, string, error) {
	bank, err := gwinterbank.ResolveBankByRoutingNumber(partnerRoutingNumber)
	if err != nil || bank.BankURL == "" {
		return 0, "", fmt.Errorf("cannot resolve partner bank: %w", err)
	}

	resp, err := sendOtcRequest(http.MethodPost, bank.BankURL+"/otc/interbank/negotiations", bank.APIKey, body)
	if err != nil {
		return 0, "", fmt.Errorf("partner request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("partner returned status %d", resp.StatusCode)
	}

	var result struct {
		RoutingNumber int32  `json:"routingNumber"`
		ID            string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", fmt.Errorf("decode partner response: %w", err)
	}
	return result.RoutingNumber, result.ID, nil
}
