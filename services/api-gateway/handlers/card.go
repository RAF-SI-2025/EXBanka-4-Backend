package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pbaccount "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	pbcard "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/card"
	pbclient "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pbemail "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func cardToJSON(c *pbcard.CardResponse) gin.H {
	return gin.H{
		"id":            c.Id,
		"cardNumber":    c.CardNumber,
		"cardType":      c.CardType,
		"cardName":      c.CardName,
		"expiryDate":    c.ExpiryDate,
		"accountNumber": c.AccountNumber,
		"cardLimit":     c.CardLimit,
		"status":        c.Status,
		"createdAt":     c.CreatedAt,
	}
}

// GetCardsByAccount godoc
// @Summary  Get all cards for a given account (employee only)
// @Tags     cards
// @Produce  json
// @Param    accountNumber  path  string  true  "Account number"
// @Success  200  {array}  map[string]interface{}
// @Router   /api/cards/by-account/{accountNumber} [get]
func GetCardsByAccount(cardClient pbcard.CardServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := cardClient.GetCardsByAccount(ctx, &pbcard.GetCardsByAccountRequest{
			AccountNumber: c.Param("accountNumber"),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch cards"})
			return
		}
		cards := make([]gin.H, 0, len(resp.Cards))
		for _, card := range resp.Cards {
			cards = append(cards, cardToJSON(card))
		}
		c.JSON(http.StatusOK, cards)
	}
}

// GetMyCards godoc
// @Summary  Get all cards for the authenticated client
// @Tags     cards
// @Produce  json
// @Success  200  {array}  map[string]interface{}
// @Router   /api/cards [get]
func GetMyCards(accountClient pbaccount.AccountServiceClient, cardClient pbcard.CardServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		acctResp, err := accountClient.GetMyAccounts(ctx, &pbaccount.GetMyAccountsRequest{OwnerId: clientID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch accounts"})
			return
		}

		var cards []gin.H
		for _, acct := range acctResp.Accounts {
			resp, err := cardClient.GetCardsByAccount(ctx, &pbcard.GetCardsByAccountRequest{AccountNumber: acct.AccountNumber})
			if err != nil {
				continue // skip accounts that error, don't fail the whole request
			}
			for _, card := range resp.Cards {
				cards = append(cards, cardToJSON(card))
			}
		}
		if cards == nil {
			cards = []gin.H{}
		}
		c.JSON(http.StatusOK, cards)
	}
}

// GetCardById godoc
// @Summary  Get details of a specific card by database ID
// @Tags     cards
// @Produce  json
// @Param    id  path  int  true  "Card ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /api/cards/id/{id} [get]
func GetCardById(cardClient pbcard.CardServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		var id int64
		if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := cardClient.GetCardById(ctx, &pbcard.GetCardByIdRequest{Id: id})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch card"})
			}
			return
		}
		c.JSON(http.StatusOK, cardToJSON(resp.Card))
	}
}

// GetCardByNumber godoc
// @Summary  Get details of a specific card
// @Tags     cards
// @Produce  json
// @Param    number  path  string  true  "Card number"
// @Success  200  {object}  map[string]interface{}
// @Router   /api/cards/{number} [get]
func GetCardByNumber(cardClient pbcard.CardServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := cardClient.GetCardByNumber(ctx, &pbcard.GetCardByNumberRequest{CardNumber: c.Param("number")})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch card"})
			}
			return
		}
		c.JSON(http.StatusOK, cardToJSON(resp.Card))
	}
}

// resolveCardNumber fetches the full card number for a given card ID.
func resolveCardNumber(ctx context.Context, cardClient pbcard.CardServiceClient, rawID string) (string, error) {
	var id int64
	if _, err := fmt.Sscanf(rawID, "%d", &id); err != nil {
		return "", err
	}
	resp, err := cardClient.GetCardById(ctx, &pbcard.GetCardByIdRequest{Id: id})
	if err != nil {
		return "", err
	}
	return resp.Card.CardNumber, nil
}

// BlockCard godoc
// @Summary  Block a card (client — own card only)
// @Tags     cards
// @Produce  json
// @Param    id  path  int  true  "Card ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /api/cards/{id}/block [put]
func BlockCard(cardClient pbcard.CardServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		cardNumber, err := resolveCardNumber(ctx, cardClient, c.Param("id"))
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
			}
			return
		}

		_, err = cardClient.BlockCard(ctx, &pbcard.BlockCardRequest{
			CardNumber:     cardNumber,
			CallerClientId: clientID,
			CallerRole:     middleware.GetCallerRoleFromToken(c),
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.PermissionDenied:
				c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
			case codes.FailedPrecondition:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to block card"})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "card blocked successfully"})
	}
}

// UnblockCard godoc
// @Summary  Unblock a card (employee only)
// @Tags     cards
// @Produce  json
// @Param    id  path  int  true  "Card ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /api/cards/{id}/unblock [put]
func UnblockCard(cardClient pbcard.CardServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		cardNumber, err := resolveCardNumber(ctx, cardClient, c.Param("id"))
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
			}
			return
		}

		_, err = cardClient.UnblockCard(ctx, &pbcard.UnblockCardRequest{CardNumber: cardNumber})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.FailedPrecondition:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unblock card"})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "card unblocked successfully"})
	}
}

// DeactivateCard godoc
// @Summary  Deactivate a card permanently (employee only)
// @Tags     cards
// @Produce  json
// @Param    id  path  int  true  "Card ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /api/cards/{id}/deactivate [put]
func DeactivateCard(cardClient pbcard.CardServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		cardNumber, err := resolveCardNumber(ctx, cardClient, c.Param("id"))
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
			}
			return
		}

		_, err = cardClient.DeactivateCard(ctx, &pbcard.DeactivateCardRequest{CardNumber: cardNumber})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.FailedPrecondition:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate card"})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "card deactivated successfully"})
	}
}

// UpdateCardLimit godoc
// @Summary  Update card spending limit (employee only)
// @Tags     cards
// @Accept   json
// @Produce  json
// @Param    id    path  int     true  "Card ID"
// @Param    body  body  object  true  "New limit"
// @Success  200  {object}  map[string]interface{}
// @Router   /api/cards/{id}/limit [put]
func UpdateCardLimit(cardClient pbcard.CardServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			NewLimit float64 `json:"newLimit" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		cardNumber, err := resolveCardNumber(ctx, cardClient, c.Param("id"))
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
			}
			return
		}

		_, err = cardClient.UpdateCardLimit(ctx, &pbcard.UpdateCardLimitRequest{
			CardNumber: cardNumber,
			NewLimit:   req.NewLimit,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.FailedPrecondition:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update card limit"})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "card limit updated"})
	}
}

// InitiateCardRequest godoc
// @Summary  Request a new card — sends email confirmation code to client
// @Tags     cards
// @Accept   json
// @Produce  json
// @Param    body  body  object  true  "Card request"
// @Success  200   {object}  map[string]interface{}
// @Router   /api/cards/request [post]
func InitiateCardRequest(cardClient pbcard.CardServiceClient, clientClient pbclient.ClientServiceClient, emailClient pbemail.EmailServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		var req struct {
			AccountNumber string `json:"accountNumber" binding:"required"`
			CardName      string `json:"cardName"      binding:"required"`
			ForSelf       bool   `json:"forSelf"`
			AuthorizedPerson *struct {
				FirstName   string `json:"firstName"`
				LastName    string `json:"lastName"`
				DateOfBirth string `json:"dateOfBirth"`
				Gender      string `json:"gender"`
				Email       string `json:"email"`
				PhoneNumber string `json:"phoneNumber"`
				Address     string `json:"address"`
			} `json:"authorizedPerson"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var apData *pbcard.AuthorizedPersonData
		if !req.ForSelf {
			if req.AuthorizedPerson == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "authorized_person data required when for_self is false"})
				return
			}
			apData = &pbcard.AuthorizedPersonData{
				FirstName:   req.AuthorizedPerson.FirstName,
				LastName:    req.AuthorizedPerson.LastName,
				DateOfBirth: req.AuthorizedPerson.DateOfBirth,
				Gender:      req.AuthorizedPerson.Gender,
				Email:       req.AuthorizedPerson.Email,
				PhoneNumber: req.AuthorizedPerson.PhoneNumber,
				Address:     req.AuthorizedPerson.Address,
			}
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		initResp, err := cardClient.InitiateCardRequest(ctx, &pbcard.InitiateCardRequestRequest{
			AccountNumber:    req.AccountNumber,
			CardName:         req.CardName,
			CallerClientId:   clientID,
			ForSelf:          req.ForSelf,
			AuthorizedPerson: apData,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			case codes.ResourceExhausted:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initiate card request"})
			}
			return
		}

		// Fetch client info to send email
		clientResp, err := clientClient.GetClientById(ctx, &pbclient.GetClientByIdRequest{Id: clientID})
		if err == nil {
			emailTarget := clientResp.Client.Email
			firstName := clientResp.Client.FirstName
			_, emailErr := emailClient.SendCardConfirmationEmail(ctx, &pbemail.SendCardConfirmationEmailRequest{
				Email:            emailTarget,
				FirstName:        firstName,
				ConfirmationCode: initResp.ConfirmationCode,
			})
			if emailErr != nil {
				// Log but don't fail — client can retry
				_ = emailErr
			}
		}

		c.JSON(http.StatusOK, gin.H{"requestToken": initResp.RequestToken})
	}
}

// ConfirmCardRequest godoc
// @Summary  Confirm card request with email code — creates the card
// @Tags     cards
// @Accept   json
// @Produce  json
// @Param    body  body  object  true  "Token and code"
// @Success  201   {object}  map[string]interface{}
// @Router   /api/cards/request/confirm [post]
func ConfirmCardRequest(cardClient pbcard.CardServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		var req struct {
			RequestToken string `json:"requestToken" binding:"required"`
			Code         string `json:"code"         binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := cardClient.ConfirmCardRequest(ctx, &pbcard.ConfirmCardRequestRequest{
			RequestToken: req.RequestToken,
			Code:         req.Code,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.FailedPrecondition:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			case codes.PermissionDenied:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid confirmation code"})
			case codes.ResourceExhausted:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm card request"})
			}
			return
		}
		c.JSON(http.StatusCreated, cardToJSON(resp.Card))
	}
}
