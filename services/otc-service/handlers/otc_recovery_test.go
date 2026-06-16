package handlers

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ── recoverSaga: compensation steps ──────────────────────────────────────────

func TestRecoverSaga_Step1Compensation(t *testing.T) {
	s, mOTC, _, _, mAcc, _, _, _ := newOtcServerWithExchange(t)
	// Mark Compensating
	mOTC.ExpectExec("UPDATE otc_saga SET status='Compensating'").WillReturnResult(sqlmock.NewResult(1, 1))
	// Completed steps: [1]
	mOTC.ExpectQuery("SELECT DISTINCT step FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step"}).AddRow(1))
	// Load contract (RSD so same-currency conversion, no ExchangeDB needed)
	mOTC.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, ticker, amount, strike_price, currency").
		WillReturnRows(sqlmock.NewRows([]string{
			"seller_id", "seller_type", "buyer_id", "buyer_type", "ticker", "amount", "strike_price", "currency",
		}).AddRow(int64(1), "CLIENT", int64(2), "CLIENT", "AAPL", int32(10), 50.0, "RSD"))
	// findAccount for buyer (currencyID=1 for RSD)
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))
	// getAccountCurrencyID for buyer
	mAcc.ExpectQuery("SELECT currency_id FROM accounts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(1)))
	// convertAmount RSD→RSD: same ID, no DB query needed
	// findAccount for seller
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(20)))
	// listingIDForTicker
	mOTC.ExpectQuery("SELECT id FROM listing WHERE ticker").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1 compensation: UPDATE accounts available_balance
	mAcc.ExpectExec("UPDATE accounts SET available_balance").WillReturnResult(sqlmock.NewResult(1, 1))
	// sagaLogRecovery: INSERT into otc_saga_log
	mOTC.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Final update: saga Compensated
	mOTC.ExpectExec("UPDATE otc_saga SET status='Compensated'").WillReturnResult(sqlmock.NewResult(1, 1))

	s.recoverSaga(1, 1)
}

func TestRecoverSaga_BuyerAccountNotFound(t *testing.T) {
	s, mOTC, _, _, mAcc, _, _, _ := newOtcServerWithExchange(t)
	mOTC.ExpectExec("UPDATE otc_saga SET status='Compensating'").WillReturnResult(sqlmock.NewResult(1, 1))
	mOTC.ExpectQuery("SELECT DISTINCT step FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step"}).AddRow(1))
	mOTC.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, ticker, amount, strike_price, currency").
		WillReturnRows(sqlmock.NewRows([]string{
			"seller_id", "seller_type", "buyer_id", "buyer_type", "ticker", "amount", "strike_price", "currency",
		}).AddRow(int64(1), "CLIENT", int64(2), "CLIENT", "AAPL", int32(10), 50.0, "RSD"))
	// findAccount for buyer fails
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"})) // not found

	s.recoverSaga(1, 1) // should not panic, just log
}

// ── exerciseCrossBank: deeper paths ──────────────────────────────────────────

func TestExerciseCrossBank_FindBuyerAccountFails(t *testing.T) {
	s, mOTC, _, _, mAcc, _, _, _ := newOtcServerWithExchange(t)
	// routing query returns valid data
	mOTC.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"seller_routing_number", "partner_negotiation_id", "seller_external_id"}).
			AddRow(int32(444), "neg-42", "ext-7"))
	// BuyerAccountId=0 → findAccount called, returns nothing
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err := s.exerciseCrossBank(context.Background(), &pb.ExerciseContractRequest{ContractId: 1},
		nil, 1, 2, "CLIENT", 10, 100.0, "RSD", "AAPL", futureSettlementDate())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find buyer account")
}

func TestExerciseCrossBank_GetBuyerCurrencyFails(t *testing.T) {
	s, mOTC, _, _, mAcc, _, _, _ := newOtcServerWithExchange(t)
	mOTC.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"seller_routing_number", "partner_negotiation_id", "seller_external_id"}).
			AddRow(int32(444), "neg-42", "ext-7"))
	// BuyerAccountId=10 → skip findAccount, go to getAccountCurrencyID
	mAcc.ExpectQuery("SELECT currency_id FROM accounts WHERE id").
		WillReturnError(sql.ErrConnDone)

	_, err := s.exerciseCrossBank(context.Background(),
		&pb.ExerciseContractRequest{ContractId: 1, BuyerAccountId: 10},
		nil, 1, 2, "CLIENT", 10, 100.0, "RSD", "AAPL", futureSettlementDate())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "buyer account currency")
}

func TestExerciseCrossBank_InsufficientFunds(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, mOTC, _, _, mAcc, _, _, _ := newOtcServerWithExchange(t)
	mOTC.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"seller_routing_number", "partner_negotiation_id", "seller_external_id"}).
			AddRow(int32(444), "neg-42", "ext-7"))
	// BuyerAccountId=10
	// getAccountCurrencyID
	mAcc.ExpectQuery("SELECT currency_id FROM accounts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(1)))
	// convertAmount RSD→RSD: no-op
	// account_number query
	mAcc.ExpectQuery("SELECT account_number FROM accounts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("ACC-BUYER"))
	// sagaLog INSERT (for step 1 FAILED)
	mOTC.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 1: reserve funds → 0 rows (insufficient)
	mAcc.ExpectExec("UPDATE accounts SET available_balance").WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := s.exerciseCrossBank(context.Background(),
		&pb.ExerciseContractRequest{ContractId: 1, BuyerAccountId: 10},
		nil, 1, 2, "CLIENT", 10, 100.0, "RSD", "AAPL", futureSettlementDate())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Insufficient funds")
}

// ── sagaFaultHook: delay injection ───────────────────────────────────────────

func TestSagaFaultHook_DelayNonMatchingPhase(t *testing.T) {
	t.Setenv("OTC_SAGA_TEST_HOOKS", "true")
	// inject delay for F2, but we ask for F1 — should be no-op (no sleep, no error)
	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("x-saga-inject-delay", "F2:10"))
	err := sagaFaultHook(ctx, "F1", map[string]int{})
	assert.NoError(t, err)
}

// ── InterbankAcceptNegotiation: update fails ──────────────────────────────────

func TestInterbankAcceptNegotiation_UpdateFails(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)

	addLookupByLocalID(mOTC, 42, "PENDING_BUYER")
	mOTC.ExpectBegin()
	mOTC.ExpectQuery("SELECT id, status, ticker, currency").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "ticker", "currency", "settlement_date",
			"amount", "premium", "price_per_stock", "seller_id", "seller_type",
		}).AddRow(int64(42), "PENDING_BUYER", "AAPL", "RSD", "2099-12-31",
			int32(10), 5.0, 100.0, int64(7), "CLIENT"))
	// UPDATE fails
	mOTC.ExpectExec("UPDATE otc_negotiations").WillReturnError(sql.ErrConnDone)
	mOTC.ExpectRollback()

	_, err := s.InterbankAcceptNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "42",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}
