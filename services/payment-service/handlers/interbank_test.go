package handlers

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func makePrepareReq(postings []*pb.InterbankPosting) *pb.PrepareInterbankPaymentRequest {
	return &pb.PrepareInterbankPaymentRequest{
		IdempotenceKey: &pb.InterbankIdempotenceKey{RoutingNumber: 444, LocallyGeneratedKey: "idem-1"},
		TransactionId:  &pb.InterbankTransactionId{RoutingNumber: 444, Id: "tx-1"},
		Postings:       postings,
	}
}

func makeCommitReq() *pb.CommitRollbackInterbankRequest {
	return &pb.CommitRollbackInterbankRequest{
		TransactionId: &pb.InterbankTransactionId{RoutingNumber: 444, Id: "tx-1"},
	}
}

func balancedPostings(accountNum string) []*pb.InterbankPosting {
	return []*pb.InterbankPosting{
		{AccountType: "ACCOUNT", AccountNum: "SRC-ACC", Amount: -100, AssetType: "MONAS", Currency: "RSD"},
		{AccountType: "ACCOUNT", AccountNum: accountNum, Amount: 100, AssetType: "MONAS", Currency: "RSD"},
	}
}

// ── PrepareInterbankPayment ───────────────────────────────────────────────────

func TestPrepareInterbankPayment_IdempotentCachedVote(t *testing.T) {
	s, dbMock, _, _ := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT cached_vote FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}).AddRow("YES"))

	resp, err := s.PrepareInterbankPayment(context.Background(), makePrepareReq(balancedPostings("DEST-ACC")))
	require.NoError(t, err)
	assert.Equal(t, "YES", resp.Vote)
}

func TestPrepareInterbankPayment_UnbalancedPostings(t *testing.T) {
	s, dbMock, _, _ := newPaymentServerWithExchange(t)
	// No cached vote
	dbMock.ExpectQuery("SELECT cached_vote FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	// insertInterbankTx
	dbMock.ExpectExec("INSERT INTO interbank_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))

	postings := []*pb.InterbankPosting{
		{AccountType: "ACCOUNT", AccountNum: "SRC", Amount: -100, AssetType: "MONAS", Currency: "RSD"},
		{AccountType: "ACCOUNT", AccountNum: "DEST", Amount: 50, AssetType: "MONAS", Currency: "RSD"},
	}
	resp, err := s.PrepareInterbankPayment(context.Background(), makePrepareReq(postings))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
	assert.NotEmpty(t, resp.Reasons)
}

func TestPrepareInterbankPayment_NoCreditPosting(t *testing.T) {
	s, dbMock, _, _ := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT cached_vote FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	dbMock.ExpectExec("INSERT INTO interbank_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Only debit, no credit MONAS
	postings := []*pb.InterbankPosting{
		{AccountType: "ACCOUNT", AccountNum: "SRC", Amount: -100, AssetType: "MONAS", Currency: "RSD"},
		{AccountType: "ACCOUNT", AccountNum: "SRC", Amount: 100, AssetType: "STOCK", Currency: ""},
	}
	resp, err := s.PrepareInterbankPayment(context.Background(), makePrepareReq(postings))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
}

func TestPrepareInterbankPayment_AccountNotFound(t *testing.T) {
	s, dbMock, accountMock, _ := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT cached_vote FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	accountMock.ExpectQuery("SELECT status, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"status", "currency_id"}))
	dbMock.ExpectExec("INSERT INTO interbank_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareInterbankPayment(context.Background(), makePrepareReq(balancedPostings("UNKNOWN-ACC")))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
}

func TestPrepareInterbankPayment_CurrencyMismatch(t *testing.T) {
	s, dbMock, accountMock, exchangeMock := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT cached_vote FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	accountMock.ExpectQuery("SELECT status, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"status", "currency_id"}).AddRow("ACTIVE", int64(2)))
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("EUR"))
	dbMock.ExpectExec("INSERT INTO interbank_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareInterbankPayment(context.Background(), makePrepareReq(balancedPostings("DEST-ACC")))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
}

func TestPrepareInterbankPayment_HappyPath(t *testing.T) {
	s, dbMock, accountMock, exchangeMock := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT cached_vote FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	accountMock.ExpectQuery("SELECT status, currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"status", "currency_id"}).AddRow("ACTIVE", int64(1)))
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	dbMock.ExpectExec("INSERT INTO interbank_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareInterbankPayment(context.Background(), makePrepareReq(balancedPostings("DEST-ACC")))
	require.NoError(t, err)
	assert.Equal(t, "YES", resp.Vote)
}

// ── CommitInterbankPayment ────────────────────────────────────────────────────

func TestCommitInterbankPayment_NotFound(t *testing.T) {
	s, dbMock, _, _ := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT id, status, to_account, amount FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "to_account", "amount"}))

	_, err := s.CommitInterbankPayment(context.Background(), makeCommitReq())
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCommitInterbankPayment_AlreadyRolledBack(t *testing.T) {
	s, dbMock, _, _ := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT id, status, to_account, amount FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "to_account", "amount"}).
			AddRow(int64(1), "ROLLED_BACK", "DEST", 100.0))

	_, err := s.CommitInterbankPayment(context.Background(), makeCommitReq())
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestCommitInterbankPayment_AlreadyCommitted(t *testing.T) {
	s, dbMock, _, _ := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT id, status, to_account, amount FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "to_account", "amount"}).
			AddRow(int64(1), "PENDING", "DEST", 100.0))
	// UPDATE returns 0 rows affected (was already committed by a prior retry)
	dbMock.ExpectExec("UPDATE interbank_transactions SET status = 'COMMITTED'").
		WillReturnResult(sqlmock.NewResult(0, 0))

	resp, err := s.CommitInterbankPayment(context.Background(), makeCommitReq())
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestCommitInterbankPayment_HappyPath(t *testing.T) {
	s, dbMock, accountMock, _ := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT id, status, to_account, amount FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "to_account", "amount"}).
			AddRow(int64(1), "PENDING", "DEST-ACC", 100.0))
	dbMock.ExpectExec("UPDATE interbank_transactions SET status = 'COMMITTED'").
		WillReturnResult(sqlmock.NewResult(1, 1))
	accountMock.ExpectExec("UPDATE accounts SET balance").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.CommitInterbankPayment(context.Background(), makeCommitReq())
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ── RollbackInterbankPayment ──────────────────────────────────────────────────

func TestRollbackInterbankPayment_NotFound_IsIdempotent(t *testing.T) {
	s, dbMock, _, _ := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT id, status FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}))

	resp, err := s.RollbackInterbankPayment(context.Background(), makeCommitReq())
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestRollbackInterbankPayment_AlreadyRolledBack_IsIdempotent(t *testing.T) {
	s, dbMock, _, _ := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT id, status FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).
			AddRow(int64(1), "ROLLED_BACK"))

	resp, err := s.RollbackInterbankPayment(context.Background(), makeCommitReq())
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestRollbackInterbankPayment_AlreadyCommitted(t *testing.T) {
	s, dbMock, _, _ := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT id, status FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).
			AddRow(int64(1), "COMMITTED"))

	_, err := s.RollbackInterbankPayment(context.Background(), makeCommitReq())
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestRollbackInterbankPayment_HappyPath(t *testing.T) {
	s, dbMock, _, _ := newPaymentServerWithExchange(t)
	dbMock.ExpectQuery("SELECT id, status FROM interbank_transactions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).
			AddRow(int64(1), "PENDING"))
	dbMock.ExpectExec("UPDATE interbank_transactions SET status = 'ROLLED_BACK'").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.RollbackInterbankPayment(context.Background(), makeCommitReq())
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
