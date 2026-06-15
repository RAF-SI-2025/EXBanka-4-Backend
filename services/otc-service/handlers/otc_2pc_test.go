package handlers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// newOtcServerWithExchange creates a full OtcServer with all seven mock DBs including ExchangeDB.
func newOtcServerWithExchange(t *testing.T) (*OtcServer, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	newDB := func() (*sql.DB, sqlmock.Sqlmock) {
		db, m, err := sqlmock.New()
		require.NoError(t, err)
		return db, m
	}
	db, mOTC := newDB()
	empDB, mEmp := newDB()
	cliDB, mCli := newDB()
	accDB, mAcc := newDB()
	portDB, mPort := newDB()
	secDB, mSec := newDB()
	excDB, mExc := newDB()
	t.Cleanup(func() {
		_ = db.Close()
		_ = empDB.Close()
		_ = cliDB.Close()
		_ = accDB.Close()
		_ = portDB.Close()
		_ = secDB.Close()
		_ = excDB.Close()
	})
	return &OtcServer{
		DB: db, EmployeeDB: empDB, ClientDB: cliDB,
		AccountDB: accDB, PortfolioDB: portDB, SecuritiesDB: secDB, ExchangeDB: excDB,
	}, mOTC, mEmp, mCli, mAcc, mPort, mSec, mExc
}

// makePrepareOtcReq builds a minimal OtcInterbankPrepareRequest.
func makePrepareOtcReq(negID int64, isAccept bool, idemRouting string) *pb.OtcInterbankPrepareRequest {
	return &pb.OtcInterbankPrepareRequest{
		IdemRoutingNumber: idemRouting,
		IdemKey:           "idem-1",
		TxRoutingNumber:   idemRouting,
		TxId:              "tx-abc",
		NegotiationId:     negID,
		StockAmount:       10,
		IsAccept:          isAccept,
	}
}

// makeCommitOtcReq builds a minimal OtcInterbankTxRequest.
func makeCommitOtcReq() *pb.OtcInterbankTxRequest {
	return &pb.OtcInterbankTxRequest{TxRoutingNumber: "444", TxId: "tx-abc"}
}

// ── PrepareOtcInterbank ───────────────────────────────────────────────────────

func TestPrepareOtcInterbank_IdempotentCachedVote(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}).AddRow("YES"))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(1, false, "888"))
	require.NoError(t, err)
	assert.Equal(t, "YES", resp.Vote)
}

func TestPrepareOtcInterbank_IdempotencyCheckFails(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnError(sql.ErrConnDone)

	_, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(1, false, "888"))
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestPrepareOtcInterbank_Exercise_NegotiationNotFound(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	// No cached vote
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	// Negotiation not found
	mOTC.ExpectQuery("SELECT status, ticker, seller_id, seller_type, amount").
		WillReturnRows(sqlmock.NewRows([]string{"status", "ticker", "seller_id", "seller_type", "amount"}))
	// insertOtcInterbankTx
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(99, false, "888"))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
	assert.Equal(t, "OPTION_NEGOTIATION_NOT_FOUND", resp.Reason)
}

func TestPrepareOtcInterbank_Exercise_ContractNotFound(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	mOTC.ExpectQuery("SELECT status, ticker, seller_id, seller_type, amount").
		WillReturnRows(sqlmock.NewRows([]string{"status", "ticker", "seller_id", "seller_type", "amount"}).
			AddRow("ACCEPTED", "AAPL", int64(1), "CLIENT", int32(10)))
	// Contract not found
	mOTC.ExpectQuery("SELECT c.status, c.amount, n.settlement_date").
		WillReturnRows(sqlmock.NewRows([]string{"status", "amount", "settlement_date"}))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(1, false, "888"))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
}

func TestPrepareOtcInterbank_Exercise_ContractNotActive(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	mOTC.ExpectQuery("SELECT status, ticker, seller_id, seller_type, amount").
		WillReturnRows(sqlmock.NewRows([]string{"status", "ticker", "seller_id", "seller_type", "amount"}).
			AddRow("ACCEPTED", "AAPL", int64(1), "CLIENT", int32(10)))
	// Contract is EXERCISED
	mOTC.ExpectQuery("SELECT c.status, c.amount, n.settlement_date").
		WillReturnRows(sqlmock.NewRows([]string{"status", "amount", "settlement_date"}).
			AddRow("EXERCISED", int32(10), time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(1, false, "888"))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
	assert.Equal(t, "OPTION_USED_OR_EXPIRED", resp.Reason)
}

func TestPrepareOtcInterbank_Exercise_HappyPath(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	mOTC.ExpectQuery("SELECT status, ticker, seller_id, seller_type, amount").
		WillReturnRows(sqlmock.NewRows([]string{"status", "ticker", "seller_id", "seller_type", "amount"}).
			AddRow("ACCEPTED", "AAPL", int64(1), "CLIENT", int32(10)))
	// Active contract with far future settlement date and matching amount
	futureDate := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
	mOTC.ExpectQuery("SELECT c.status, c.amount, n.settlement_date").
		WillReturnRows(sqlmock.NewRows([]string{"status", "amount", "settlement_date"}).
			AddRow("ACTIVE", int32(10), futureDate))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(1, false, "888"))
	require.NoError(t, err)
	assert.Equal(t, "YES", resp.Vote)
}

func TestPrepareOtcInterbank_BuyerAccept_NegotiationNotFound(t *testing.T) {
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	// partner_negotiation_id lookup returns nothing
	mOTC.ExpectQuery("SELECT id FROM otc_negotiations WHERE partner_negotiation_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(42, true, "444"))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
	assert.Equal(t, "OPTION_NEGOTIATION_NOT_FOUND", resp.Reason)
}

func TestPrepareOtcInterbank_BuyerAccept_StatusNotAccepted(t *testing.T) {
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, mOTC, _, _, mAcc, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	mOTC.ExpectQuery("SELECT id FROM otc_negotiations WHERE partner_negotiation_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))
	mOTC.ExpectQuery("SELECT buyer_account_number, premium, status FROM otc_negotiations WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"buyer_account_number", "premium", "status"}).
			AddRow("ACC-001", 5.0, "PENDING_BUYER"))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))
	_ = mAcc

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(42, true, "444"))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
	assert.Equal(t, "OPTION_USED_OR_EXPIRED", resp.Reason)
}

func TestPrepareOtcInterbank_BuyerAccept_InsufficientFunds(t *testing.T) {
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, mOTC, _, _, mAcc, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	mOTC.ExpectQuery("SELECT id FROM otc_negotiations WHERE partner_negotiation_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))
	mOTC.ExpectQuery("SELECT buyer_account_number, premium, status FROM otc_negotiations WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"buyer_account_number", "premium", "status"}).
			AddRow("ACC-001", 100.0, "ACCEPTED"))
	// available_balance < premium
	mAcc.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(10.0))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(42, true, "444"))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
	assert.Equal(t, "INSUFFICIENT_FUNDS", resp.Reason)
}

func TestPrepareOtcInterbank_BuyerAccept_HappyPath(t *testing.T) {
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, mOTC, _, _, mAcc, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	mOTC.ExpectQuery("SELECT id FROM otc_negotiations WHERE partner_negotiation_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))
	mOTC.ExpectQuery("SELECT buyer_account_number, premium, status FROM otc_negotiations WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"buyer_account_number", "premium", "status"}).
			AddRow("ACC-001", 5.0, "ACCEPTED"))
	mAcc.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(500.0))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(42, true, "444"))
	require.NoError(t, err)
	assert.Equal(t, "YES", resp.Vote)
}

// ── CommitOtcInterbank ────────────────────────────────────────────────────────

func TestCommitOtcInterbank_TxNotFound(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT id, status, tx_type, negotiation_id, stock_amount").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "tx_type", "negotiation_id", "stock_amount"}))

	resp, err := s.CommitOtcInterbank(context.Background(), makeCommitOtcReq())
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestCommitOtcInterbank_LoadError(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT id, status, tx_type, negotiation_id, stock_amount").
		WillReturnError(sql.ErrConnDone)

	_, err := s.CommitOtcInterbank(context.Background(), makeCommitOtcReq())
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCommitOtcInterbank_AlreadyCommitted(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT id, status, tx_type, negotiation_id, stock_amount").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "tx_type", "negotiation_id", "stock_amount"}).
			AddRow(int64(1), "COMMITTED", "ACCEPT", int64(7), int32(10)))

	resp, err := s.CommitOtcInterbank(context.Background(), makeCommitOtcReq())
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestCommitOtcInterbank_AlreadyRolledBack(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT id, status, tx_type, negotiation_id, stock_amount").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "tx_type", "negotiation_id", "stock_amount"}).
			AddRow(int64(1), "ROLLED_BACK", "ACCEPT", int64(7), int32(10)))

	_, err := s.CommitOtcInterbank(context.Background(), makeCommitOtcReq())
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestCommitOtcInterbank_Accept_BuyerSide_HappyPath(t *testing.T) {
	s, mOTC, _, _, mAcc, _, _, _ := newOtcServerWithExchange(t)
	// tx lookup
	mOTC.ExpectQuery("SELECT id, status, tx_type, negotiation_id, stock_amount").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "tx_type", "negotiation_id", "stock_amount"}).
			AddRow(int64(1), "PENDING", "ACCEPT", int64(7), int32(10)))
	// load negotiation for ACCEPT type — sellerType="INTERBANK" means we're buyer
	mOTC.ExpectQuery("SELECT ticker, currency, settlement_date").
		WillReturnRows(sqlmock.NewRows([]string{
			"ticker", "currency", "settlement_date", "seller_id", "seller_type",
			"amount", "price_per_stock", "premium",
		}).AddRow("AAPL", "RSD", "2099-12-31", int64(0), "INTERBANK", int32(10), 100.0, 5.0))
	// load buyer info
	mOTC.ExpectQuery("SELECT buyer_account_number, buyer_id, buyer_type FROM otc_negotiations WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"buyer_account_number", "buyer_id", "buyer_type"}).
			AddRow("ACC-BUYER", int64(3), "CLIENT"))
	// currency conversion: RSD→RSD same ID → no exchange needed (currencyIDMap["RSD"]=1)
	// account currency lookup
	mAcc.ExpectQuery("SELECT currency_id FROM accounts WHERE account_number").
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(1)))
	// debit buyer account
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(1, 1))
	// insert contract
	mOTC.ExpectExec("INSERT INTO otc_contracts").WillReturnResult(sqlmock.NewResult(1, 1))
	// mark committed
	mOTC.ExpectExec("UPDATE otc_interbank_tx SET status='COMMITTED'").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.CommitOtcInterbank(context.Background(), makeCommitOtcReq())
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestCommitOtcInterbank_Exercise_HappyPath(t *testing.T) {
	s, mOTC, _, _, mAcc, mPort, mSec := newTestServer(t)
	// tx lookup — EXERCISE type
	mOTC.ExpectQuery("SELECT id, status, tx_type, negotiation_id, stock_amount").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "tx_type", "negotiation_id", "stock_amount"}).
			AddRow(int64(1), "PENDING", "EXERCISE", int64(7), int32(10)))
	// load contract
	mOTC.ExpectQuery("SELECT seller_id, seller_type, ticker, amount, currency, strike_price FROM otc_contracts").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "ticker", "amount", "currency", "strike_price"}).
			AddRow(int64(5), "CLIENT", "AAPL", int32(10), "RSD", 50.0))
	// listingIDForTicker
	mSec.ExpectQuery("SELECT id FROM listing WHERE ticker").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// UPDATE portfolio_entry (remove shares)
	mPort.ExpectExec("UPDATE portfolio_entry").WillReturnResult(sqlmock.NewResult(1, 1))
	// DELETE portfolio_entry with amount<=0
	mPort.ExpectExec("DELETE FROM portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 0))
	// findAccount for seller (currency RSD=1)
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))
	// credit seller account
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(1, 1))
	// UPDATE contract status
	mOTC.ExpectExec("UPDATE otc_contracts SET status='EXERCISED'").WillReturnResult(sqlmock.NewResult(1, 1))
	// UPDATE tx status
	mOTC.ExpectExec("UPDATE otc_interbank_tx SET status='COMMITTED'").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.CommitOtcInterbank(context.Background(), makeCommitOtcReq())
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ── RollbackOtcInterbank ──────────────────────────────────────────────────────

func TestRollbackOtcInterbank_HappyPath(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectExec("UPDATE otc_interbank_tx SET status='ROLLED_BACK'").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.RollbackOtcInterbank(context.Background(), makeCommitOtcReq())
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ── ExpireContracts ───────────────────────────────────────────────────────────

func TestExpireContracts_DBError(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("UPDATE otc_contracts SET status='EXPIRED'").WillReturnError(sql.ErrConnDone)
	// Should not panic
	s.ExpireContracts()
}

func TestExpireContracts_NoRows(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("UPDATE otc_contracts SET status='EXPIRED'").
		WillReturnRows(sqlmock.NewRows([]string{"id", "buyer_id", "buyer_type", "premium", "currency"}))
	s.ExpireContracts()
}

// ── RecoverInFlightSagas ──────────────────────────────────────────────────────

func TestRecoverInFlightSagas_DBError(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT id, contract_id, current_step, status FROM otc_saga").
		WillReturnError(sql.ErrConnDone)
	// Should not panic
	s.RecoverInFlightSagas()
}

func TestRecoverInFlightSagas_NoSagas(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT id, contract_id, current_step, status FROM otc_saga").
		WillReturnRows(sqlmock.NewRows([]string{"id", "contract_id", "current_step", "status"}))
	s.RecoverInFlightSagas()
}

// ── sagaFaultHook ─────────────────────────────────────────────────────────────

func TestSagaFaultHook_NotEnabled(t *testing.T) {
	ctx := context.Background()
	err := sagaFaultHook(ctx, "F1", map[string]int{})
	assert.NoError(t, err)
}

func TestSagaFaultHook_ForceFail(t *testing.T) {
	t.Setenv("OTC_SAGA_TEST_HOOKS", "true")
	md := metadata.Pairs("x-saga-force-fail", "F1")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	err := sagaFaultHook(ctx, "F1", map[string]int{})
	assert.Error(t, err)
}

func TestSagaFaultHook_NoMetadata(t *testing.T) {
	t.Setenv("OTC_SAGA_TEST_HOOKS", "true")
	ctx := context.Background()
	err := sagaFaultHook(ctx, "F1", map[string]int{})
	assert.NoError(t, err)
}
