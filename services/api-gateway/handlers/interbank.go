package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	pb     "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	pb_otc "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"github.com/gin-gonic/gin"
)

type interbankTransactionId struct {
	RoutingNumber int32  `json:"routingNumber"`
	ID            string `json:"id"`
}

type interbankIdempotenceKey struct {
	RoutingNumber       int32  `json:"routingNumber"`
	LocallyGeneratedKey string `json:"locallyGeneratedKey"`
}

type interbankEnvelope struct {
	IdempotenceKey interbankIdempotenceKey `json:"idempotenceKey"`
	MessageType    string                  `json:"messageType"`
	Message        json.RawMessage         `json:"message"`
}

type newTxMessage struct {
	TransactionId  interbankTransactionId `json:"transactionId"`
	Postings       []interbankPosting     `json:"postings"`
	PaymentCode    string                 `json:"paymentCode"`
	PaymentPurpose string                 `json:"paymentPurpose"`
}

type interbankPosting struct {
	AccountType string  `json:"accountType"`
	AccountNum  string  `json:"accountNum"`
	Amount      float64 `json:"amount"`
	AssetType   string  `json:"assetType"`
	Currency    string  `json:"currency"`
}

type commitRollbackMessage struct {
	TransactionId interbankTransactionId `json:"transactionId"`
}

// InterbankHandler handles POST /interbank for all 2PC message types.
// Authenticated via X-Api-Key header matched against OWN_INTERBANK_API_KEY env var.
func InterbankHandler(paymentClient pb.PaymentServiceClient, otcClient pb_otc.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := os.Getenv("OWN_INTERBANK_API_KEY")
		if apiKey == "" || c.GetHeader("X-Api-Key") != apiKey {
			c.Status(http.StatusUnauthorized)
			return
		}

		var env interbankEnvelope
		if err := c.ShouldBindJSON(&env); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		switch env.MessageType {
		case "NEW_TX":
			handleNewTx(c, paymentClient, otcClient, env)
		case "COMMIT_TX":
			handleCommitRollbackTx(c, paymentClient, otcClient, env, false)
		case "ROLLBACK_TX":
			handleCommitRollbackTx(c, paymentClient, otcClient, env, true)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown messageType: " + env.MessageType})
		}
	}
}

func handleNewTx(c *gin.Context, client pb.PaymentServiceClient, otcClient pb_otc.OtcServiceClient, env interbankEnvelope) {
	var msg newTxMessage
	if err := json.Unmarshal(env.Message, &msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid NEW_TX message"})
		return
	}

	// Detect OPTION posting (cross-bank OTC).
	var optionPosting *interbankPosting
	for i := range msg.Postings {
		if msg.Postings[i].AccountType == "OPTION" {
			optionPosting = &msg.Postings[i]
			break
		}
	}

	pbPostings := make([]*pb.InterbankPosting, len(msg.Postings))
	for i, p := range msg.Postings {
		pbPostings[i] = &pb.InterbankPosting{
			AccountType: p.AccountType,
			AccountNum:  p.AccountNum,
			Amount:      p.Amount,
			AssetType:   p.AssetType,
			Currency:    p.Currency,
		}
	}

	ctx := c.Request.Context()

	paymentResp, err := client.PrepareInterbankPayment(ctx, &pb.PrepareInterbankPaymentRequest{
		IdempotenceKey: &pb.InterbankIdempotenceKey{
			RoutingNumber:       env.IdempotenceKey.RoutingNumber,
			LocallyGeneratedKey: env.IdempotenceKey.LocallyGeneratedKey,
		},
		TransactionId: &pb.InterbankTransactionId{
			RoutingNumber: msg.TransactionId.RoutingNumber,
			Id:            msg.TransactionId.ID,
		},
		Postings:       pbPostings,
		PaymentCode:    msg.PaymentCode,
		PaymentPurpose: msg.PaymentPurpose,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type reason struct {
		Reason string `json:"reason"`
	}

	if optionPosting != nil {
		negID, parseErr := strconv.ParseInt(optionPosting.AccountNum, 10, 64)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid OPTION account num"})
			return
		}
		stockAmount := int32(optionPosting.Amount)
		if stockAmount < 0 {
			stockAmount = -stockAmount
		}

		otcResp, otcErr := otcClient.PrepareOtcInterbank(ctx, &pb_otc.OtcInterbankPrepareRequest{
			IdemRoutingNumber: strconv.Itoa(int(env.IdempotenceKey.RoutingNumber)),
			IdemKey:           env.IdempotenceKey.LocallyGeneratedKey,
			TxRoutingNumber:   strconv.Itoa(int(msg.TransactionId.RoutingNumber)),
			TxId:              msg.TransactionId.ID,
			NegotiationId:     negID,
			StockAmount:       stockAmount,
			IsAccept:          optionPosting.Amount > 0,
		})

		// If OTC voted NO (or errored), rollback payment if it voted YES.
		if otcErr != nil || (otcResp != nil && otcResp.Vote != "YES") {
			if paymentResp.Vote == "YES" {
				_, _ = client.RollbackInterbankPayment(ctx, &pb.CommitRollbackInterbankRequest{
					TransactionId: &pb.InterbankTransactionId{
						RoutingNumber: msg.TransactionId.RoutingNumber,
						Id:            msg.TransactionId.ID,
					},
				})
			}
			otcReason := "OTC_INTERNAL_ERROR"
			if otcResp != nil {
				otcReason = otcResp.Reason
			}
			c.JSON(http.StatusOK, gin.H{"vote": "NO", "reasons": []reason{{Reason: otcReason}}})
			return
		}

		// If payment voted NO, rollback OTC.
		if paymentResp.Vote != "YES" {
			_, _ = otcClient.RollbackOtcInterbank(ctx, &pb_otc.OtcInterbankTxRequest{
				TxRoutingNumber: strconv.Itoa(int(msg.TransactionId.RoutingNumber)),
				TxId:            msg.TransactionId.ID,
			})
			reasons := make([]reason, len(paymentResp.Reasons))
			for i, r := range paymentResp.Reasons {
				reasons[i] = reason{Reason: r.Reason}
			}
			c.JSON(http.StatusOK, gin.H{"vote": "NO", "reasons": reasons})
			return
		}
	} else if paymentResp.Vote != "YES" {
		reasons := make([]reason, len(paymentResp.Reasons))
		for i, r := range paymentResp.Reasons {
			reasons[i] = reason{Reason: r.Reason}
		}
		c.JSON(http.StatusOK, gin.H{"vote": "NO", "reasons": reasons})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vote": "YES", "reasons": []reason{}})
}

func handleCommitRollbackTx(c *gin.Context, client pb.PaymentServiceClient, otcClient pb_otc.OtcServiceClient, env interbankEnvelope, isRollback bool) {
	var msg commitRollbackMessage
	if err := json.Unmarshal(env.Message, &msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid COMMIT_TX/ROLLBACK_TX message"})
		return
	}

	ctx := c.Request.Context()
	paymentReq := &pb.CommitRollbackInterbankRequest{
		TransactionId: &pb.InterbankTransactionId{
			RoutingNumber: msg.TransactionId.RoutingNumber,
			Id:            msg.TransactionId.ID,
		},
	}
	otcReq := &pb_otc.OtcInterbankTxRequest{
		TxRoutingNumber: strconv.Itoa(int(msg.TransactionId.RoutingNumber)),
		TxId:            msg.TransactionId.ID,
	}

	if isRollback {
		_, _ = client.RollbackInterbankPayment(ctx, paymentReq)
		_, _ = otcClient.RollbackOtcInterbank(ctx, otcReq)
	} else {
		if _, err := client.CommitInterbankPayment(ctx, paymentReq); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if _, err := otcClient.CommitOtcInterbank(ctx, otcReq); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.Status(http.StatusNoContent)
}
