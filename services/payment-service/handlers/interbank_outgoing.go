package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/payment-service/interbank"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	interbankTimeout    = 10 * time.Second
	interbankMaxRetries = 3
)

// --- JSON types for the /interbank envelope ---

type ibTransactionID struct {
	RoutingNumber int    `json:"routingNumber"`
	ID            string `json:"id"`
}

type ibIdempotenceKey struct {
	RoutingNumber       int    `json:"routingNumber"`
	LocallyGeneratedKey string `json:"locallyGeneratedKey"`
}

type ibPosting struct {
	AccountType string  `json:"accountType"`
	AccountNum  string  `json:"accountNum"`
	Amount      float64 `json:"amount"`
	AssetType   string  `json:"assetType"`
	Currency    string  `json:"currency"`
}

type ibNewTxMessage struct {
	TransactionID  ibTransactionID `json:"transactionId"`
	Postings       []ibPosting     `json:"postings"`
	PaymentCode    string          `json:"paymentCode"`
	PaymentPurpose string          `json:"paymentPurpose"`
}

type ibCommitRollbackMessage struct {
	TransactionID ibTransactionID `json:"transactionId"`
}

type ibEnvelope struct {
	IdempotenceKey ibIdempotenceKey `json:"idempotenceKey"`
	MessageType    string           `json:"messageType"`
	Message        any              `json:"message"`
}

type ibVoteResponse struct {
	Vote         string   `json:"vote"`
	Reasons      []string `json:"reasons"`
	ExchangeRate float64  `json:"exchange_rate,omitempty"`
	Fee          float64  `json:"fee,omitempty"`
	FinalAmount  float64  `json:"final_amount,omitempty"`
}

// generateUUID returns a random UUID v4 string without external deps.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ownRoutingInt returns OWN_ROUTING_NUMBER as int.
func ownRoutingInt() int {
	var n int
	fmt.Sscanf(os.Getenv("OWN_ROUTING_NUMBER"), "%d", &n)
	return n
}

// sendInterbankRequest POSTs body to url with X-Api-Key auth.
// Retries up to interbankMaxRetries times if the partner responds 202.
func sendInterbankRequest(ctx context.Context, url, apiKey string, body any) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	client := &http.Client{Timeout: interbankTimeout}
	var lastErr error
	for attempt := 0; attempt < interbankMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Api-Key", apiKey)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusAccepted {
			_ = resp.Body.Close()
			continue
		}

		return resp, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", interbankMaxRetries, lastErr)
	}
	return nil, fmt.Errorf("partner bank returned 202 for all %d attempts", interbankMaxRetries)
}

func sendRollback(ctx context.Context, bankURL, apiKey string, txID ibTransactionID) {
	ownRouting := ownRoutingInt()
	envelope := ibEnvelope{
		IdempotenceKey: ibIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: generateUUID()},
		MessageType:    "ROLLBACK_TX",
		Message:        ibCommitRollbackMessage{TransactionID: txID},
	}
	resp, err := sendInterbankRequest(ctx, bankURL, apiKey, envelope)
	if err == nil {
		_ = resp.Body.Close()
	}
}

// executeOutgoing2PC runs the full outgoing Two-Phase Commit when the
// recipient account belongs to a partner bank.
func executeOutgoing2PC(ctx context.Context, s *PaymentServer, req *pb.CreatePaymentRequest, routingNum string) (*pb.CreatePaymentResponse, error) {
	bank, err := interbank.ResolveBankByRoutingNumber(routingNum)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unknown bank routing number %s", routingNum)
	}
	bankURL := bank.BankURL + "/interbank"

	// Load sender metadata.
	var fromID, ownerID, fromCurrencyID int64
	if err := s.AccountDB.QueryRowContext(ctx,
		`SELECT id, owner_id, currency_id FROM accounts WHERE account_number = $1`,
		req.FromAccount,
	).Scan(&fromID, &ownerID, &fromCurrencyID); err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "source account %s not found", req.FromAccount)
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "load source account: %v", err)
	}
	if ownerID != req.ClientId {
		return nil, status.Errorf(codes.PermissionDenied, "account does not belong to this client")
	}

	var currencyCode string
	if err := s.ExchangeDB.QueryRowContext(ctx,
		`SELECT code FROM currencies WHERE id = $1`, fromCurrencyID,
	).Scan(&currencyCode); err != nil {
		return nil, status.Errorf(codes.Internal, "resolve currency: %v", err)
	}

	// DB tx #1: validate limits and reserve funds.
	resvTx, err := s.AccountDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin reservation tx: %v", err)
	}

	var availBal float64
	var dailyLimit, monthlyLimit sql.NullFloat64
	var dailySpent, monthlySpent float64
	if err = resvTx.QueryRowContext(ctx, `
		SELECT available_balance, daily_limit, monthly_limit, daily_spent, monthly_spent
		FROM accounts WHERE id = $1 FOR UPDATE`, fromID,
	).Scan(&availBal, &dailyLimit, &monthlyLimit, &dailySpent, &monthlySpent); err != nil {
		_ = resvTx.Rollback()
		return nil, status.Errorf(codes.Internal, "lock source account: %v", err)
	}
	if availBal < req.Amount {
		_ = resvTx.Rollback()
		return nil, status.Errorf(codes.FailedPrecondition, "insufficient funds")
	}
	if dailyLimit.Valid && dailySpent+req.Amount > dailyLimit.Float64 {
		_ = resvTx.Rollback()
		return nil, status.Errorf(codes.FailedPrecondition, "daily limit exceeded")
	}
	if monthlyLimit.Valid && monthlySpent+req.Amount > monthlyLimit.Float64 {
		_ = resvTx.Rollback()
		return nil, status.Errorf(codes.FailedPrecondition, "monthly limit exceeded")
	}

	if _, err = resvTx.ExecContext(ctx, `
		UPDATE accounts SET
			available_balance = available_balance - $1,
			daily_spent       = daily_spent + $1,
			monthly_spent     = monthly_spent + $1
		WHERE id = $2`, req.Amount, fromID); err != nil {
		_ = resvTx.Rollback()
		return nil, status.Errorf(codes.Internal, "reserve funds: %v", err)
	}
	if err = resvTx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit reservation: %v", err)
	}

	releaseReservation := func() {
		_, _ = s.AccountDB.ExecContext(ctx, `
			UPDATE accounts SET
				available_balance = available_balance + $1,
				daily_spent       = daily_spent - $1,
				monthly_spent     = monthly_spent - $1
			WHERE id = $2`, req.Amount, fromID)
	}

	// Generate stable identifiers for this transaction.
	ownRouting := ownRoutingInt()
	txID := ibTransactionID{RoutingNumber: ownRouting, ID: generateUUID()}
	idemKey := ibIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: generateUUID()}

	// Phase 1: NEW_TX — ask partner bank to prepare.
	newTxResp, err := sendInterbankRequest(ctx, bankURL, bank.APIKey, ibEnvelope{
		IdempotenceKey: idemKey,
		MessageType:    "NEW_TX",
		Message: ibNewTxMessage{
			TransactionID: txID,
			Postings: []ibPosting{
				{AccountType: "ACCOUNT", AccountNum: req.FromAccount, Amount: -req.Amount, AssetType: "MONAS", Currency: currencyCode},
				{AccountType: "ACCOUNT", AccountNum: req.RecipientAccount, Amount: req.Amount, AssetType: "MONAS", Currency: currencyCode},
			},
			PaymentCode:    req.PaymentCode,
			PaymentPurpose: req.Purpose,
		},
	})
	if err != nil {
		releaseReservation()
		return nil, status.Errorf(codes.Unavailable, "NEW_TX request failed: %v", err)
	}
	defer newTxResp.Body.Close()

	if newTxResp.StatusCode != http.StatusOK {
		releaseReservation()
		return nil, status.Errorf(codes.Unavailable, "partner bank NEW_TX returned status %d", newTxResp.StatusCode)
	}

	var vote ibVoteResponse
	if err := json.NewDecoder(newTxResp.Body).Decode(&vote); err != nil {
		sendRollback(ctx, bankURL, bank.APIKey, txID)
		releaseReservation()
		return nil, status.Errorf(codes.Internal, "decode vote response: %v", err)
	}

	if vote.Vote != "YES" {
		releaseReservation()
		if len(vote.Reasons) > 0 {
			return nil, status.Errorf(codes.FailedPrecondition, "partner bank voted NO: %v", vote.Reasons)
		}
		return nil, status.Errorf(codes.FailedPrecondition, "partner bank voted NO")
	}

	// Phase 2: COMMIT_TX.
	commitResp, err := sendInterbankRequest(ctx, bankURL, bank.APIKey, ibEnvelope{
		IdempotenceKey: ibIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: generateUUID()},
		MessageType:    "COMMIT_TX",
		Message:        ibCommitRollbackMessage{TransactionID: txID},
	})
	if err != nil {
		sendRollback(ctx, bankURL, bank.APIKey, txID)
		releaseReservation()
		return nil, status.Errorf(codes.Unavailable, "COMMIT_TX request failed: %v", err)
	}
	defer commitResp.Body.Close()

	if commitResp.StatusCode != http.StatusNoContent {
		sendRollback(ctx, bankURL, bank.APIKey, txID)
		releaseReservation()
		return nil, status.Errorf(codes.Unavailable, "partner bank COMMIT_TX returned status %d", commitResp.StatusCode)
	}

	// DB tx #2: apply debit now that partner committed.
	// Decreases balance and restores available_balance (cancels reservation).
	_, _ = s.AccountDB.ExecContext(ctx, `
		UPDATE accounts SET
			balance           = balance - $1,
			available_balance = available_balance + $1
		WHERE id = $2`, req.Amount, fromID)

	// Persist payment record.
	orderNumber := fmt.Sprintf("ORD-%d-%s", time.Now().UnixMilli(), generateUUID()[:8])
	now := time.Now()

	var paymentID int64
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO payments
			(order_number, from_account, to_account, initial_amount, final_amount,
			 fee, recipient_id, payment_code, reference_number, purpose, timestamp, status)
		VALUES ($1, $2, $3, $4, $5, 0,
			(SELECT id FROM payment_recipients WHERE client_id = $6 AND account_number = $7 LIMIT 1),
			$8, $9, $10, $11, 'COMPLETED')
		RETURNING id`,
		orderNumber, req.FromAccount, req.RecipientAccount,
		req.Amount, req.Amount,
		req.ClientId, req.RecipientAccount,
		req.PaymentCode, req.ReferenceNumber, req.Purpose, now,
	).Scan(&paymentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "persist payment: %v", err)
	}

	return &pb.CreatePaymentResponse{
		Id:            paymentID,
		OrderNumber:   orderNumber,
		FromAccount:   req.FromAccount,
		ToAccount:     req.RecipientAccount,
		InitialAmount: req.Amount,
		FinalAmount:   req.Amount,
		Status:        "COMPLETED",
		Timestamp:     now.Format(time.RFC3339),
	}, nil
}
