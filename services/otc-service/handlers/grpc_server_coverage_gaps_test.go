package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ===== CreateNegotiation =====

func TestCreateNegotiation_MissingSettlementDate(t *testing.T) {
	s, _, _, _, _, _, _ := newTestServer(t)
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker: "AAPL", Amount: 5, PricePerStock: 100.0, Currency: "USD", SettlementDate: "",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "settlement date is required")
}

func TestCreateNegotiation_PastSettlementDate(t *testing.T) {
	s, _, _, _, _, _, _ := newTestServer(t)
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker: "AAPL", Amount: 5, PricePerStock: 100.0, Currency: "USD", SettlementDate: "2020-01-01",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "settlement date must be in the future")
}

func TestCreateNegotiation_UnknownTicker(t *testing.T) {
	s, _, _, _, _, _, mSec := newTestServer(t)
	mSec.ExpectQuery("SELECT id FROM listing").WillReturnError(sql.ErrNoRows)
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker: "FAKE", Amount: 5, PricePerStock: 100.0, Currency: "USD", SettlementDate: "2027-06-01",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "unknown ticker")
}

func TestCreateNegotiation_PortfolioDBError(t *testing.T) {
	s, _, _, _, _, mPort, mSec := newTestServer(t)
	mSec.ExpectQuery("SELECT id FROM listing").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").WillReturnError(fmt.Errorf("portfolio db error"))
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker: "AAPL", Amount: 5, PricePerStock: 100.0, Currency: "USD",
		SellerId: 10, SellerType: "EMPLOYEE", SettlementDate: "2027-06-01",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "failed to check seller portfolio")
}

func TestCreateNegotiation_InsufficientShares(t *testing.T) {
	s, mainMock, _, _, _, mPort, mSec := newTestServer(t)
	mSec.ExpectQuery("SELECT id FROM listing").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(3)))
	mainMock.ExpectQuery("SELECT COALESCE.*FROM otc_contracts").WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mainMock.ExpectQuery("SELECT COALESCE.*FROM otc_negotiations").WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker: "AAPL", Amount: 5, PricePerStock: 100.0, Currency: "USD",
		SellerId: 10, SellerType: "EMPLOYEE", SettlementDate: "2027-06-01",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "enough free shares")
}

// ===== CounterOffer =====

func TestCounterOffer_MissingSettlementDate(t *testing.T) {
	s, _, _, _, _, _, _ := newTestServer(t)
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, SettlementDate: "",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "settlement date is required")
}

func TestCounterOffer_PastSettlementDate(t *testing.T) {
	s, _, _, _, _, _, _ := newTestServer(t)
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, SettlementDate: "2020-01-01",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "settlement date must be in the future")
}

func TestCounterOffer_BeginTxFails(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin().WillReturnError(sql.ErrConnDone)
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE", SettlementDate: "2027-06-01",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "failed to begin transaction")
}

func TestCounterOffer_CommitFails(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))
	mainMock.ExpectExec("UPDATE otc_negotiations").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectCommit().WillReturnError(sql.ErrConnDone)
	mainMock.ExpectRollback()
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE", SettlementDate: "2027-06-01",
		Amount: 5, PricePerStock: 100.0,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "failed to commit counter offer")
}

// ===== AcceptNegotiation =====

func TestAcceptNegotiation_BeginTxFails(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin().WillReturnError(sql.ErrConnDone)
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "failed to begin transaction")
}

func TestAcceptNegotiation_CommitFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mSec.ExpectQuery("SELECT id FROM listing").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(50)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	// BuyerAccountId=0 → findAccount buyer
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))
	mAcc.ExpectQuery("SELECT currency_id FROM accounts WHERE id").WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(4)))
	mAcc.ExpectQuery("SELECT available_balance FROM accounts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(500)))
	// findAccount seller
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1)) // deduct buyer
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1)) // credit seller
	mainMock.ExpectQuery("INSERT INTO otc_contracts").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mainMock.ExpectExec("UPDATE otc_negotiations").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectCommit().WillReturnError(sql.ErrConnDone)
	// After a failed Commit(), database/sql marks tx.done=true so deferred Rollback()
	// returns ErrTxDone without hitting the driver — no ExpectRollback needed here.
	// compensation: restore buyer, restore seller
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT", BuyerAccountId: 0,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "failed to commit accept")
	assert.NoError(t, mainMock.ExpectationsWereMet())
	assert.NoError(t, mAcc.ExpectationsWereMet())
}

// ===== RejectNegotiation =====

func TestRejectNegotiation_BeginTxFails(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin().WillReturnError(sql.ErrConnDone)
	_, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "failed to begin transaction")
}

func TestRejectNegotiation_CommitFails(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))
	mainMock.ExpectExec("UPDATE otc_negotiations SET status = 'REJECTED'").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectCommit().WillReturnError(sql.ErrConnDone)
	mainMock.ExpectRollback()
	_, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "failed to commit rejection")
}

// ===== ExerciseContract =====

func TestExerciseContract_IdempotentAlreadyExercised(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}).AddRow(5, "SUCCESS"))
	resp, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT",
	})
	require.NoError(t, err)
	assert.Equal(t, "EXERCISED", resp.Status)
	assert.NoError(t, mainMock.ExpectationsWereMet())
}

func TestExerciseContract_BeginTxFails(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin().WillReturnError(sql.ErrConnDone)
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "failed to begin transaction")
}

// exerciseStep1to4 sets up sqlmock expectations for ExerciseContract steps 1-4 (all succeed).
// Seller=10 (CLIENT), buyer=20 (CLIENT), 5 shares at 100 USD each. buyerAccountId=100, sellerAccountId=200.
func exerciseStep1to4(t *testing.T, mainMock, mAcc, mPort, mSec sqlmock.Sqlmock, future time.Time) {
	t.Helper()
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mAcc.ExpectQuery("SELECT currency_id FROM accounts WHERE id").WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(4)))
	// Step 1
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 2
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 3
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance - ").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 4
	mPort.ExpectExec(`(?s)UPDATE portfolio_entry.*public_amount = GREATEST`).WillReturnResult(sqlmock.NewResult(0, 1))
	mPort.ExpectExec("DELETE FROM portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 0))
	mPort.ExpectExec("INSERT INTO portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
}

// comp4to1 sets up sqlmock expectations for full compensation (comp4 → comp3 → comp2 → comp1).
func comp4to1(mainMock, mAcc, mPort sqlmock.Sqlmock) {
	// comp4: restore seller + remove buyer shares
	mPort.ExpectExec("UPDATE portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 1))
	mPort.ExpectExec("UPDATE portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// comp3: restore buyer balance + restore seller balance
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// comp2: restore seller reserved_amount
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// comp1: restore buyer available_balance
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance \\+").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
}

func TestExerciseContract_Step5DoubleCheckFailed(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	exerciseStep1to4(t, mainMock, mAcc, mPort, mSec, future)
	// Step 5: double-check buyer holding returns 0 (inconsistent)
	mPort.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(0)))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 5 FAILED
	comp4to1(mainMock, mAcc, mPort)
	mainMock.ExpectRollback()

	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "step 5 double check failed")
	assert.NoError(t, mainMock.ExpectationsWereMet())
	assert.NoError(t, mAcc.ExpectationsWereMet())
	assert.NoError(t, mPort.ExpectationsWereMet())
}

func TestExerciseContract_CommitFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	exerciseStep1to4(t, mainMock, mAcc, mPort, mSec, future)
	// Step 5: double-check succeeds
	mPort.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(5)))
	mainMock.ExpectExec("UPDATE otc_contracts SET status").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectCommit().WillReturnError(sql.ErrConnDone)
	// After a failed Commit(), database/sql marks tx.done=true so deferred Rollback()
	// returns ErrTxDone without hitting the driver — no ExpectRollback needed here.
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 5 FAILED
	comp4to1(mainMock, mAcc, mPort)

	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "step 5 commit failed")
	assert.NoError(t, mainMock.ExpectationsWereMet())
	assert.NoError(t, mAcc.ExpectationsWereMet())
	assert.NoError(t, mPort.ExpectationsWereMet())
}

// ===== GetMarket =====

func TestGetMarket_SkipsZeroFreeShares(t *testing.T) {
	s, mainMock, _, _, _, mPort, mSec := newTestServer(t)

	// Portfolio row: 5 public shares available
	mPort.ExpectQuery("SELECT user_id, user_type, listing_id, public_amount, last_modified").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "user_type", "listing_id", "public_amount", "last_modified"}).
			AddRow(int64(1), "CLIENT", int64(42), int32(5), time.Now()))

	// Securities returns valid data so we reach the freeAmount check
	mSec.ExpectQuery("SELECT l.ticker, l.name, l.price, se.currency").
		WillReturnRows(sqlmock.NewRows([]string{"ticker", "name", "price", "currency"}).
			AddRow("AAPL", "Apple Inc.", float64(150.0), "USD"))

	// Pending negotiations sum = 5 (same as publicAmount) → freeAmount = 5 - 5 - 0 = 0 → skip
	mainMock.ExpectQuery("SELECT COALESCE.*FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(5)))
	mainMock.ExpectQuery("SELECT COALESCE.*FROM otc_contracts").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))

	resp, err := s.GetMarket(context.Background(), &pb.GetMarketRequest{CallerId: 99, CallerType: "CLIENT"})
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.NoError(t, mainMock.ExpectationsWereMet())
	assert.NoError(t, mPort.ExpectationsWereMet())
	assert.NoError(t, mSec.ExpectationsWereMet())
}
