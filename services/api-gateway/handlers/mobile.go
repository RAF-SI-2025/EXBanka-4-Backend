package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	pb_account "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	pb_payment "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type approvalResp struct {
	ID        int64       `json:"id"`
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Status    string      `json:"status"`
	CreatedAt string      `json:"createdAt"`
	ExpiresAt string      `json:"expiresAt"`
}

func toApprovalResp(a *pb.Approval) approvalResp {
	var payload interface{} = map[string]interface{}{}
	_ = json.Unmarshal([]byte(a.Payload), &payload)
	return approvalResp{
		ID:        a.Id,
		Type:      a.ActionType,
		Payload:   payload,
		Status:    a.Status,
		CreatedAt: a.CreatedAt,
		ExpiresAt: a.ExpiresAt,
	}
}

// CreateApproval creates a new two-factor approval request for the authenticated client.
func CreateApproval(authClient pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var body struct {
			ActionType string `json:"actionType" binding:"required"`
			Payload    string `json:"payload"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := authClient.CreateApproval(ctx, &pb.CreateApprovalRequest{
			ClientId:   userID,
			ActionType: body.ActionType,
			Payload:    body.Payload,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, toApprovalResp(resp.Approval))
	}
}

// PollLoginApproval is a public endpoint for polling a LOGIN approval status.
// No authentication required — used by the web frontend while waiting for mobile approval.
func PollLoginApproval(authClient pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := authClient.PollApproval(ctx, &pb.PollApprovalRequest{Id: id})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "approval not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":        resp.Status,
			"access_token":  resp.AccessToken,
			"refresh_token": resp.RefreshToken,
		})
	}
}

// GetMyApprovals returns all approvals for the authenticated client.
func GetMyApprovals(authClient pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := authClient.GetClientApprovals(ctx, &pb.GetClientApprovalsRequest{ClientId: userID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		result := make([]approvalResp, len(resp.Approvals))
		for i, a := range resp.Approvals {
			result[i] = toApprovalResp(a)
		}
		c.JSON(http.StatusOK, result)
	}
}

// GetMyApprovalById returns a single approval by ID, enforcing client ownership.
func GetMyApprovalById(authClient pb.AuthServiceClient) gin.HandlerFunc {
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
		resp, err := authClient.GetApproval(ctx, &pb.GetApprovalRequest{Id: id})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "approval not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if resp.Approval.ClientId != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusOK, toApprovalResp(resp.Approval))
	}
}

// ApproveApproval approves a pending two-factor approval.
func ApproveApproval(authClient pb.AuthServiceClient, accountClient pb_account.AccountServiceClient, paymentClient pb_payment.PaymentServiceClient) gin.HandlerFunc {
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
		resp, err := authClient.UpdateApprovalStatus(ctx, &pb.UpdateApprovalStatusRequest{
			Id:       id,
			ClientId: userID,
			Status:   "APPROVED",
		})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "approval not found or already resolved"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Post-approval side effects
		switch resp.Approval.ActionType {
		case "LIMIT_CHANGE":
			var p struct {
				AccountID    int64   `json:"accountId"`
				DailyLimit   float64 `json:"dailyLimit"`
				MonthlyLimit float64 `json:"monthlyLimit"`
			}
			if json.Unmarshal([]byte(resp.Approval.Payload), &p) == nil && p.AccountID != 0 {
				sCtx, sCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer sCancel()
				_, _ = accountClient.UpdateAccountLimits(sCtx, &pb_account.UpdateAccountLimitsRequest{
					AccountId:    p.AccountID,
					OwnerId:      userID,
					DailyLimit:   p.DailyLimit,
					MonthlyLimit: p.MonthlyLimit,
				})
			}
		case "PAYMENT":
			var p struct {
				FromAccount     string  `json:"fromAccount"`
				RecipientName   string  `json:"recipientName"`
				RecipientAccount string `json:"recipientAccount"`
				Amount          float64 `json:"amount"`
				PaymentCode     string  `json:"paymentCode"`
				ReferenceNumber string  `json:"referenceNumber"`
				Purpose         string  `json:"purpose"`
			}
			if json.Unmarshal([]byte(resp.Approval.Payload), &p) == nil && p.FromAccount != "" {
				sCtx, sCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer sCancel()
				_, _ = paymentClient.CreatePayment(sCtx, &pb_payment.CreatePaymentRequest{
					ClientId:         userID,
					FromAccount:      p.FromAccount,
					RecipientName:    p.RecipientName,
					RecipientAccount: p.RecipientAccount,
					Amount:           p.Amount,
					PaymentCode:      p.PaymentCode,
					ReferenceNumber:  p.ReferenceNumber,
					Purpose:          p.Purpose,
				})
			}
		case "TRANSFER":
			var p struct {
				FromAccount string  `json:"fromAccount"`
				ToAccount   string  `json:"toAccount"`
				Amount      float64 `json:"amount"`
			}
			if json.Unmarshal([]byte(resp.Approval.Payload), &p) == nil && p.FromAccount != "" {
				sCtx, sCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer sCancel()
				_, _ = paymentClient.CreateTransfer(sCtx, &pb_payment.CreateTransferRequest{
					ClientId:    userID,
					FromAccount: p.FromAccount,
					ToAccount:   p.ToAccount,
					Amount:      p.Amount,
				})
			}
		}

		c.JSON(http.StatusOK, toApprovalResp(resp.Approval))
	}
}

// RejectApproval rejects a pending two-factor approval.
func RejectApproval(authClient pb.AuthServiceClient) gin.HandlerFunc {
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
		resp, err := authClient.UpdateApprovalStatus(ctx, &pb.UpdateApprovalStatusRequest{
			Id:       id,
			ClientId: userID,
			Status:   "REJECTED",
		})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "approval not found or already resolved"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, toApprovalResp(resp.Approval))
	}
}

// RegisterMobilePushToken registers (or updates) the Expo push token for the client.
func RegisterMobilePushToken(authClient pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var body struct {
			Token string `json:"token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		_, err = authClient.RegisterPushToken(ctx, &pb.RegisterPushTokenRequest{
			ClientId: userID,
			Token:    body.Token,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// UnregisterMobilePushToken removes the Expo push token for the client.
func UnregisterMobilePushToken(authClient pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		_, err = authClient.UnregisterPushToken(ctx, &pb.UnregisterPushTokenRequest{ClientId: userID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
