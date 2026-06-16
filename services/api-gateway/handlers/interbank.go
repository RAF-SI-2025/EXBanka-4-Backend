package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	pb_otc "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// Nested posting format as used by Banka 4.
type ibPartyID struct {
	RoutingNumber int32  `json:"routingNumber"`
	ID            string `json:"id"`
}
type ibAccount struct {
	Type string     `json:"type"` // "ACCOUNT" | "PERSON" | "OPTION"
	Num  string     `json:"num,omitempty"`
	ID   *ibPartyID `json:"id,omitempty"`
}
type ibAssetInner struct {
	Currency      string     `json:"currency,omitempty"`
	Ticker        string     `json:"ticker,omitempty"`
	NegotiationID *ibPartyID `json:"negotiationId,omitempty"`
}
type ibAsset struct {
	Type  string        `json:"type"` // "MONAS" | "STOCK" | "OPTION"
	Asset *ibAssetInner `json:"asset,omitempty"`
}
type interbankPosting struct {
	Account ibAccount `json:"account"`
	Amount  float64   `json:"amount"`
	Asset   ibAsset   `json:"asset"`
}

type newTxMessage struct {
	TransactionId  interbankTransactionId `json:"transactionId"`
	Postings       []interbankPosting     `json:"postings"`
	PaymentCode    string                 `json:"paymentCode"`
	PaymentPurpose string                 `json:"paymentPurpose"`
}

type commitRollbackMessage struct {
	TransactionId interbankTransactionId `json:"transactionId"`
}

// InterbankHandler handles POST /interbank for all 2PC message types.
// Authenticated via X-Api-Key header matched against OWN_INTERBANK_API_KEY env var.
func InterbankHandler(paymentClient pb.PaymentServiceClient, otcClient pb_otc.OtcServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validateOtcInterbankKey(c) {
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

	ctx := c.Request.Context()
	ownRouting := gatewayOwnRoutingInt()

	// Detect OTC postings in the nested Banka 4 format.
	var exercisePosting, acceptPosting *interbankPosting
	for i := range msg.Postings {
		p := &msg.Postings[i]
		if p.Account.ID == nil {
			continue
		}
		rn := p.Account.ID.RoutingNumber
		// Exercise: OPTION account on our bank gives STOCK (amount < 0 means asset leaves)
		if p.Account.Type == "OPTION" && rn == ownRouting && p.Asset.Type == "STOCK" && p.Amount < 0 {
			exercisePosting = p
		}
		// Accept: PERSON account on our bank receives OPTION right (amount > 0 means asset enters)
		if p.Account.Type == "PERSON" && rn == ownRouting && p.Asset.Type == "OPTION" && p.Amount > 0 {
			acceptPosting = p
		}
	}

	if exercisePosting != nil || acceptPosting != nil {
		// OTC transaction — skip payment service entirely.
		var negID int64
		var partnerNegExternalID string
		var stockAmt int32
		var isAccept bool
		if acceptPosting != nil {
			if acceptPosting.Asset.Asset != nil && acceptPosting.Asset.Asset.NegotiationID != nil {
				partnerNegExternalID = acceptPosting.Asset.Asset.NegotiationID.ID
				negID, _ = strconv.ParseInt(partnerNegExternalID, 10, 64)
			}
			stockAmt = 1
			isAccept = true
		} else {
			partnerNegExternalID = exercisePosting.Account.ID.ID
			negID, _ = strconv.ParseInt(partnerNegExternalID, 10, 64)
			stockAmt = int32(math.Abs(exercisePosting.Amount))
			isAccept = false
		}
		otcResp, otcErr := otcClient.PrepareOtcInterbank(ctx, &pb_otc.OtcInterbankPrepareRequest{
			IdemRoutingNumber:    fmt.Sprintf("%d", env.IdempotenceKey.RoutingNumber),
			IdemKey:              env.IdempotenceKey.LocallyGeneratedKey,
			TxRoutingNumber:      fmt.Sprintf("%d", msg.TransactionId.RoutingNumber),
			TxId:                 msg.TransactionId.ID,
			NegotiationId:        negID,
			StockAmount:          stockAmt,
			IsAccept:             isAccept,
			PartnerNegExternalId: partnerNegExternalID,
		})
		vote := "NO"
		reason := ""
		if otcErr == nil && otcResp != nil && otcResp.Vote == "YES" {
			vote = "YES"
		} else if otcResp != nil {
			reason = otcResp.Reason
		}
		c.JSON(http.StatusOK, gin.H{"vote": vote, "reasons": []string{reason}})
		return
	}

	// Non-OTC: convert nested Banka 4 format → flat proto for payment service.
	var protoPostings []*pb.InterbankPosting
	for _, p := range msg.Postings {
		accountNum := p.Account.Num // set for "ACCOUNT" type
		if accountNum == "" && p.Account.ID != nil {
			accountNum = p.Account.ID.ID // set for "PERSON" type
		}
		currency := ""
		if p.Asset.Asset != nil {
			currency = p.Asset.Asset.Currency
		}
		protoPostings = append(protoPostings, &pb.InterbankPosting{
			AccountType: p.Account.Type,
			AccountNum:  accountNum,
			Amount:      p.Amount,
			AssetType:   p.Asset.Type,
			Currency:    currency,
		})
	}

	type reason struct {
		Reason string `json:"reason"`
	}

	paymentResp, err := client.PrepareInterbankPayment(ctx, &pb.PrepareInterbankPaymentRequest{
		IdempotenceKey: &pb.InterbankIdempotenceKey{
			RoutingNumber:       env.IdempotenceKey.RoutingNumber,
			LocallyGeneratedKey: env.IdempotenceKey.LocallyGeneratedKey,
		},
		TransactionId: &pb.InterbankTransactionId{
			RoutingNumber: msg.TransactionId.RoutingNumber,
			Id:            msg.TransactionId.ID,
		},
		Postings:       protoPostings,
		PaymentCode:    msg.PaymentCode,
		PaymentPurpose: msg.PaymentPurpose,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if paymentResp.Vote != "YES" {
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
		// Payment service returns codes.NotFound for OTC-only transactions — treat as no-op.
		if _, err := client.CommitInterbankPayment(ctx, paymentReq); err != nil {
			if status.Code(err) != codes.NotFound {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		if _, err := otcClient.CommitOtcInterbank(ctx, otcReq); err != nil {
			if status.Code(err) != codes.NotFound {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}
	c.Status(http.StatusNoContent)
}
