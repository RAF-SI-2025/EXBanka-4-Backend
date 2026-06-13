package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func futureSettlementDate() time.Time {
	return time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
}

// ── executeInterbankAcceptOutgoing ────────────────────────────────────────────

func TestExecuteInterbankAcceptOutgoing_LoadNegotiationFails(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT buyer_account_number, buyer_routing_number, buyer_external_id").
		WillReturnError(sql.ErrConnDone)

	err := s.executeInterbankAcceptOutgoing(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load negotiation")
}

func TestExecuteInterbankAcceptOutgoing_UnknownBuyerRouting(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, mOTC, _, _, _, _, _ := newTestServer(t)
	// buyerRoutingNum=999 — not own bank (888) or partner (444) → ResolveBankByRoutingNumber fails
	mOTC.ExpectQuery("SELECT buyer_account_number, buyer_routing_number, buyer_external_id").
		WillReturnRows(sqlmock.NewRows([]string{
			"buyer_account_number", "buyer_routing_number", "buyer_external_id",
			"seller_id", "seller_type", "premium", "currency", "amount", "ticker",
		}).AddRow("ACC-001", int32(999), "ext-1", int64(5), "CLIENT", 5.0, "RSD", int32(10), "AAPL"))

	err := s.executeInterbankAcceptOutgoing(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve buyer bank")
}

func TestExecuteInterbankAcceptOutgoing_HTTPVoteNo(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"vote": "NO", "reason": "INSUFFICIENT_FUNDS"})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("PARTNER_BANK_URL", srv.URL)
	t.Setenv("PARTNER_API_KEY", "test-key")

	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT buyer_account_number, buyer_routing_number, buyer_external_id").
		WillReturnRows(sqlmock.NewRows([]string{
			"buyer_account_number", "buyer_routing_number", "buyer_external_id",
			"seller_id", "seller_type", "premium", "currency", "amount", "ticker",
		}).AddRow("ACC-001", int32(444), "ext-1", int64(5), "CLIENT", 5.0, "RSD", int32(10), "AAPL"))

	err := s.executeInterbankAcceptOutgoing(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "voted NO")
}

func TestExecuteInterbankAcceptOutgoing_TickerNotFound(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"vote": "YES"})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("PARTNER_BANK_URL", srv.URL)
	t.Setenv("PARTNER_API_KEY", "test-key")

	s, mOTC, _, _, _, _, mSec := newTestServer(t)
	mOTC.ExpectQuery("SELECT buyer_account_number, buyer_routing_number, buyer_external_id").
		WillReturnRows(sqlmock.NewRows([]string{
			"buyer_account_number", "buyer_routing_number", "buyer_external_id",
			"seller_id", "seller_type", "premium", "currency", "amount", "ticker",
		}).AddRow("ACC-001", int32(444), "ext-1", int64(5), "CLIENT", 5.0, "RSD", int32(10), "AAPL"))
	// listingIDForTicker: not found
	mSec.ExpectQuery("SELECT id FROM listing WHERE ticker").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	err := s.executeInterbankAcceptOutgoing(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticker not found")
}

func TestExecuteInterbankAcceptOutgoing_InsufficientShares(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"vote": "YES"})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("PARTNER_BANK_URL", srv.URL)
	t.Setenv("PARTNER_API_KEY", "test-key")

	s, mOTC, _, _, _, mPort, mSec := newTestServer(t)
	mOTC.ExpectQuery("SELECT buyer_account_number, buyer_routing_number, buyer_external_id").
		WillReturnRows(sqlmock.NewRows([]string{
			"buyer_account_number", "buyer_routing_number", "buyer_external_id",
			"seller_id", "seller_type", "premium", "currency", "amount", "ticker",
		}).AddRow("ACC-001", int32(444), "ext-1", int64(5), "CLIENT", 5.0, "RSD", int32(10), "AAPL"))
	mSec.ExpectQuery("SELECT id FROM listing WHERE ticker").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	// UPDATE returns 0 rows (no free shares)
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := s.executeInterbankAcceptOutgoing(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient free shares")
}

// ── recoverSaga ───────────────────────────────────────────────────────────────

func TestRecoverSaga_NoCompletedSteps(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	// Mark as Compensating
	mOTC.ExpectExec("UPDATE otc_saga SET status='Compensating'").WillReturnResult(sqlmock.NewResult(1, 1))
	// No completed steps
	mOTC.ExpectQuery("SELECT DISTINCT step FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step"}))
	// Mark as Compensated
	mOTC.ExpectExec("UPDATE otc_saga SET status='Compensated'").WillReturnResult(sqlmock.NewResult(1, 1))

	s.recoverSaga(1, 1)
}

func TestRecoverSaga_LogQueryFails(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectExec("UPDATE otc_saga SET status='Compensating'").WillReturnResult(sqlmock.NewResult(1, 1))
	mOTC.ExpectQuery("SELECT DISTINCT step FROM otc_saga_log").
		WillReturnError(sql.ErrConnDone)

	// Should not panic
	s.recoverSaga(1, 1)
}

func TestRecoverSaga_ContractLoadFails(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectExec("UPDATE otc_saga SET status='Compensating'").WillReturnResult(sqlmock.NewResult(1, 1))
	mOTC.ExpectQuery("SELECT DISTINCT step FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step"}).AddRow(1))
	// contract load fails
	mOTC.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, ticker, amount, strike_price, currency").
		WillReturnError(sql.ErrConnDone)

	s.recoverSaga(1, 1)
}

// ── exerciseCrossBank: early failure paths ────────────────────────────────────

func TestExerciseCrossBank_UnsupportedCurrency(t *testing.T) {
	s, _, _, _, _, _, _, _ := newOtcServerWithExchange(t)
	// Currency check happens before any DB or tx access — safe to pass nil tx.
	_, err := s.exerciseCrossBank(context.Background(), &pb.ExerciseContractRequest{ContractId: 1},
		nil, 1, 2, "CLIENT", 10, 100.0, "UNKNOWN_CURRENCY", "AAPL", futureSettlementDate())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported currency")
}

func TestExerciseCrossBank_RoutingQueryFails(t *testing.T) {
	s, mOTC, _, _, _, _, _, _ := newOtcServerWithExchange(t)
	// Currency is valid; routing query on s.DB fails before contractTx is used.
	mOTC.ExpectQuery("SELECT COALESCE").WillReturnError(sql.ErrConnDone)

	_, err := s.exerciseCrossBank(context.Background(), &pb.ExerciseContractRequest{ContractId: 1},
		nil, 1, 2, "CLIENT", 10, 100.0, "RSD", "AAPL", futureSettlementDate())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cross-bank exercise requires")
}
