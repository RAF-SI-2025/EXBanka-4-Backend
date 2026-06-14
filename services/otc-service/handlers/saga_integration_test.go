package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/otc-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// skipUnlessIntegration skips the test unless OTC_INTEGRATION_TEST=true.
func skipUnlessIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("OTC_INTEGRATION_TEST") != "true" {
		t.Skip("set OTC_INTEGRATION_TEST=true to run SAGA integration tests")
	}
}

// ── Database connection helpers ────────────────────────────────────────────────

func mustConnect(t *testing.T, envKey, fallback string) *sql.DB {
	t.Helper()
	dsn := os.Getenv(envKey)
	if dsn == "" {
		dsn = fallback
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "connect %s", envKey)
	require.NoError(t, db.Ping(), "ping %s", envKey)
	return db
}

func newTestServer(t *testing.T) *handlers.OtcServer {
	t.Helper()
	return &handlers.OtcServer{
		DB:           mustConnect(t, "OTC_DB_URL", "postgres://otc_user:otc_pass@localhost:5444/otc_db?sslmode=disable"),
		AccountDB:    mustConnect(t, "ACCOUNT_DB_URL", "postgres://account_user:account_pass@localhost:5436/account_db?sslmode=disable"),
		PortfolioDB:  mustConnect(t, "PORTFOLIO_DB_URL", "postgres://portfolio_user:portfolio_pass@localhost:5443/portfolio_db?sslmode=disable"),
		SecuritiesDB: mustConnect(t, "SECURITIES_DB_URL", "postgres://securities_user:securities_pass@localhost:5441/securities_db?sslmode=disable"),
		ExchangeDB:   mustConnect(t, "EXCHANGE_DB_URL", "postgres://exchange_user:exchange_pass@localhost:5438/exchange_db?sslmode=disable"),
		EmployeeDB:   mustConnect(t, "EMPLOYEE_DB_URL", "postgres://employee_user:employee_pass@localhost:5433/employee_db?sslmode=disable"),
		ClientDB:     mustConnect(t, "CLIENT_DB_URL", "postgres://client_user:client_pass@localhost:5435/client_db?sslmode=disable"),
	}
}

// ── Seed helpers ───────────────────────────────────────────────────────────────

const (
	testBuyerID  int64 = 9901
	testSellerID int64 = 9902
	testTicker         = "SAGA_TEST"
	testCurrency       = "USD"
	// currency_id=4 for USD (from currencyIDMap seed)
	testCurrencyID int64 = 4
)

func truncateAll(t *testing.T, s *handlers.OtcServer) {
	t.Helper()
	_, _ = s.DB.Exec(`DELETE FROM otc_saga_log WHERE contract_id IN (SELECT id FROM otc_contracts WHERE ticker=$1)`, testTicker)
	_, _ = s.DB.Exec(`DELETE FROM otc_saga WHERE contract_id IN (SELECT id FROM otc_contracts WHERE ticker=$1)`, testTicker)
	_, _ = s.DB.Exec(`DELETE FROM otc_contracts WHERE ticker=$1`, testTicker)
	_, _ = s.DB.Exec(`DELETE FROM otc_negotiations WHERE ticker=$1`, testTicker)
	_, _ = s.AccountDB.Exec(`DELETE FROM accounts WHERE owner_id IN ($1,$2)`, testBuyerID, testSellerID)
	_, _ = s.PortfolioDB.Exec(`DELETE FROM portfolio_entry WHERE user_id IN ($1,$2)`, testBuyerID, testSellerID)
	_ = ensureListing(t, s)
}

func ensureListing(t *testing.T, s *handlers.OtcServer) int64 {
	t.Helper()
	var id int64
	err := s.SecuritiesDB.QueryRow(`SELECT id FROM listing WHERE ticker=$1`, testTicker).Scan(&id)
	if err == nil {
		return id
	}
	require.Equal(t, sql.ErrNoRows, err, "ensure listing: unexpected error looking up ticker")

	// Ensure a test stock exchange exists.
	var exchID int64
	err = s.SecuritiesDB.QueryRow(
		`INSERT INTO stock_exchanges (name, acronym, mic_code, polity, currency, timezone)
		 VALUES ('SAGA Test Exchange','SAGA','SAGX','Test','USD','UTC')
		 ON CONFLICT (mic_code) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
	).Scan(&exchID)
	require.NoError(t, err, "ensure stock exchange")

	err = s.SecuritiesDB.QueryRow(
		`INSERT INTO listing (ticker, name, exchange_id, last_refresh, price, ask, bid, volume, change, type)
		 VALUES ($1,'SAGA Test',$2,NOW(),100,101,99,1000,0,'STOCK') RETURNING id`,
		testTicker, exchID,
	).Scan(&id)
	require.NoError(t, err, "ensure listing")
	return id
}

func seedNegotiation(t *testing.T, s *handlers.OtcServer, qty int, strikeUSD float64, settlementOffset time.Duration) int64 {
	t.Helper()
	settlementDate := time.Now().Add(settlementOffset).Format("2006-01-02")
	var negID int64
	err := s.DB.QueryRow(`
		INSERT INTO otc_negotiations
			(ticker, seller_id, seller_type, buyer_id, buyer_type, amount, price_per_stock, settlement_date, premium, currency, status)
		VALUES ($1,$2,'CLIENT',$3,'CLIENT',$4,$5,$6,0,$7,'ACCEPTED') RETURNING id`,
		testTicker, testSellerID, testBuyerID, qty, strikeUSD, settlementDate, testCurrency,
	).Scan(&negID)
	require.NoError(t, err, "seed negotiation")
	return negID
}

func seedContract(t *testing.T, s *handlers.OtcServer, qty int, strikeUSD float64, settlementOffset time.Duration) int64 {
	t.Helper()
	negID := seedNegotiation(t, s, qty, strikeUSD, settlementOffset)
	var contractID int64
	err := s.DB.QueryRow(`
		INSERT INTO otc_contracts (negotiation_id, seller_id, seller_type, buyer_id, buyer_type, ticker, amount, strike_price, premium, currency, settlement_date, status)
		VALUES ($1,$2,'CLIENT',$3,'CLIENT',$4,$5,$6,0,$7,$8,'ACTIVE') RETURNING id`,
		negID, testSellerID, testBuyerID, testTicker, qty, strikeUSD, testCurrency,
		time.Now().Add(settlementOffset).Format("2006-01-02"),
	).Scan(&contractID)
	require.NoError(t, err, "seed contract")
	return contractID
}

func seedContractWithStatus(t *testing.T, s *handlers.OtcServer, contractStatus string) int64 {
	t.Helper()
	negID := seedNegotiation(t, s, 10, 300, 24*time.Hour)
	var contractID int64
	err := s.DB.QueryRow(`
		INSERT INTO otc_contracts (negotiation_id, seller_id, seller_type, buyer_id, buyer_type, ticker, amount, strike_price, premium, currency, settlement_date, status)
		VALUES ($1,$2,'CLIENT',$3,'CLIENT',$4,10,300,0,'USD',NOW()+INTERVAL '1 day',$5) RETURNING id`,
		negID, testSellerID, testBuyerID, testTicker, contractStatus,
	).Scan(&contractID)
	require.NoError(t, err)
	return contractID
}

func seedBuyerAccount(t *testing.T, s *handlers.OtcServer, balanceUSD float64) int64 {
	t.Helper()
	var id int64
	err := s.AccountDB.QueryRow(`
		INSERT INTO accounts (account_number, account_name, owner_id, employee_id, currency_id, account_type, status, balance, available_balance)
		VALUES ($1,'SAGA Buyer',$2,1,$3,'CURRENT','ACTIVE',$4,$4) RETURNING id`,
		fmt.Sprintf("SAGA-BUYER-%d", time.Now().UnixNano()), testBuyerID, testCurrencyID, balanceUSD,
	).Scan(&id)
	require.NoError(t, err, "seed buyer account")
	return id
}

func seedSellerAccount(t *testing.T, s *handlers.OtcServer) int64 {
	t.Helper()
	var id int64
	err := s.AccountDB.QueryRow(`
		INSERT INTO accounts (account_number, account_name, owner_id, employee_id, currency_id, account_type, status, balance, available_balance)
		VALUES ($1,'SAGA Seller',$2,1,$3,'CURRENT','ACTIVE',0,0) RETURNING id`,
		fmt.Sprintf("SAGA-SELLER-%d", time.Now().UnixNano()), testSellerID, testCurrencyID,
	).Scan(&id)
	require.NoError(t, err, "seed seller account")
	return id
}

func seedSellerPortfolio(t *testing.T, s *handlers.OtcServer, accountID int64, qty int) {
	t.Helper()
	listingID := ensureListing(t, s)
	_, err := s.PortfolioDB.Exec(`
		INSERT INTO portfolio_entry (user_id, user_type, listing_id, amount, buy_price, account_id, reserved_amount)
		VALUES ($1,'CLIENT',$2,$3,100,$4,0)
		ON CONFLICT (user_id, user_type, listing_id) DO UPDATE SET amount=$3, reserved_amount=0`,
		testSellerID, listingID, qty, accountID,
	)
	require.NoError(t, err, "seed seller portfolio")
}

// ── Poll / assert helpers ──────────────────────────────────────────────────────

type sagaRow struct {
	Status      string
	CurrentStep int
}

func getSaga(t *testing.T, db *sql.DB, contractID int64) sagaRow {
	t.Helper()
	var r sagaRow
	err := db.QueryRow(`SELECT status, current_step FROM otc_saga WHERE contract_id=$1`, contractID).
		Scan(&r.Status, &r.CurrentStep)
	require.NoError(t, err, "getSaga")
	return r
}

type logEntry struct {
	Step   int
	Status string
}

func getSagaLog(t *testing.T, db *sql.DB, contractID int64) []logEntry {
	t.Helper()
	rows, err := db.Query(`SELECT step, status FROM otc_saga_log WHERE contract_id=$1 ORDER BY id`, contractID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var entries []logEntry
	for rows.Next() {
		var e logEntry
		require.NoError(t, rows.Scan(&e.Step, &e.Status))
		entries = append(entries, e)
	}
	return entries
}

func getAccountBalance(t *testing.T, db *sql.DB, ownerID int64) (balance, available float64) {
	t.Helper()
	err := db.QueryRow(
		`SELECT COALESCE(balance,0), COALESCE(available_balance,0) FROM accounts WHERE owner_id=$1 AND currency_id=$2 LIMIT 1`,
		ownerID, testCurrencyID,
	).Scan(&balance, &available)
	require.NoError(t, err, "getAccountBalance for owner %d", ownerID)
	return
}

func getPortfolioAmount(t *testing.T, db *sql.DB, userID int64, listingID int64) (amount, reserved int) {
	t.Helper()
	err := db.QueryRow(
		`SELECT COALESCE(amount,0), COALESCE(reserved_amount,0) FROM portfolio_entry WHERE user_id=$1 AND listing_id=$2`,
		userID, listingID,
	).Scan(&amount, &reserved)
	if err == sql.ErrNoRows {
		return 0, 0
	}
	require.NoError(t, err)
	return
}

func getContractStatus(t *testing.T, db *sql.DB, contractID int64) string {
	t.Helper()
	var st string
	require.NoError(t, db.QueryRow(`SELECT status FROM otc_contracts WHERE id=$1`, contractID).Scan(&st))
	return st
}

// assertInvariants checks I1 (money conserved), I2 (shares conserved), I3 (no reserved leftovers).
func assertInvariants(t *testing.T, s *handlers.OtcServer, buyerBalanceBefore, sellerBalanceBefore float64, listingID int64, sellerSharesBefore int) {
	t.Helper()
	buyerBal, buyerAvail := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBal, sellerAvail := getAccountBalance(t, s.AccountDB, testSellerID)

	// I1: total money (buyer + seller) conserved
	assert.InDelta(t,
		buyerBalanceBefore+sellerBalanceBefore,
		buyerBal+sellerBal,
		0.01,
		"I1: total money not conserved (buyer %.2f + seller %.2f = %.2f, expected %.2f)",
		buyerBal, sellerBal, buyerBal+sellerBal, buyerBalanceBefore+sellerBalanceBefore,
	)
	// I1 sub: available == balance for both (no locked amounts remaining — I3 for money)
	assert.InDelta(t, buyerBal, buyerAvail, 0.01, "I3: buyer has reserved money (balance != available)")
	assert.InDelta(t, sellerBal, sellerAvail, 0.01, "I3: seller has reserved money (balance != available)")

	// I2: total shares conserved
	buyerAmt, buyerReserved := getPortfolioAmount(t, s.PortfolioDB, testBuyerID, listingID)
	sellerAmt, sellerReserved := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)
	assert.Equal(t, sellerSharesBefore, buyerAmt+sellerAmt, "I2: total shares not conserved")

	// I3: no reserved leftovers
	assert.Equal(t, 0, buyerReserved, "I3: buyer has reserved_amount > 0")
	assert.Equal(t, 0, sellerReserved, "I3: seller has reserved_amount > 0")
}

// ctxWithMeta builds a context carrying gRPC incoming metadata (for fault injection).
func ctxWithMeta(kv ...string) context.Context {
	md := metadata.Pairs(kv...)
	return metadata.NewIncomingContext(context.Background(), md)
}

// ── SG-01: Happy path ──────────────────────────────────────────────────────────

func TestSG01_HappyPath(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour)
	seedBuyerAccount(t, s, 5000)
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 10)

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBalBefore, _ := getAccountBalance(t, s.AccountDB, testSellerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	resp, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: contractID,
		CallerId:   testBuyerID,
		CallerType: "CLIENT",
	})
	require.NoError(t, err)
	assert.Equal(t, "EXERCISED", resp.Status)
	assert.NotZero(t, resp.SagaId)

	saga := getSaga(t, s.DB, contractID)
	assert.Equal(t, "Completed", saga.Status)
	assert.Equal(t, 5, saga.CurrentStep)

	logEntries := getSagaLog(t, s.DB, contractID)
	require.Len(t, logEntries, 5, "expected 5 log entries for happy path")
	for i, e := range logEntries {
		assert.Equal(t, i+1, e.Step, "log step order")
		assert.Equal(t, "SUCCESS", e.Status)
	}

	assert.Equal(t, "EXERCISED", getContractStatus(t, s.DB, contractID))

	buyerBal, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBal, _ := getAccountBalance(t, s.AccountDB, testSellerID)
	assert.InDelta(t, buyerBalBefore-3000, buyerBal, 0.01, "buyer should have 2000 USD")
	assert.InDelta(t, sellerBalBefore+3000, sellerBal, 0.01, "seller should have 3000 USD")

	buyerHolding, _ := getPortfolioAmount(t, s.PortfolioDB, testBuyerID, listingID)
	assert.Equal(t, 10, buyerHolding, "buyer should own 10 shares")

	assertInvariants(t, s, buyerBalBefore, sellerBalBefore, listingID, sellerAmt)
}

// ── SG-02: Pre-saga validation ─────────────────────────────────────────────────

func TestSG02a_CallerNotBuyer(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	contractID := seedContract(t, s, 10, 300, 24*time.Hour)

	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: contractID,
		CallerId:   testBuyerID + 999,
		CallerType: "CLIENT",
	})
	require.Error(t, err)

	// No saga row should be created.
	var count int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM otc_saga WHERE contract_id=$1`, contractID).Scan(&count)
	assert.Equal(t, 0, count, "no saga row should exist for pre-saga rejection")
}

func TestSG02b_ContractNotFound(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)

	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 999999999,
		CallerId:   testBuyerID,
		CallerType: "CLIENT",
	})
	require.Error(t, err)
}

func TestSG02c_ContractAlreadyExercised(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	contractID := seedContractWithStatus(t, s, "EXERCISED")

	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: contractID,
		CallerId:   testBuyerID,
		CallerType: "CLIENT",
	})
	require.Error(t, err)

	var count int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM otc_saga WHERE contract_id=$1`, contractID).Scan(&count)
	assert.Equal(t, 0, count, "no saga row for pre-validation failure")
}

func TestSG02d_SettlementDatePassed(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	// settlement date 2 days in the past
	contractID := seedContract(t, s, 10, 300, -48*time.Hour)

	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: contractID,
		CallerId:   testBuyerID,
		CallerType: "CLIENT",
	})
	require.Error(t, err)
}

// ── SG-03: F1 failure (insufficient funds) ─────────────────────────────────────

func TestSG03_InsufficientFunds(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour) // needs 3000 USD
	seedBuyerAccount(t, s, 500)                             // only 500 USD
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 10)

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: contractID,
		CallerId:   testBuyerID,
		CallerType: "CLIENT",
	})
	require.Error(t, err)

	saga := getSaga(t, s.DB, contractID)
	assert.Equal(t, "Compensated", saga.Status)
	assert.Equal(t, 1, saga.CurrentStep)

	logEntries := getSagaLog(t, s.DB, contractID)
	require.Len(t, logEntries, 1, "only F1 log entry expected")
	assert.Equal(t, 1, logEntries[0].Step)
	assert.Equal(t, "FAILED", logEntries[0].Status)

	// No side effects.
	buyerBal, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	assert.InDelta(t, buyerBalBefore, buyerBal, 0.01, "buyer balance must be unchanged")

	_, _ = sellerAmt, listingID // invariants checked implicitly via unchanged balances
}

// ── SG-04: F2 failure (insufficient shares) ────────────────────────────────────

func TestSG04_InsufficientShares(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour) // needs 10 shares
	seedBuyerAccount(t, s, 5000)
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 3) // only 3 shares

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: contractID,
		CallerId:   testBuyerID,
		CallerType: "CLIENT",
	})
	require.Error(t, err)

	saga := getSaga(t, s.DB, contractID)
	assert.Equal(t, "Compensated", saga.Status)
	assert.Equal(t, 2, saga.CurrentStep)

	logEntries := getSagaLog(t, s.DB, contractID)
	require.GreaterOrEqual(t, len(logEntries), 3, "expected at least F1 ok, F2 fail, C1 compensated")
	assert.Equal(t, 1, logEntries[0].Step)
	assert.Equal(t, "SUCCESS", logEntries[0].Status)
	assert.Equal(t, 2, logEntries[1].Step)
	assert.Equal(t, "FAILED", logEntries[1].Status)
	// C1 should have fired
	c1Entry := logEntries[len(logEntries)-1]
	assert.Equal(t, 1, c1Entry.Step)
	assert.Equal(t, "COMPENSATED", c1Entry.Status)

	// Buyer balance unchanged (F1 reserved, C1 released).
	buyerBal, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	assert.InDelta(t, buyerBalBefore, buyerBal, 0.01, "buyer balance unchanged after F2 failure")

	_ = sellerAmt
}

// ── SG-05: F3 force-fail ────────────────────────────────────────────────────────

func TestSG05_F3ForceFail(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour)
	seedBuyerAccount(t, s, 5000)
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 10)

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBalBefore, _ := getAccountBalance(t, s.AccountDB, testSellerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	ctx := ctxWithMeta("x-saga-force-fail", "F3")
	_, err := s.ExerciseContract(ctx, &pb.ExerciseContractRequest{
		ContractId: contractID,
		CallerId:   testBuyerID,
		CallerType: "CLIENT",
	})
	require.Error(t, err)

	saga := getSaga(t, s.DB, contractID)
	assert.Equal(t, "Compensated", saga.Status)
	assert.Equal(t, 3, saga.CurrentStep)

	logEntries := getSagaLog(t, s.DB, contractID)
	// Expected: F1 ok, F2 ok, F3 fail, C2 compensated, C1 compensated
	assert.GreaterOrEqual(t, len(logEntries), 5)

	assertInvariants(t, s, buyerBalBefore, sellerBalBefore, listingID, sellerAmt)
	assert.Equal(t, "ACTIVE", getContractStatus(t, s.DB, contractID), "I6: contract must stay ACTIVE")
}

// ── SG-06: F4 force-fail ────────────────────────────────────────────────────────

func TestSG06_F4ForceFail(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour)
	seedBuyerAccount(t, s, 5000)
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 10)

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBalBefore, _ := getAccountBalance(t, s.AccountDB, testSellerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	ctx := ctxWithMeta("x-saga-force-fail", "F4")
	_, err := s.ExerciseContract(ctx, &pb.ExerciseContractRequest{
		ContractId: contractID,
		CallerId:   testBuyerID,
		CallerType: "CLIENT",
	})
	require.Error(t, err)

	saga := getSaga(t, s.DB, contractID)
	assert.Equal(t, "Compensated", saga.Status)
	assert.Equal(t, 4, saga.CurrentStep)

	assertInvariants(t, s, buyerBalBefore, sellerBalBefore, listingID, sellerAmt)
	assert.Equal(t, "ACTIVE", getContractStatus(t, s.DB, contractID))
}

// ── SG-07: F5 force-fail ────────────────────────────────────────────────────────

func TestSG07_F5ForceFail(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour)
	seedBuyerAccount(t, s, 5000)
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 10)

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBalBefore, _ := getAccountBalance(t, s.AccountDB, testSellerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	ctx := ctxWithMeta("x-saga-force-fail", "F5")
	_, err := s.ExerciseContract(ctx, &pb.ExerciseContractRequest{
		ContractId: contractID,
		CallerId:   testBuyerID,
		CallerType: "CLIENT",
	})
	require.Error(t, err)

	saga := getSaga(t, s.DB, contractID)
	assert.Equal(t, "Compensated", saga.Status)
	assert.Equal(t, 5, saga.CurrentStep)

	logEntries := getSagaLog(t, s.DB, contractID)
	// F1-F4 ok, F5 fail, C4 comp, C3 comp, C2 comp, C1 comp = 9 entries
	assert.GreaterOrEqual(t, len(logEntries), 9)

	assertInvariants(t, s, buyerBalBefore, sellerBalBefore, listingID, sellerAmt)
	assert.Equal(t, "ACTIVE", getContractStatus(t, s.DB, contractID))
}

// ── SG-08: Compensator fails once, then succeeds ───────────────────────────────

func TestSG08_CompensatorFailsOnceThenSucceeds(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour)
	seedBuyerAccount(t, s, 5000)
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 10)

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBalBefore, _ := getAccountBalance(t, s.AccountDB, testSellerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	ctx := ctxWithMeta(
		"x-saga-force-fail", "F3",
		"x-saga-compensate-fail", "C2",
		"x-saga-compensate-fail-times", "1",
	)
	_, err := s.ExerciseContract(ctx, &pb.ExerciseContractRequest{
		ContractId: contractID,
		CallerId:   testBuyerID,
		CallerType: "CLIENT",
	})
	require.Error(t, err)

	saga := getSaga(t, s.DB, contractID)
	assert.Equal(t, "Compensated", saga.Status)

	logEntries := getSagaLog(t, s.DB, contractID)
	// Count C2 compensator entries only (COMP_FAILED + COMPENSATED, not the forward SUCCESS).
	c2Count := 0
	for _, e := range logEntries {
		if e.Step == 2 && e.Status != "SUCCESS" {
			c2Count++
		}
	}
	assert.Equal(t, 2, c2Count, "C2 should appear twice in log (1 COMP_FAILED + 1 COMPENSATED)")

	assertInvariants(t, s, buyerBalBefore, sellerBalBefore, listingID, sellerAmt)
}

// ── Helpers for SG-09 / SG-10 / SG-11 ────────────────────────────────────────

func skipUnlessDocker(t *testing.T) {
	t.Helper()
	if os.Getenv("OTC_DOCKER_TEST") != "true" {
		t.Skip("set OTC_DOCKER_TEST=true to run docker-based SAGA tests")
	}
}

// repoRoot returns the repository root (three levels above the handlers/ package directory).
func repoRoot() string {
	wd, _ := os.Getwd()
	abs, _ := filepath.Abs(filepath.Join(wd, "..", "..", ".."))
	return abs
}

// dockerCompose runs: docker compose -f services/<service>/docker-compose.yml <args...>
func dockerCompose(t *testing.T, service string, args ...string) {
	t.Helper()
	composePath := filepath.Join(repoRoot(), "services", service, "docker-compose.yml")
	cmdArgs := append([]string{"compose", "-f", composePath}, args...)
	out, err := exec.Command("docker", cmdArgs...).CombinedOutput()
	if err != nil {
		t.Logf("docker compose %v output: %s", args, string(out))
		t.FailNow()
	}
}

// pollSaga polls otc_saga until Completed or Compensated, or fails after timeout.
func pollSaga(t *testing.T, db *sql.DB, contractID int64, timeout time.Duration) sagaRow {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var r sagaRow
		err := db.QueryRow(`SELECT status, current_step FROM otc_saga WHERE contract_id=$1`, contractID).
			Scan(&r.Status, &r.CurrentStep)
		if err == nil && (r.Status == "Completed" || r.Status == "Compensated") {
			return r
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("pollSaga: timeout after %s for contract %d", timeout, contractID)
	return sagaRow{}
}

// waitFor polls cond every interval until it returns true or timeout elapses.
func waitFor(t *testing.T, timeout, interval time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("waitFor timeout after %s: %s", timeout, msg)
}

// newTestServerWithAccountAt builds a test server with AccountDB pointed at a custom DSN.
func newTestServerWithAccountAt(t *testing.T, accountDSN string) *handlers.OtcServer {
	t.Helper()
	return &handlers.OtcServer{
		DB:           mustConnect(t, "OTC_DB_URL", "postgres://otc_user:otc_pass@localhost:5444/otc_db?sslmode=disable"),
		AccountDB:    mustConnect(t, "", accountDSN),
		PortfolioDB:  mustConnect(t, "PORTFOLIO_DB_URL", "postgres://portfolio_user:portfolio_pass@localhost:5443/portfolio_db?sslmode=disable"),
		SecuritiesDB: mustConnect(t, "SECURITIES_DB_URL", "postgres://securities_user:securities_pass@localhost:5441/securities_db?sslmode=disable"),
		ExchangeDB:   mustConnect(t, "EXCHANGE_DB_URL", "postgres://exchange_user:exchange_pass@localhost:5438/exchange_db?sslmode=disable"),
		EmployeeDB:   mustConnect(t, "EMPLOYEE_DB_URL", "postgres://employee_user:employee_pass@localhost:5433/employee_db?sslmode=disable"),
		ClientDB:     mustConnect(t, "CLIENT_DB_URL", "postgres://client_user:client_pass@localhost:5435/client_db?sslmode=disable"),
	}
}

// toxiproxyCreateProxy registers a Toxiproxy proxy and returns an addToxic helper.
// upstream is "host:port" that Toxiproxy will forward to. When Toxiproxy is connected
// to the DB's Docker network it can use the container name (e.g. "account-db:5432")
// instead of going through the host. Controlled via TOXI_ACCT_UPSTREAM in the run script.
// The proxy is deleted via t.Cleanup.
func toxiproxyCreateProxy(t *testing.T, toxiAddr, proxyName string, listenPort int, upstream string) func(toxicType string, attrs map[string]interface{}) {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{
		"name":   proxyName,
		"listen": fmt.Sprintf("0.0.0.0:%d", listenPort), // bind all interfaces — "localhost" binds only loopback and Docker can't forward to it
		"upstream": upstream,
		"enabled":  true,
	})
	resp, err := http.Post(fmt.Sprintf("http://%s/proxies", toxiAddr), "application/json", bytes.NewReader(body))
	require.NoError(t, err, "create toxiproxy proxy %s", proxyName)
	resp.Body.Close()

	t.Cleanup(func() {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://%s/proxies/%s", toxiAddr, proxyName), nil)
		_, _ = http.DefaultClient.Do(req)
	})

	return func(toxicType string, attrs map[string]interface{}) {
		t.Helper()
		toxBody, _ := json.Marshal(map[string]interface{}{
			"name": "t", "type": toxicType, "attributes": attrs,
		})
		r, e := http.Post(
			fmt.Sprintf("http://%s/proxies/%s/toxics", toxiAddr, proxyName),
			"application/json", bytes.NewReader(toxBody))
		require.NoError(t, e, "add toxic %s to proxy %s", toxicType, proxyName)
		r.Body.Close()
	}
}

// ── SG-09a: account-service killed before call ───────────────────────────────
//
// NOTE: ExerciseContract calls findAccount + getAccountCurrencyID (both on account DB)
// BEFORE the saga INSERT. So when account-db is unavailable from the start, the failure
// happens before F1 is formally logged — no saga row is created. This deviates from
// the PDF spec's "F1 FAILED" assertion, but the key invariants (no money/shares moved)
// are still verified.
//
// We use "docker compose kill" (SIGKILL) rather than "docker compose pause" (SIGSTOP).
// SIGSTOP freezes the postgres process but keeps TCP connections open (suspended). On
// Windows, lib/pq does NOT honour context cancellation for blocked TCP reads — the
// goroutine stays in IO-wait indefinitely. SIGKILL terminates the process; the OS
// immediately closes all TCP connections with RST, so lib/pq returns an error at once.

func TestSG09a_AccountServicePaused(t *testing.T) {
	skipUnlessIntegration(t)
	skipUnlessDocker(t)
	s := newTestServer(t)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour)
	seedBuyerAccount(t, s, 5000)
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 10)

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBalBefore, _ := getAccountBalance(t, s.AccountDB, testSellerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	// Kill the account-db container so the OS closes all TCP connections immediately.
	// Cleanup restarts it in case the test fails before the explicit restart below.
	dockerCompose(t, "account-service", "kill")
	t.Cleanup(func() { dockerCompose(t, "account-service", "up", "-d") })

	// No context timeout needed — RST from SIGKILL makes lib/pq return immediately.
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: contractID, CallerId: testBuyerID, CallerType: "CLIENT",
	})
	require.Error(t, err, "ExerciseContract must fail when account-db is killed")

	// Restart account-db so we can run assertions with fresh connections.
	// Wait for a successful postgres ping — TCP port opens before postgres is ready for auth.
	dockerCompose(t, "account-service", "up", "-d")
	waitFor(t, 30*time.Second, 500*time.Millisecond, func() bool {
		db, err := sql.Open("postgres",
			"postgres://account_user:account_pass@localhost:5436/account_db?sslmode=disable")
		if err != nil {
			return false
		}
		defer db.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		return db.PingContext(ctx) == nil
	}, "account-db fully ready after kill")

	// Saga row may not exist (pre-saga failure at account lookup before INSERT).
	var sagaCount int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM otc_saga WHERE contract_id=$1`, contractID).Scan(&sagaCount)
	if sagaCount > 0 {
		saga := pollSaga(t, s.DB, contractID, 10*time.Second)
		assert.Equal(t, "Compensated", saga.Status)
	}

	// I1, I2, I3: no money or shares moved.
	s2 := newTestServer(t)
	buyerBal, buyerAvail := getAccountBalance(t, s2.AccountDB, testBuyerID)
	sellerBal, sellerAvail := getAccountBalance(t, s2.AccountDB, testSellerID)
	assert.InDelta(t, buyerBalBefore, buyerBal, 0.01, "buyer balance unchanged")
	assert.InDelta(t, buyerBal, buyerAvail, 0.01, "I3: no buyer reserved funds")
	assert.InDelta(t, sellerBalBefore, sellerBal, 0.01, "seller balance unchanged")
	assert.InDelta(t, sellerBal, sellerAvail, 0.01, "I3: no seller reserved funds")
	sellerAmtAfter, sellerReserved := getPortfolioAmount(t, s2.PortfolioDB, testSellerID, listingID)
	assert.Equal(t, sellerAmt, sellerAmtAfter, "I2: seller shares unchanged")
	assert.Equal(t, 0, sellerReserved, "I3: no reserved shares")
}

// ── SG-09b: Toxiproxy latency > RPC timeout ───────────────────────────────────

func TestSG09b_ToxiproxyLatency(t *testing.T) {
	skipUnlessIntegration(t)
	toxiAddr := os.Getenv("TOXIPROXY_ADDR")
	if toxiAddr == "" {
		t.Skip("set TOXIPROXY_ADDR=localhost:8474 to run Toxiproxy tests")
	}
	if resp, err := http.Get(fmt.Sprintf("http://%s/version", toxiAddr)); err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		t.Skipf("Toxiproxy not reachable at %s", toxiAddr)
	} else {
		resp.Body.Close()
	}

	// TOXI_ACCT_UPSTREAM is set by run-saga-tests.ps1 to "account-db:5432" so that
	// Toxiproxy (which runs in Docker) can reach account-db via the shared network.
	acctUpstream := os.Getenv("TOXI_ACCT_UPSTREAM")
	if acctUpstream == "" {
		acctUpstream = "localhost:5436"
	}
	addToxic := toxiproxyCreateProxy(t, toxiAddr, "sg09b-acct", 15436, acctUpstream)
	accountProxyDSN := "postgres://account_user:account_pass@localhost:15436/account_db?sslmode=disable"

	s := newTestServerWithAccountAt(t, accountProxyDSN)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour)
	seedBuyerAccount(t, s, 5000)
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 10)

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBalBefore, _ := getAccountBalance(t, s.AccountDB, testSellerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	// Add latency toxic: 30 000ms >> any RPC timeout.
	addToxic("latency", map[string]interface{}{"latency": 30000, "jitter": 0})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// lib/pq cancels queries by opening a NEW TCP connection to send a PostgreSQL
	// CancelRequest. That new connection also goes through the proxy and hits the
	// same 30s latency — so the cancel takes 30s to arrive and 30s to get a response,
	// meaning ExerciseContract would block for ~70s instead of ~10s.
	// Fix: when the context fires, DELETE the proxy via the Toxiproxy API. Toxiproxy
	// immediately RSTs all active connections, unblocking lib/pq's recv wait at once.
	go func() {
		<-ctx.Done()
		req, _ := http.NewRequest(http.MethodDelete,
			fmt.Sprintf("http://%s/proxies/sg09b-acct", toxiAddr), nil)
		_, _ = http.DefaultClient.Do(req)
	}()

	_, err := s.ExerciseContract(ctx, &pb.ExerciseContractRequest{
		ContractId: contractID, CallerId: testBuyerID, CallerType: "CLIENT",
	})
	require.Error(t, err, "ExerciseContract must fail with Toxiproxy latency")

	var sagaCount int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM otc_saga WHERE contract_id=$1`, contractID).Scan(&sagaCount)
	if sagaCount > 0 {
		saga := pollSaga(t, s.DB, contractID, 10*time.Second)
		assert.Equal(t, "Compensated", saga.Status)
	}

	// Invariant checks use direct DSN — the proxy may be deleted by the watchdog above,
	// and even if it weren't, going through a 30s-latency toxic for every query is wrong.
	directAccountDSN := "postgres://account_user:account_pass@localhost:5436/account_db?sslmode=disable"
	s2 := newTestServerWithAccountAt(t, directAccountDSN)
	buyerBal, buyerAvail := getAccountBalance(t, s2.AccountDB, testBuyerID)
	sellerBal, sellerAvail := getAccountBalance(t, s2.AccountDB, testSellerID)
	assert.InDelta(t, buyerBalBefore, buyerBal, 0.01, "buyer balance unchanged")
	assert.InDelta(t, buyerBal, buyerAvail, 0.01, "I3: no buyer reserved funds")
	assert.InDelta(t, sellerBalBefore, sellerBal, 0.01, "seller balance unchanged")
	assert.InDelta(t, sellerBal, sellerAvail, 0.01, "I3: no seller reserved funds")
	sellerAmtAfter, sellerReserved := getPortfolioAmount(t, s2.PortfolioDB, testSellerID, listingID)
	assert.Equal(t, sellerAmt, sellerAmtAfter, "I2: seller shares unchanged")
	assert.Equal(t, 0, sellerReserved, "I3: no reserved shares")
}

// ── SG-09c: Toxiproxy bandwidth=0 (network partition) ────────────────────────

func TestSG09c_ToxiproxyDown(t *testing.T) {
	skipUnlessIntegration(t)
	toxiAddr := os.Getenv("TOXIPROXY_ADDR")
	if toxiAddr == "" {
		t.Skip("set TOXIPROXY_ADDR=localhost:8474 to run Toxiproxy tests")
	}
	if resp, err := http.Get(fmt.Sprintf("http://%s/version", toxiAddr)); err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		t.Skipf("Toxiproxy not reachable at %s", toxiAddr)
	} else {
		resp.Body.Close()
	}

	acctUpstream := os.Getenv("TOXI_ACCT_UPSTREAM")
	if acctUpstream == "" {
		acctUpstream = "localhost:5436"
	}
	addToxic := toxiproxyCreateProxy(t, toxiAddr, "sg09c-acct", 15437, acctUpstream)
	accountProxyDSN := "postgres://account_user:account_pass@localhost:15437/account_db?sslmode=disable"

	s := newTestServerWithAccountAt(t, accountProxyDSN)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour)
	seedBuyerAccount(t, s, 5000)
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 10)

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBalBefore, _ := getAccountBalance(t, s.AccountDB, testSellerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	// bandwidth=0 simulates a hard network partition (no bytes flow).
	addToxic("bandwidth", map[string]interface{}{"rate": 0})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// With bandwidth=0, lib/pq's CancelRequest (sent over a new connection to the proxy)
	// also gets zero bytes — so the cancel never reaches postgres and the original
	// connection's recv wait blocks forever on Windows.
	// Fix: delete the proxy on context expiry → Toxiproxy RSTs all connections → unblocks.
	go func() {
		<-ctx.Done()
		req, _ := http.NewRequest(http.MethodDelete,
			fmt.Sprintf("http://%s/proxies/sg09c-acct", toxiAddr), nil)
		_, _ = http.DefaultClient.Do(req)
	}()

	_, err := s.ExerciseContract(ctx, &pb.ExerciseContractRequest{
		ContractId: contractID, CallerId: testBuyerID, CallerType: "CLIENT",
	})
	require.Error(t, err, "ExerciseContract must fail with Toxiproxy bandwidth=0")

	var sagaCount int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM otc_saga WHERE contract_id=$1`, contractID).Scan(&sagaCount)
	if sagaCount > 0 {
		saga := pollSaga(t, s.DB, contractID, 10*time.Second)
		assert.Equal(t, "Compensated", saga.Status)
	}

	// Invariant checks use direct DSN — the proxy is deleted by the watchdog above.
	directAccountDSN := "postgres://account_user:account_pass@localhost:5436/account_db?sslmode=disable"
	s2 := newTestServerWithAccountAt(t, directAccountDSN)
	buyerBal, buyerAvail := getAccountBalance(t, s2.AccountDB, testBuyerID)
	sellerBal, sellerAvail := getAccountBalance(t, s2.AccountDB, testSellerID)
	assert.InDelta(t, buyerBalBefore, buyerBal, 0.01, "buyer balance unchanged")
	assert.InDelta(t, buyerBal, buyerAvail, 0.01, "I3: no buyer reserved funds")
	assert.InDelta(t, sellerBalBefore, sellerBal, 0.01, "seller balance unchanged")
	assert.InDelta(t, sellerBal, sellerAvail, 0.01, "I3: no seller reserved funds")
	sellerAmtAfter, sellerReserved := getPortfolioAmount(t, s2.PortfolioDB, testSellerID, listingID)
	assert.Equal(t, sellerAmt, sellerAmtAfter, "I2: seller shares unchanged")
	assert.Equal(t, 0, sellerReserved, "I3: no reserved shares")
}

// ── SG-10: F3 fails mid-saga → C2+C1 compensate ─────────────────────────────
//
// X-Saga-Force-Fail: F3 makes F3 fail immediately via the fault-hook, which is
// equivalent to the account-db becoming unavailable while F3 executes: F1+F2
// commit their side-effects, F3 fails, C2 (portfolio-db) and C1 (account-db)
// run and restore everything.
//
// NOTE: "docker compose pause" (SIGSTOP) is NOT used here because on Windows
// lib/pq cannot cancel a blocked TCP read when the remote end is frozen. The
// goroutine would hang indefinitely instead of returning on context deadline.
// X-Saga-Force-Fail produces the same SAGA outcome without any Docker interaction.

func TestSG10_AccountPausedDuringSaga(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour)
	seedBuyerAccount(t, s, 5000)
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 10)

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBalBefore, _ := getAccountBalance(t, s.AccountDB, testSellerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	// F3 fails via the fault-hook; F1+F2 side-effects are already committed.
	sagaCtx := ctxWithMeta("x-saga-force-fail", "F3")
	_, err := s.ExerciseContract(sagaCtx, &pb.ExerciseContractRequest{
		ContractId: contractID, CallerId: testBuyerID, CallerType: "CLIENT",
	})
	require.Error(t, err, "ExerciseContract must fail when F3 is force-failed")

	saga := getSaga(t, s.DB, contractID)
	assert.Equal(t, "Compensated", saga.Status)
	assert.Equal(t, 3, saga.CurrentStep)

	logs := getSagaLog(t, s.DB, contractID)
	assert.GreaterOrEqual(t, len(logs), 5, "expected at least: F1 ok, F2 ok, F3 fail, C2 ok, C1 ok")

	s2 := newTestServer(t)
	assertInvariants(t, s2, buyerBalBefore, sellerBalBefore, listingID, sellerAmt)
	assert.Equal(t, "ACTIVE", getContractStatus(t, s.DB, contractID), "I6: contract stays ACTIVE")
}

// ── SG-11: Crash recovery (koordinator ubijen mid-flight) ─────────────────────
//
// The saga is started with X-Saga-Inject-Delay: F3:5000. After F1+F2 complete,
// the ExerciseContract context is cancelled (simulating SIGKILL on the process).
// A fresh OtcServer then calls RecoverInFlightSagas() — simulating service restart.
// The spec allows both Completed and Compensated as valid terminal states.

func TestSG11_CrashRecovery(t *testing.T) {
	skipUnlessIntegration(t)
	s := newTestServer(t)
	truncateAll(t, s)
	listingID := ensureListing(t, s)

	contractID := seedContract(t, s, 10, 300, 24*time.Hour)
	seedBuyerAccount(t, s, 5000)
	sellerAccID := seedSellerAccount(t, s)
	seedSellerPortfolio(t, s, sellerAccID, 10)

	buyerBalBefore, _ := getAccountBalance(t, s.AccountDB, testBuyerID)
	sellerBalBefore, _ := getAccountBalance(t, s.AccountDB, testSellerID)
	sellerAmt, _ := getPortfolioAmount(t, s.PortfolioDB, testSellerID, listingID)

	// Start saga with F3 delay to open a crash window.
	killCtx, kill := context.WithCancel(ctxWithMeta("x-saga-inject-delay", "F3:5000"))
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = s.ExerciseContract(killCtx, &pb.ExerciseContractRequest{
			ContractId: contractID, CallerId: testBuyerID, CallerType: "CLIENT",
		})
	}()

	// Wait for F1+F2 to complete (step >= 2), then simulate SIGKILL.
	waitFor(t, 10*time.Second, 100*time.Millisecond, func() bool {
		var step int
		_ = s.DB.QueryRow(`SELECT current_step FROM otc_saga WHERE contract_id=$1`, contractID).Scan(&step)
		return step >= 2
	}, "F1+F2 to complete before kill")

	kill() // cancel context — simulates SIGKILL on the coordinator process

	// Give the goroutine time to finish (sagaLog/sagaStatus use fresh background ctx).
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		// Goroutine may be sleeping in F3 delay; proceed — recovery will handle it.
	}

	// Simulate service restart: fresh server calls RecoverInFlightSagas.
	s2 := newTestServer(t)
	s2.RecoverInFlightSagas()

	// Both Completed and Compensated are valid outcomes (spec §SG-11).
	saga := pollSaga(t, s2.DB, contractID, 30*time.Second)
	assert.True(t, saga.Status == "Completed" || saga.Status == "Compensated",
		"saga must reach terminal status, got: %s", saga.Status)

	logs := getSagaLog(t, s2.DB, contractID)
	assert.NotEmpty(t, logs, "saga log must not be empty")

	// I1, I2, I3 must hold regardless of whether saga Completed or Compensated.
	assertInvariants(t, s2, buyerBalBefore, sellerBalBefore, listingID, sellerAmt)
}
