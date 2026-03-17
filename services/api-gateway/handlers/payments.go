package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CreatePaymentRequest struct {
	RecipientName    string  `json:"recipientName"    binding:"required"`
	RecipientAccount string  `json:"recipientAccount" binding:"required"`
	Amount           float64 `json:"amount"           binding:"required,gt=0"`
	PaymentCode      string  `json:"paymentCode"`
	ReferenceNumber  string  `json:"referenceNumber"`
	Purpose          string  `json:"purpose"`
	FromAccount      string  `json:"fromAccount"      binding:"required"`
}

// CreatePayment godoc
// @Summary      Create a new payment
// @Description  Initiates a payment from client's account to a recipient account.
// @Tags         payments
// @Accept       json
// @Produce      json
// @Param        body  body      CreatePaymentRequest  true  "Payment data"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      403   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/payments/create [post]
func CreatePayment(paymentClient pb.PaymentServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreatePaymentRequest
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

		resp, err := paymentClient.CreatePayment(ctx, &pb.CreatePaymentRequest{
			ClientId:        clientID,
			FromAccount:     req.FromAccount,
			RecipientName:   req.RecipientName,
			RecipientAccount: req.RecipientAccount,
			Amount:          req.Amount,
			PaymentCode:     req.PaymentCode,
			ReferenceNumber: req.ReferenceNumber,
			Purpose:         req.Purpose,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.PermissionDenied:
				c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
			case codes.FailedPrecondition:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":            resp.Id,
			"orderNumber":   resp.OrderNumber,
			"fromAccount":   resp.FromAccount,
			"toAccount":     resp.ToAccount,
			"initialAmount": resp.InitialAmount,
			"finalAmount":   resp.FinalAmount,
			"fee":           resp.Fee,
			"status":        resp.Status,
			"timestamp":     resp.Timestamp,
		})
	}
}

type CreatePaymentRecipientRequest struct {
	Name          string `json:"name"          binding:"required"`
	AccountNumber string `json:"accountNumber" binding:"required"`
}

// CreatePaymentRecipient godoc
// @Summary      Create a payment recipient
// @Description  Saves a new payment recipient for the authenticated client.
// @Tags         recipients
// @Accept       json
// @Produce      json
// @Param        body  body      CreatePaymentRecipientRequest  true  "Recipient data"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/recipients [post]
func CreatePaymentRecipient(paymentClient pb.PaymentServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreatePaymentRecipientRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		clientID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := paymentClient.CreatePaymentRecipient(ctx, &pb.CreatePaymentRecipientRequest{
			ClientId:      clientID,
			Name:          req.Name,
			AccountNumber: req.AccountNumber,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		r := resp.Recipient
		c.JSON(http.StatusCreated, gin.H{
			"id":            r.Id,
			"clientId":      r.ClientId,
			"name":          r.Name,
			"accountNumber": r.AccountNumber,
		})
	}
}

// GetPaymentRecipients godoc
// @Summary      List payment recipients
// @Description  Returns all saved payment recipients for the authenticated client.
// @Tags         recipients
// @Produce      json
// @Success      200  {array}   map[string]interface{}
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/recipients [get]
func GetPaymentRecipients(paymentClient pb.PaymentServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := paymentClient.GetPaymentRecipients(ctx, &pb.GetPaymentRecipientsRequest{
			ClientId: clientID,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		type recipientJSON struct {
			ID            int64  `json:"id"`
			ClientID      int64  `json:"clientId"`
			Name          string `json:"name"`
			AccountNumber string `json:"accountNumber"`
		}
		result := make([]recipientJSON, 0, len(resp.Recipients))
		for _, r := range resp.Recipients {
			result = append(result, recipientJSON{
				ID:            r.Id,
				ClientID:      r.ClientId,
				Name:          r.Name,
				AccountNumber: r.AccountNumber,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

type UpdatePaymentRecipientRequest struct {
	Name          string `json:"name"          binding:"required"`
	AccountNumber string `json:"accountNumber" binding:"required"`
}

// UpdatePaymentRecipient godoc
// @Summary      Update a payment recipient
// @Description  Updates a saved payment recipient. Only the owning client can update.
// @Tags         recipients
// @Accept       json
// @Produce      json
// @Param        id    path  int                            true  "Recipient ID"
// @Param        body  body  UpdatePaymentRecipientRequest  true  "Recipient data"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/recipients/{id} [put]
func UpdatePaymentRecipient(paymentClient pb.PaymentServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		recipientID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient id"})
			return
		}

		var req UpdatePaymentRecipientRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		clientID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := paymentClient.UpdatePaymentRecipient(ctx, &pb.UpdatePaymentRecipientRequest{
			Id:            recipientID,
			ClientId:      clientID,
			Name:          req.Name,
			AccountNumber: req.AccountNumber,
		})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		r := resp.Recipient
		c.JSON(http.StatusOK, gin.H{
			"id":            r.Id,
			"clientId":      r.ClientId,
			"name":          r.Name,
			"accountNumber": r.AccountNumber,
		})
	}
}

// DeletePaymentRecipient godoc
// @Summary      Delete a payment recipient
// @Description  Deletes a saved payment recipient. Only the owning client can delete.
// @Tags         recipients
// @Param        id  path  int  true  "Recipient ID"
// @Success      204
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/recipients/{id} [delete]
func DeletePaymentRecipient(paymentClient pb.PaymentServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		recipientID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient id"})
			return
		}

		clientID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		_, err = paymentClient.DeletePaymentRecipient(ctx, &pb.DeletePaymentRecipientRequest{
			Id:       recipientID,
			ClientId: clientID,
		})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}
