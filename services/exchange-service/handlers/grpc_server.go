package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/exchange"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	spread     = 0.02  // 2% spread for buying/selling rates
	commission = 0.005 // 0.5% commission per conversion step
	rateAPIURL = "https://open.er-api.com/v6/latest/RSD"
)

// Static fallback rates (RSD → foreign) in case the external API is unavailable
var fallbackRates = map[string]float64{
	"EUR": 0.008547,
	"CHF": 0.008621,
	"USD": 0.009259,
	"GBP": 0.007353,
	"JPY": 1.380000,
	"CAD": 0.012500,
	"AUD": 0.014286,
}

type ExchangeServer struct {
	pb.UnimplementedExchangeServiceServer
	DB        *sql.DB // exchange_db
	AccountDB *sql.DB // account_db
}

// ensureTodayRates fetches rates from external API if today's rates are not yet in DB.
func (s *ExchangeServer) ensureTodayRates(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")
	var count int
	if err := s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM daily_exchange_rates WHERE date = $1`, today,
	).Scan(&count); err != nil {
		return fmt.Errorf("checking daily rates: %w", err)
	}
	if count > 0 {
		return nil // already populated
	}
	return s.fetchAndStoreRates(ctx, today)
}

type erAPIResponse struct {
	Result  string             `json:"result"`
	Rates   map[string]float64 `json:"rates"`
}

func fetchRatesFromAPI() (map[string]float64, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(rateAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data erAPIResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if data.Result != "success" {
		return nil, fmt.Errorf("API returned non-success result")
	}
	return data.Rates, nil
}

func (s *ExchangeServer) fetchAndStoreRates(ctx context.Context, today string) error {
	// 1 RSD = rates[CURRENCY] — invert to get "1 CURRENCY = X RSD"
	rawRates, err := fetchRatesFromAPI()
	if err != nil {
		log.Printf("exchange-service: external API failed (%v), using fallback rates", err)
		rawRates = fallbackRates
	}

	currencies := []string{"EUR", "CHF", "USD", "GBP", "JPY", "CAD", "AUD"}
	for _, code := range currencies {
		raw, ok := rawRates[code]
		if !ok || raw == 0 {
			if fb, fbOk := fallbackRates[code]; fbOk {
				raw = fb
			} else {
				continue
			}
		}
		middle := 1.0 / raw
		buying := middle * (1 - spread/2)
		selling := middle * (1 + spread/2)
		_, err := s.DB.ExecContext(ctx, `
			INSERT INTO daily_exchange_rates (currency_code, buying_rate, selling_rate, middle_rate, date)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (currency_code, date) DO NOTHING`,
			code,
			math.Round(buying*1e6)/1e6,
			math.Round(selling*1e6)/1e6,
			math.Round(middle*1e6)/1e6,
			today,
		)
		if err != nil {
			return fmt.Errorf("inserting rate for %s: %w", code, err)
		}
	}
	return nil
}

// GetExchangeRates returns today's buy/sell/middle rates for all currencies.
func (s *ExchangeServer) GetExchangeRates(ctx context.Context, _ *pb.GetExchangeRatesRequest) (*pb.GetExchangeRatesResponse, error) {
	if err := s.ensureTodayRates(ctx); err != nil {
		log.Printf("exchange-service: ensureTodayRates: %v", err)
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT currency_code, buying_rate, selling_rate, middle_rate, date
		FROM daily_exchange_rates
		WHERE date = CURRENT_DATE
		ORDER BY currency_code`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query exchange rates: %v", err)
	}
	defer rows.Close()

	var rates []*pb.ExchangeRate
	for rows.Next() {
		var r pb.ExchangeRate
		var d time.Time
		if err := rows.Scan(&r.CurrencyCode, &r.BuyingRate, &r.SellingRate, &r.MiddleRate, &d); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan rate: %v", err)
		}
		r.Date = d.Format("2006-01-02")
		rates = append(rates, &r)
	}
	return &pb.GetExchangeRatesResponse{Rates: rates}, nil
}

// ConvertAmount performs a currency conversion between two accounts.
// Flow (issue #75): always use selling_rate, apply commission per step, route through RSD.
func (s *ExchangeServer) ConvertAmount(ctx context.Context, req *pb.ConvertAmountRequest) (*pb.ConvertAmountResponse, error) {
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	if req.FromAccount == req.ToAccount {
		return nil, status.Error(codes.InvalidArgument, "from and to accounts must be different")
	}

	// 1. Resolve account currencies from account_db
	var fromCurrencyID, toCurrencyID int64
	var fromOwnerID int64
	var availableBalance float64

	if err := s.AccountDB.QueryRowContext(ctx,
		`SELECT owner_id, available_balance, currency_id FROM accounts WHERE account_number = $1`,
		req.FromAccount,
	).Scan(&fromOwnerID, &availableBalance, &fromCurrencyID); err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "source account not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load source account: %v", err)
	}

	if fromOwnerID != req.ClientId {
		return nil, status.Error(codes.PermissionDenied, "source account does not belong to client")
	}

	var toOwnerID int64
	if err := s.AccountDB.QueryRowContext(ctx,
		`SELECT owner_id, currency_id FROM accounts WHERE account_number = $1`,
		req.ToAccount,
	).Scan(&toOwnerID, &toCurrencyID); err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "destination account not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load destination account: %v", err)
	}

	// 2. Resolve currency codes
	var fromCode, toCode string
	if err := s.DB.QueryRowContext(ctx,
		`SELECT code FROM currencies WHERE id = $1`, fromCurrencyID,
	).Scan(&fromCode); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve source currency: %v", err)
	}
	if err := s.DB.QueryRowContext(ctx,
		`SELECT code FROM currencies WHERE id = $1`, toCurrencyID,
	).Scan(&toCode); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve destination currency: %v", err)
	}

	if fromCode == toCode {
		return nil, status.Error(codes.InvalidArgument, "accounts have the same currency; use a regular transfer")
	}

	// 3. Ensure today's rates exist
	if err := s.ensureTodayRates(ctx); err != nil {
		log.Printf("exchange-service: ensureTodayRates: %v", err)
	}

	// 4. Get selling rates for involved currencies
	getSellingRate := func(code string) (float64, error) {
		if code == "RSD" {
			return 1.0, nil
		}
		var r float64
		err := s.DB.QueryRowContext(ctx,
			`SELECT selling_rate FROM daily_exchange_rates WHERE currency_code = $1 AND date = CURRENT_DATE`,
			code,
		).Scan(&r)
		return r, err
	}

	fromSelling, err := getSellingRate(fromCode)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no exchange rate for %s today", fromCode)
	}
	toSelling, err := getSellingRate(toCode)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no exchange rate for %s today", toCode)
	}

	// 5. Calculate conversion (issue #75 rules: all via RSD, commission each step)
	var rsdAmount, toAmount, effectiveRate float64
	switch {
	case fromCode == "RSD":
		// RSD → Foreign: rsd / selling_rate(to) * (1 - commission)
		toAmount = (req.Amount / toSelling) * (1 - commission)
		effectiveRate = toSelling
	case toCode == "RSD":
		// Foreign → RSD: amount * selling_rate(from) * (1 - commission)
		toAmount = req.Amount * fromSelling * (1 - commission)
		effectiveRate = fromSelling
	default:
		// Foreign → Foreign (2 steps, commission each)
		rsdAmount = req.Amount * fromSelling * (1 - commission)
		toAmount = (rsdAmount / toSelling) * (1 - commission)
		effectiveRate = fromSelling / toSelling
	}

	toAmount = math.Round(toAmount*100) / 100
	_ = rsdAmount

	// 6. Check sufficient balance
	if availableBalance < req.Amount {
		return nil, status.Error(codes.FailedPrecondition, "insufficient funds")
	}

	commissionAmount := math.Round((req.Amount*commission)*100) / 100

	// 7. Update account balances in account_db transaction
	tx, err := s.AccountDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, `
		UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1
		WHERE account_number = $2`, req.Amount, req.FromAccount); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to debit source account: %v", err)
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1
		WHERE account_number = $2`, toAmount, req.ToAccount); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to credit destination account: %v", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	// 8. Record exchange transaction
	var txID int64
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO exchange_transactions
			(client_id, from_account, to_account, from_currency, to_currency,
			 from_amount, to_amount, rate, commission)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		req.ClientId, req.FromAccount, req.ToAccount, fromCode, toCode,
		req.Amount, toAmount,
		math.Round(effectiveRate*1e6)/1e6,
		commissionAmount,
	).Scan(&txID)
	if err != nil {
		log.Printf("exchange-service: failed to record transaction: %v", err)
	}

	return &pb.ConvertAmountResponse{
		FromCurrency:  fromCode,
		ToCurrency:    toCode,
		FromAmount:    req.Amount,
		ToAmount:      toAmount,
		Rate:          math.Round(effectiveRate*1e6) / 1e6,
		Commission:    commissionAmount,
		TransactionId: txID,
	}, nil
}

// GetExchangeHistory returns past exchange transactions for a client.
func (s *ExchangeServer) GetExchangeHistory(ctx context.Context, req *pb.GetExchangeHistoryRequest) (*pb.GetExchangeHistoryResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, from_account, to_account, from_currency, to_currency,
		       from_amount, to_amount, rate, commission, timestamp, status
		FROM exchange_transactions
		WHERE client_id = $1
		ORDER BY timestamp DESC`,
		req.ClientId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query exchange history: %v", err)
	}
	defer rows.Close()

	var txs []*pb.ExchangeTransaction
	for rows.Next() {
		var t pb.ExchangeTransaction
		var ts time.Time
		if err := rows.Scan(
			&t.Id, &t.FromAccount, &t.ToAccount, &t.FromCurrency, &t.ToCurrency,
			&t.FromAmount, &t.ToAmount, &t.Rate, &t.Commission, &ts, &t.Status,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan transaction: %v", err)
		}
		t.Timestamp = ts.Format(time.RFC3339)
		txs = append(txs, &t)
	}
	return &pb.GetExchangeHistoryResponse{Transactions: txs}, nil
}
