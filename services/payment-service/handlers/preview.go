package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/payment-service/interbank"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *PaymentServer) PreviewPayment(ctx context.Context, req *pb.PreviewPaymentRequest) (*pb.PreviewPaymentResponse, error) {
	req.FromAccount = strings.ReplaceAll(req.FromAccount, "-", "")
	req.RecipientAccount = strings.ReplaceAll(req.RecipientAccount, "-", "")

	var ownerID, fromCurrencyID int64
	if err := s.AccountDB.QueryRowContext(ctx,
		`SELECT owner_id, currency_id FROM accounts WHERE account_number = $1`,
		req.FromAccount,
	).Scan(&ownerID, &fromCurrencyID); err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "source account not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "load source account: %v", err)
	}
	if ownerID != req.ClientId {
		return nil, status.Errorf(codes.PermissionDenied, "account does not belong to this client")
	}

	var fromCurrencyCode string
	if err := s.ExchangeDB.QueryRowContext(ctx,
		`SELECT code FROM currencies WHERE id = $1`, fromCurrencyID,
	).Scan(&fromCurrencyCode); err != nil {
		return nil, status.Errorf(codes.Internal, "resolve source currency: %v", err)
	}

	routingNum := interbank.ExtractRoutingNumber(req.RecipientAccount)

	if !interbank.IsOwnBank(routingNum) {
		return s.previewCrossBank(ctx, req, fromCurrencyCode, routingNum)
	}
	return s.previewOwnBank(ctx, req, fromCurrencyID, fromCurrencyCode)
}

// previewCrossBank probes the partner bank with a NEW_TX and immediately rolls it back,
// returning whatever rate/fee the partner quoted. Falls back gracefully on any error.
func (s *PaymentServer) previewCrossBank(ctx context.Context, req *pb.PreviewPaymentRequest, fromCurrencyCode, routingNum string) (*pb.PreviewPaymentResponse, error) {
	fallback := &pb.PreviewPaymentResponse{
		IsCrossBank:  true,
		ExchangeRate: 0,
		Fee:          0,
		FinalAmount:  req.Amount,
		FromCurrency: fromCurrencyCode,
	}

	bank, err := interbank.ResolveBankByRoutingNumber(routingNum)
	if err != nil || bank.BankURL == "" {
		return fallback, nil
	}
	bankURL := bank.BankURL + "/interbank"

	ownRouting := ownRoutingInt()
	txID := ibTransactionID{RoutingNumber: ownRouting, ID: generateUUID()}
	idemKey := ibIdempotenceKey{RoutingNumber: ownRouting, LocallyGeneratedKey: generateUUID()}

	probeCtx, cancel := context.WithTimeout(ctx, interbankTimeout)
	defer cancel()

	probeResp, probeErr := sendInterbankRequest(probeCtx, bankURL, bank.APIKey, ibEnvelope{
		IdempotenceKey: idemKey,
		MessageType:    "NEW_TX",
		Message: ibNewTxMessage{
			TransactionID: txID,
			Postings: []ibPosting{
				{AccountType: "ACCOUNT", AccountNum: req.FromAccount, Amount: -req.Amount, AssetType: "MONAS", Currency: fromCurrencyCode},
				{AccountType: "ACCOUNT", AccountNum: req.RecipientAccount, Amount: req.Amount, AssetType: "MONAS", Currency: fromCurrencyCode},
			},
		},
	})

	// Always roll back — we must not leave a dangling PENDING on the partner side
	sendRollback(ctx, bankURL, bank.APIKey, txID)

	if probeErr != nil || probeResp.StatusCode != http.StatusOK {
		return fallback, nil
	}
	defer probeResp.Body.Close()

	var vote ibVoteResponse
	if err := json.NewDecoder(probeResp.Body).Decode(&vote); err != nil || vote.Vote != "YES" {
		return fallback, nil
	}

	finalAmount := vote.FinalAmount
	if finalAmount == 0 {
		finalAmount = req.Amount - vote.Fee
	}
	if finalAmount <= 0 {
		finalAmount = req.Amount
	}
	return &pb.PreviewPaymentResponse{
		IsCrossBank:  true,
		ExchangeRate: vote.ExchangeRate,
		Fee:          vote.Fee,
		FinalAmount:  finalAmount,
		FromCurrency: fromCurrencyCode,
	}, nil
}

// previewOwnBank calculates rate/fee for a same-bank payment (may still be cross-currency).
func (s *PaymentServer) previewOwnBank(ctx context.Context, req *pb.PreviewPaymentRequest, fromCurrencyID int64, fromCurrencyCode string) (*pb.PreviewPaymentResponse, error) {
	var toCurrencyID int64
	err := s.AccountDB.QueryRowContext(ctx,
		`SELECT currency_id FROM accounts WHERE account_number = $1`,
		req.RecipientAccount,
	).Scan(&toCurrencyID)
	if err == sql.ErrNoRows || toCurrencyID == fromCurrencyID {
		return &pb.PreviewPaymentResponse{
			IsCrossBank:  false,
			ExchangeRate: 1,
			Fee:          0,
			FinalAmount:  req.Amount,
			FromCurrency: fromCurrencyCode,
		}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "look up recipient: %v", err)
	}

	var toCurrencyCode string
	if err := s.ExchangeDB.QueryRowContext(ctx,
		`SELECT code FROM currencies WHERE id = $1`, toCurrencyID,
	).Scan(&toCurrencyCode); err != nil {
		return nil, status.Errorf(codes.Internal, "resolve destination currency: %v", err)
	}

	const commission = 0.005
	today := time.Now().Format("2006-01-02")
	getRate := func(code, rateType string) (float64, error) {
		if code == "RSD" {
			return 1.0, nil
		}
		var r float64
		e := s.ExchangeDB.QueryRowContext(ctx,
			`SELECT `+rateType+` FROM daily_exchange_rates WHERE currency_code = $1 AND date = $2`,
			code, today,
		).Scan(&r)
		if e == sql.ErrNoRows {
			e = s.ExchangeDB.QueryRowContext(ctx,
				`SELECT rate FROM exchange_rates WHERE from_currency = $1 AND to_currency = 'RSD'`,
				code,
			).Scan(&r)
		}
		return r, e
	}

	var exchangeRate, finalAmount float64
	finalAmount = req.Amount
	switch {
	case fromCurrencyCode == "RSD":
		toSelling, err := getRate(toCurrencyCode, "selling_rate")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "get rate for %s: %v", toCurrencyCode, err)
		}
		finalAmount = (req.Amount / toSelling) * (1 - commission)
		exchangeRate = 1 / toSelling
	case toCurrencyCode == "RSD":
		fromBuying, err := getRate(fromCurrencyCode, "buying_rate")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "get rate for %s: %v", fromCurrencyCode, err)
		}
		finalAmount = req.Amount * fromBuying * (1 - commission)
		exchangeRate = fromBuying
	default:
		fromBuying, err := getRate(fromCurrencyCode, "buying_rate")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "get rate for %s: %v", fromCurrencyCode, err)
		}
		toSelling, err := getRate(toCurrencyCode, "selling_rate")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "get rate for %s: %v", toCurrencyCode, err)
		}
		rsdAmount := req.Amount * fromBuying * (1 - commission)
		finalAmount = (rsdAmount / toSelling) * (1 - commission)
		exchangeRate = fromBuying / toSelling
	}

	return &pb.PreviewPaymentResponse{
		IsCrossBank:  false,
		ExchangeRate: math.Round(exchangeRate*10000) / 10000,
		Fee:          math.Round(req.Amount*commission*100) / 100,
		FinalAmount:  math.Round(finalAmount*100) / 100,
		FromCurrency: fromCurrencyCode,
	}, nil
}
