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

// newTestServer creates an OtcServer with six sqlmock databases.
// Returns: server, mockOTC, mockEmp, mockCli, mockAcc, mockPort, mockSec
func newTestServer(t *testing.T) (*OtcServer, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock) {
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
	t.Cleanup(func() {
		_ = db.Close()
		_ = empDB.Close()
		_ = cliDB.Close()
		_ = accDB.Close()
		_ = portDB.Close()
		_ = secDB.Close()
	})
	return &OtcServer{
		DB: db, EmployeeDB: empDB, ClientDB: cliDB,
		AccountDB: accDB, PortfolioDB: portDB, SecuritiesDB: secDB,
	}, mOTC, mEmp, mCli, mAcc, mPort, mSec
}

// negotiationColumns returns the columns scanned by fetchNegotiationByID.
func negotiationColumns() []string {
	return []string{
		"id", "ticker", "seller_id", "seller_type", "buyer_id", "buyer_type",
		"amount", "price_per_stock", "settlement_date", "premium", "currency",
		"last_modified", "modified_by_id", "modified_by_type", "status",
	}
}

// addFetchNegotiationRows sets up mock expectations for fetchNegotiationByID
// (SELECT + name lookups for seller, buyer, and modifiedBy).
func addFetchNegotiationRows(mainMock, empMock, clientMock sqlmock.Sqlmock,
	id, sellerID, buyerID int64, sellerType, buyerType, negotiationStatus string) {
	now := time.Now()
	mainMock.ExpectQuery("SELECT id, ticker").
		WillReturnRows(sqlmock.NewRows(negotiationColumns()).
			AddRow(id, "AAPL", sellerID, sellerType, buyerID, buyerType,
				int32(100), float64(150.0), "2026-06-01", float64(0), "RSD",
				now, sql.NullInt64{Int64: buyerID, Valid: true},
				sql.NullString{String: buyerType, Valid: true},
				negotiationStatus))
	if sellerType == "EMPLOYEE" {
		empMock.ExpectQuery("SELECT first_name").
			WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Jane Doe"))
	} else {
		clientMock.ExpectQuery("SELECT first_name").
			WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Jane Doe"))
	}
	if buyerType == "EMPLOYEE" {
		empMock.ExpectQuery("SELECT first_name").
			WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("John Smith"))
	} else {
		clientMock.ExpectQuery("SELECT first_name").
			WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("John Smith"))
	}
	// modifiedBy name lookup (same type as buyer)
	if buyerType == "EMPLOYEE" {
		empMock.ExpectQuery("SELECT first_name").
			WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("John Smith"))
	} else {
		clientMock.ExpectQuery("SELECT first_name").
			WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("John Smith"))
	}
}

// acceptNegRow returns all 11 columns required by AcceptNegotiation's initial FOR UPDATE SELECT.
func acceptNegRow(sellerID, buyerID int64, sellerType, buyerType, state, ticker, currency string, amount int32, premium float64) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"seller_id", "seller_type", "buyer_id", "buyer_type", "status",
		"ticker", "amount", "premium", "currency",
		"settlement_date", "price_per_stock",
	}).AddRow(sellerID, sellerType, buyerID, buyerType, state, ticker, amount, premium, currency,
		"2026-12-31", float64(100.0))
}

// contractRow returns the 10 columns scanned in ExerciseContract's initial load.
func contractRow(sellerID, buyerID int64, settlementDate time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"seller_id", "seller_type", "buyer_id", "buyer_type", "status",
		"ticker", "amount", "strike_price", "currency", "settlement_date",
	}).AddRow(sellerID, "CLIENT", buyerID, "CLIENT", "ACTIVE",
		"AAPL", int32(5), float64(100.0), "USD", settlementDate)
}

// ===== Ping =====

func TestPing_OtcService(t *testing.T) {
	s, _, _, _, _, _, _ := newTestServer(t)
	resp, err := s.Ping(context.Background(), &pb.PingRequest{})
	require.NoError(t, err)
	assert.Equal(t, "otc-service ok", resp.Message)
}

// ===== getUserName =====

func TestGetUserName_ZeroID(t *testing.T) {
	name := getUserName(nil, nil, 0, "CLIENT")
	assert.Equal(t, "", name)
}

func TestGetUserName_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("SELECT first_name").WillReturnError(fmt.Errorf("db error"))
	name := getUserName(nil, db, 42, "CLIENT")
	assert.Equal(t, "", name)
}

// ===== CreateNegotiation =====

func TestCreateNegotiation_Happy(t *testing.T) {
	s, mainMock, empMock, clientMock, _, mPort, mSec := newTestServer(t)

	// Pre-flight: ticker lookup, portfolio free shares, active/pending sums
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(1000)))
	mainMock.ExpectQuery("SELECT COALESCE.*FROM otc_contracts").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mainMock.ExpectQuery("SELECT COALESCE.*FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))

	mainMock.ExpectQuery("INSERT INTO otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	addFetchNegotiationRows(mainMock, empMock, clientMock, 1, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_SELLER")

	resp, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker: "AAPL", SellerId: 10, SellerType: "EMPLOYEE",
		BuyerId: 20, BuyerType: "CLIENT", Amount: 100,
		PricePerStock: 150.0, SettlementDate: "2027-06-01", Currency: "RSD",
	})
	require.NoError(t, err)
	assert.Equal(t, "AAPL", resp.Ticker)
	assert.Equal(t, "PENDING_SELLER", resp.Status)
}

func TestCreateNegotiation_MissingTicker(t *testing.T) {
	s, _, _, _, _, _, _ := newTestServer(t)
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Amount: 100, PricePerStock: 150.0, Currency: "RSD",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateNegotiation_InvalidAmount(t *testing.T) {
	s, _, _, _, _, _, _ := newTestServer(t)
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker: "AAPL", Amount: 0, PricePerStock: 100.0,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateNegotiation_InvalidPrice(t *testing.T) {
	s, _, _, _, _, _, _ := newTestServer(t)
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker: "AAPL", Amount: 10, PricePerStock: 0,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateNegotiation_DBError(t *testing.T) {
	s, mainMock, _, _, _, mPort, mSec := newTestServer(t)
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(1000)))
	mainMock.ExpectQuery("SELECT COALESCE.*FROM otc_contracts").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mainMock.ExpectQuery("SELECT COALESCE.*FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mainMock.ExpectQuery("INSERT INTO otc_negotiations").
		WillReturnError(sql.ErrConnDone)
	_, err := s.CreateNegotiation(context.Background(), &pb.CreateNegotiationRequest{
		Ticker: "AAPL", Amount: 10, PricePerStock: 100.0, Currency: "RSD",
		SettlementDate: "2027-06-01",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ===== ListNegotiations =====

func TestListNegotiations_Empty(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT id FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	resp, err := s.ListNegotiations(context.Background(), &pb.ListNegotiationsRequest{
		CallerId: 5, CallerType: "CLIENT",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Negotiations)
}

func TestListNegotiations_WithResults(t *testing.T) {
	s, mainMock, empMock, clientMock, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT id FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)).AddRow(int64(2)))
	addFetchNegotiationRows(mainMock, empMock, clientMock, 1, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_SELLER")
	addFetchNegotiationRows(mainMock, empMock, clientMock, 2, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_BUYER")
	resp, err := s.ListNegotiations(context.Background(), &pb.ListNegotiationsRequest{
		CallerId: 10, CallerType: "EMPLOYEE",
	})
	require.NoError(t, err)
	assert.Len(t, resp.Negotiations, 2)
}

func TestListNegotiations_DBError(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT id FROM otc_negotiations").
		WillReturnError(sql.ErrConnDone)
	_, err := s.ListNegotiations(context.Background(), &pb.ListNegotiationsRequest{})
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ===== GetNegotiation =====

func TestGetNegotiation_Happy(t *testing.T) {
	s, mainMock, empMock, clientMock, _, _, _ := newTestServer(t)
	addFetchNegotiationRows(mainMock, empMock, clientMock, 5, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_SELLER")
	resp, err := s.GetNegotiation(context.Background(), &pb.GetNegotiationRequest{NegotiationId: 5})
	require.NoError(t, err)
	assert.Equal(t, int64(5), resp.Id)
	assert.Equal(t, "PENDING_SELLER", resp.Status)
}

func TestGetNegotiation_NotFound(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT id, ticker").
		WillReturnRows(sqlmock.NewRows(negotiationColumns()))
	_, err := s.GetNegotiation(context.Background(), &pb.GetNegotiationRequest{NegotiationId: 999})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ===== CounterOffer =====

func TestCounterOffer_NotFound(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}))
	mainMock.ExpectRollback()
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 999, CallerId: 10, CallerType: "EMPLOYEE", SettlementDate: "2027-01-01",
	})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCounterOffer_NotParticipant(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))
	mainMock.ExpectRollback()
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 99, CallerType: "CLIENT", SettlementDate: "2027-01-01",
	})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestCounterOffer_NotYourTurn(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_BUYER"))
	mainMock.ExpectRollback()
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE", SettlementDate: "2027-01-01",
	})
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestCounterOffer_TerminalState(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "ACCEPTED"))
	mainMock.ExpectRollback()
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE", SettlementDate: "2027-01-01",
	})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestCounterOffer_Happy(t *testing.T) {
	s, mainMock, empMock, clientMock, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))
	mainMock.ExpectExec("UPDATE otc_negotiations").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mainMock.ExpectCommit()
	addFetchNegotiationRows(mainMock, empMock, clientMock, 1, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_BUYER")
	resp, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
		Amount: 90, PricePerStock: 155.0, SettlementDate: "2026-06-15",
	})
	require.NoError(t, err)
	assert.Equal(t, "PENDING_BUYER", resp.Status)
}

func TestCounterOffer_BuyerTurn(t *testing.T) {
	s, mainMock, empMock, clientMock, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_BUYER"))
	mainMock.ExpectExec("UPDATE otc_negotiations").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mainMock.ExpectCommit()
	addFetchNegotiationRows(mainMock, empMock, clientMock, 1, 10, 20, "EMPLOYEE", "CLIENT", "PENDING_SELLER")
	resp, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 20, CallerType: "CLIENT",
		Amount: 80, PricePerStock: 160.0, SettlementDate: "2026-07-01",
	})
	require.NoError(t, err)
	assert.Equal(t, "PENDING_SELLER", resp.Status)
}

// ===== AcceptNegotiation =====

func TestAcceptNegotiation_NotFound(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status",
			"ticker", "amount", "premium", "currency", "settlement_date", "price_per_stock"}))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 999, CallerId: 10, CallerType: "EMPLOYEE",
	})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestAcceptNegotiation_NotParticipant(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 99, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestAcceptNegotiation_TerminalState(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "REJECTED", "AAPL", "USD", 5, 10))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestAcceptNegotiation_WrongTurn(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	// status=PENDING_SELLER but the buyer tries to accept
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 20, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestAcceptNegotiation_InsufficientShares(t *testing.T) {
	s, mainMock, _, _, _, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	// seller (10) accepts; amount=10, but portfolio only has 5
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 10, 10))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(5)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "free shares")
}

func TestAcceptNegotiation_InsufficientFunds(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	// premium=100, buyer available_balance=50
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 100))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(100)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))
	mAcc.ExpectQuery("SELECT available_balance FROM accounts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(50)))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "Insufficient funds")
}

func TestAcceptNegotiation_HappyPath(t *testing.T) {
	s, mainMock, _, mCli, mAcc, mPort, mSec := newTestServer(t)
	// seller (10) accepts PENDING_SELLER; buyer is 20
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(50)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id"). // findAccount buyer
									WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))
	mAcc.ExpectQuery("SELECT available_balance FROM accounts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(500)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id"). // findAccount seller
									WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance").
		WillReturnResult(sqlmock.NewResult(0, 1)) // deduct buyer premium
	mAcc.ExpectExec("UPDATE accounts SET balance").
		WillReturnResult(sqlmock.NewResult(0, 1)) // credit seller premium
	// settlement_date and price_per_stock are now in the initial FOR UPDATE SELECT — no separate query
	mainMock.ExpectQuery("INSERT INTO otc_contracts").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mainMock.ExpectExec("UPDATE otc_negotiations").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectCommit()
	// fetchNegotiationByID
	mainMock.ExpectQuery("SELECT id, ticker").
		WillReturnRows(sqlmock.NewRows(negotiationColumns()).
			AddRow(int64(1), "AAPL", int64(10), "CLIENT", int64(20), "CLIENT",
				int32(5), float64(100.0), "2026-12-31", float64(10.0), "USD",
				time.Now(), nil, nil, "ACCEPTED"))
	mCli.ExpectQuery("SELECT first_name"). // getUserName seller (CLIENT)
						WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Seller Name"))
	mCli.ExpectQuery("SELECT first_name"). // getUserName buyer (CLIENT)
						WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Buyer Name"))
	// modifiedByID=nil → no third getUserName call

	resp, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT", BuyerAccountId: 0,
	})

	require.NoError(t, err)
	assert.Equal(t, "ACCEPTED", resp.Status)
	assert.NoError(t, mainMock.ExpectationsWereMet())
	assert.NoError(t, mAcc.ExpectationsWereMet())
}

// ===== RejectNegotiation =====

func TestRejectNegotiation_NotFound(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}))
	mainMock.ExpectRollback()
	_, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 999, CallerId: 10, CallerType: "EMPLOYEE",
	})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestRejectNegotiation_NotParticipant(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))
	mainMock.ExpectRollback()
	_, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 1, CallerId: 99, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestRejectNegotiation_TerminalState(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "ACCEPTED"))
	mainMock.ExpectRollback()
	_, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
	})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestRejectNegotiation_Happy(t *testing.T) {
	s, mainMock, empMock, clientMock, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))
	mainMock.ExpectExec("UPDATE otc_negotiations").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mainMock.ExpectCommit()
	addFetchNegotiationRows(mainMock, empMock, clientMock, 1, 10, 20, "EMPLOYEE", "CLIENT", "REJECTED")
	resp, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
	})
	require.NoError(t, err)
	assert.Equal(t, "REJECTED", resp.Status)
}

// ===== ListContracts =====

func TestListContracts_Empty(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "negotiation_id", "seller_id", "seller_type", "buyer_id", "buyer_type",
			"ticker", "amount", "strike_price", "premium", "currency",
			"settlement_date", "status", "created_at",
		}))
	resp, err := s.ListContracts(context.Background(), &pb.ListContractsRequest{
		CallerId: 10, CallerType: "CLIENT",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Contracts)
	assert.NoError(t, mainMock.ExpectationsWereMet())
}

func TestListContracts_CallerSeesOwnContracts(t *testing.T) {
	s, mainMock, _, mCli, _, _, mSec := newTestServer(t)
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "negotiation_id", "seller_id", "seller_type", "buyer_id", "buyer_type",
			"ticker", "amount", "strike_price", "premium", "currency",
			"settlement_date", "status", "created_at",
		}).AddRow(
			int64(1), int64(1), int64(10), "CLIENT", int64(20), "CLIENT",
			"AAPL", int32(5), float64(100.0), float64(10.0), "USD",
			"2026-12-31", "ACTIVE", time.Now(),
		))
	mSec.ExpectQuery("SELECT price FROM listing WHERE ticker").
		WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(float64(150.0)))
	mCli.ExpectQuery("SELECT first_name"). // getUserName seller
						WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Seller"))
	mCli.ExpectQuery("SELECT first_name"). // getUserName buyer
						WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Buyer"))

	resp, err := s.ListContracts(context.Background(), &pb.ListContractsRequest{
		CallerId: 10, CallerType: "CLIENT",
	})
	require.NoError(t, err)
	require.Len(t, resp.Contracts, 1)
	assert.Equal(t, "AAPL", resp.Contracts[0].Ticker)
	assert.Equal(t, float64(250.0), resp.Contracts[0].Profit) // (150-100)*5 = 250
}

func TestListContracts_WithStatusFilter(t *testing.T) {
	s, mainMock, _, mCli, _, _, mSec := newTestServer(t)
	// Expect the query to contain both the OR clause and AND status filter.
	// The correct SQL wraps the OR pair in outer parens so AND applies to both.
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "negotiation_id", "seller_id", "seller_type", "buyer_id", "buyer_type",
			"ticker", "amount", "strike_price", "premium", "currency",
			"settlement_date", "status", "created_at",
		}).AddRow(
			int64(2), int64(1), int64(10), "CLIENT", int64(20), "CLIENT",
			"AAPL", int32(5), float64(100.0), float64(10.0), "USD",
			"2026-12-31", "ACTIVE", time.Now(),
		))
	mSec.ExpectQuery("SELECT price FROM listing WHERE ticker").
		WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(float64(150.0)))
	mCli.ExpectQuery("SELECT first_name"). // seller name
						WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Seller"))
	mCli.ExpectQuery("SELECT first_name"). // buyer name
						WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Buyer"))

	resp, err := s.ListContracts(context.Background(), &pb.ListContractsRequest{
		CallerId: 10, CallerType: "CLIENT", StatusFilter: "ACTIVE",
	})
	require.NoError(t, err)
	require.Len(t, resp.Contracts, 1)
	assert.Equal(t, "ACTIVE", resp.Contracts[0].Status)
	assert.NoError(t, mainMock.ExpectationsWereMet())
}

// ===== ExerciseContract =====

func TestExerciseContract_NotFound(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnError(sql.ErrNoRows)
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestExerciseContract_NotBuyer(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 99, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestExerciseContract_NotActive(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{
			"seller_id", "seller_type", "buyer_id", "buyer_type", "status",
			"ticker", "amount", "strike_price", "currency", "settlement_date",
		}).AddRow(int64(10), "CLIENT", int64(20), "CLIENT", "EXPIRED",
			"AAPL", int32(5), float64(100.0), "USD", future))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestExerciseContract_InsufficientFunds(t *testing.T) {
	s, mainMock, _, _, mAcc, _, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1: atomic UPDATE returns 0 rows (insufficient funds)
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mainMock.ExpectExec("INSERT INTO otc_saga_log"). // step 1 FAILED
								WillReturnResult(sqlmock.NewResult(1, 1))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "Insufficient funds")
	assert.NoError(t, mainMock.ExpectationsWereMet())
}

func TestExerciseContract_SellerNoShares(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1: atomic UPDATE returns 1 row (success)
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log"). // step 1 SUCCESS
								WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 2: atomic UPDATE returns 0 rows (no free shares)
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mainMock.ExpectExec("INSERT INTO otc_saga_log"). // step 2 FAILED
								WillReturnResult(sqlmock.NewResult(1, 1))
	// comp1: restore buyer available_balance
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance \\+").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log"). // step 1 COMPENSATED
								WillReturnResult(sqlmock.NewResult(1, 1))
	mainMock.ExpectRollback()

	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "free shares")
	assert.NoError(t, mainMock.ExpectationsWereMet())
	assert.NoError(t, mAcc.ExpectationsWereMet())
	assert.NoError(t, mPort.ExpectationsWereMet())
}

func TestExerciseContract_HappyPath(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	// Step 1: atomic UPDATE returns 1 row (success)
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").
		WillReturnResult(sqlmock.NewResult(1, 1)) // step 1 SUCCESS

	// Step 2: atomic UPDATE returns 1 row (success)
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").
		WillReturnResult(sqlmock.NewResult(1, 1)) // step 2 SUCCESS

	// Step 3
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id"). // findAccount seller
									WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance - "). // debit buyer
									WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+"). // credit seller
									WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").
		WillReturnResult(sqlmock.NewResult(1, 1)) // step 3 SUCCESS

	// Step 4
	mPort.ExpectExec(`(?s)UPDATE portfolio_entry.*public_amount = GREATEST`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mPort.ExpectExec("DELETE FROM portfolio_entry").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mPort.ExpectExec("INSERT INTO portfolio_entry").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").
		WillReturnResult(sqlmock.NewResult(1, 1)) // step 4 SUCCESS

	// Step 5: double-check buyer holding, then mark EXERCISED inside tx
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(5)))
	mainMock.ExpectExec("UPDATE otc_contracts SET status").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectCommit()
	mainMock.ExpectExec("INSERT INTO otc_saga_log").
		WillReturnResult(sqlmock.NewResult(1, 1)) // step 5 SUCCESS

	resp, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, "EXERCISED", resp.Status)
	assert.NotEmpty(t, resp.ExecutedAt)
	assert.NoError(t, mainMock.ExpectationsWereMet())
	assert.NoError(t, mAcc.ExpectationsWereMet())
	assert.NoError(t, mPort.ExpectationsWereMet())
}

func TestExerciseContract_PublicAmountDecrement(t *testing.T) {
	// Verifies that Step 4 seller UPDATE includes public_amount correction.
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	// Step 1: atomic UPDATE
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))

	// Step 2: atomic UPDATE
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))

	// Step 3
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))

	// Step 4: seller UPDATE must contain public_amount = GREATEST(...)
	mPort.ExpectExec(`(?s)UPDATE portfolio_entry.*public_amount = GREATEST`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mPort.ExpectExec("DELETE FROM portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 0))
	mPort.ExpectExec("INSERT INTO portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))

	// Step 5: double-check + EXERCISED inside tx
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(5)))
	mainMock.ExpectExec("UPDATE otc_contracts SET status").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectCommit()
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, "EXERCISED", resp.Status)
	assert.NoError(t, mPort.ExpectationsWereMet())
}

func TestExerciseContract_CompensationRetry(t *testing.T) {
	// Verifies that comp1 retries when the first DB exec fails.
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	// Step 1: atomic UPDATE succeeds (1 row)
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 1 SUCCESS

	// Step 2: atomic UPDATE returns 0 rows (no free shares) → FAILED
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 2 FAILED

	// comp1: first attempt fails (network error), second succeeds
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance \\+").
		WillReturnError(fmt.Errorf("connection reset by peer"))
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance \\+").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 1 COMPENSATED
	mainMock.ExpectRollback()

	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "free shares")
	assert.NoError(t, mainMock.ExpectationsWereMet())
	assert.NoError(t, mAcc.ExpectationsWereMet())
	assert.NoError(t, mPort.ExpectationsWereMet())
}

// ===== GetMarket =====

func TestGetMarket_ClientSeesOtherClients(t *testing.T) {
	s, _, _, _, _, mPort, _ := newTestServer(t)
	mPort.ExpectQuery("SELECT .* FROM portfolio_entry").
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "user_type", "listing_id", "public_amount", "last_modified",
		}))
	resp, err := s.GetMarket(context.Background(), &pb.GetMarketRequest{
		CallerId: 10, CallerType: "CLIENT",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.NoError(t, mPort.ExpectationsWereMet())
}

func TestGetMarket_EmployeeSeesBank(t *testing.T) {
	s, _, _, _, _, mPort, _ := newTestServer(t)
	mPort.ExpectQuery("SELECT .* FROM portfolio_entry").
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "user_type", "listing_id", "public_amount", "last_modified",
		}))
	resp, err := s.GetMarket(context.Background(), &pb.GetMarketRequest{
		CallerId: 5, CallerType: "EMPLOYEE",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.NoError(t, mPort.ExpectationsWereMet())
}

func TestGetMarket_CurrencyFromJoin(t *testing.T) {
	s, _, _, mCli, _, mPort, mSec := newTestServer(t)
	mPort.ExpectQuery("SELECT .* FROM portfolio_entry").
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "user_type", "listing_id", "public_amount", "last_modified",
		}).AddRow(int64(99), "CLIENT", int64(42), int32(10), time.Now()))
	// After Bug 3 fix: JOIN stock_exchanges returns real currency (not hardcoded "")
	mSec.ExpectQuery("SELECT l.ticker, l.name, l.price, se.currency").
		WillReturnRows(sqlmock.NewRows([]string{"ticker", "name", "price", "currency"}).
			AddRow("AAPL", "Apple Inc", float64(150.0), "USD"))
	mCli.ExpectQuery("SELECT first_name").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("John Doe"))

	resp, err := s.GetMarket(context.Background(), &pb.GetMarketRequest{
		CallerId: 10, CallerType: "CLIENT",
	})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "USD", resp.Items[0].Currency)
	assert.Equal(t, "AAPL", resp.Items[0].Ticker)
	assert.Equal(t, "John Doe", resp.Items[0].OwnerName)
	assert.NoError(t, mPort.ExpectationsWereMet())
	assert.NoError(t, mSec.ExpectationsWereMet())
}

func TestGetMarket_DBError(t *testing.T) {
	s, _, _, _, _, mPort, _ := newTestServer(t)
	mPort.ExpectQuery("SELECT .* FROM portfolio_entry").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.GetMarket(context.Background(), &pb.GetMarketRequest{
		CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetMarket_ScanError(t *testing.T) {
	s, _, _, _, _, mPort, _ := newTestServer(t)
	mPort.ExpectQuery("SELECT .* FROM portfolio_entry").
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "user_type", "listing_id", "public_amount", "last_modified",
		}).AddRow("bad", "CLIENT", int64(42), int32(10), time.Now()))
	_, err := s.GetMarket(context.Background(), &pb.GetMarketRequest{
		CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ===== calcContractProfit =====

func TestCalcContractProfit_Exercised(t *testing.T) {
	s, _, _, _, _, _, mSec := newTestServer(t)
	mSec.ExpectQuery("SELECT price FROM listing WHERE ticker").
		WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(float64(200.0)))
	profit := s.calcContractProfit("AAPL", 100.0, 5, 50.0, "EXERCISED")
	// (200-100)*5 - 50 = 450
	assert.Equal(t, float64(450.0), profit)
	assert.NoError(t, mSec.ExpectationsWereMet())
}

func TestCalcContractProfit_DBError(t *testing.T) {
	s, _, _, _, _, _, mSec := newTestServer(t)
	mSec.ExpectQuery("SELECT price FROM listing WHERE ticker").
		WillReturnError(fmt.Errorf("db error"))
	profit := s.calcContractProfit("AAPL", 100.0, 5, 0.0, "ACTIVE")
	assert.Equal(t, float64(0), profit)
}

// ===== ListContracts scan error =====

func TestListContracts_ScanError(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "negotiation_id", "seller_id", "seller_type", "buyer_id", "buyer_type",
			"ticker", "amount", "strike_price", "premium", "currency",
			"settlement_date", "status", "created_at",
		}).AddRow(
			"bad", int64(1), int64(10), "CLIENT", int64(20), "CLIENT",
			"AAPL", int32(5), float64(100.0), float64(10.0), "USD",
			"2026-12-31", "ACTIVE", time.Now(),
		))
	_, err := s.ListContracts(context.Background(), &pb.ListContractsRequest{
		CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ===== ListNegotiations scan error =====

func TestListNegotiations_ScanError(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT id FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("bad"))
	_, err := s.ListNegotiations(context.Background(), &pb.ListNegotiationsRequest{
		CallerId: 5, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ===== CounterOffer additional paths =====

func TestCounterOffer_InternalError(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnError(fmt.Errorf("connection reset"))
	mainMock.ExpectRollback()
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE", SettlementDate: "2027-01-01",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCounterOffer_UpdateFails(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))
	mainMock.ExpectExec("UPDATE otc_negotiations").
		WillReturnError(fmt.Errorf("db error"))
	mainMock.ExpectRollback()
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
		Amount: 90, PricePerStock: 155.0, SettlementDate: "2026-06-15",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ===== RejectNegotiation additional paths =====

func TestRejectNegotiation_InternalError(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnError(fmt.Errorf("connection reset"))
	mainMock.ExpectRollback()
	_, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestRejectNegotiation_UpdateFails(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))
	mainMock.ExpectExec("UPDATE otc_negotiations").
		WillReturnError(fmt.Errorf("db error"))
	mainMock.ExpectRollback()
	_, err := s.RejectNegotiation(context.Background(), &pb.RejectNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "EMPLOYEE",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ===== AcceptNegotiation additional paths =====

func TestAcceptNegotiation_InternalError(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnError(fmt.Errorf("connection reset"))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestAcceptNegotiation_WrongTurn_Buyer(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectBegin()
	// PENDING_BUYER but the seller tries to accept
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_BUYER", "AAPL", "USD", 5, 10))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "waiting for buyer")
}

func TestAcceptNegotiation_ListingIDNotFound(t *testing.T) {
	s, mainMock, _, _, _, _, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	// Return 0 rows → ErrNoRows → listingIDForTicker returns fmt.Errorf
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "ticker")
}

func TestAcceptNegotiation_PortfolioInternalError(t *testing.T) {
	s, mainMock, _, _, _, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnError(fmt.Errorf("db error"))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "seller portfolio")
}

func TestAcceptNegotiation_UnsupportedCurrency(t *testing.T) {
	s, mainMock, _, _, _, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "XYZ", 5, 10))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(100)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "unsupported currency")
}

func TestAcceptNegotiation_FindBuyerAccountFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(100)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	// findAccount buyer returns 0 rows → ErrNoRows
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT", BuyerAccountId: 0,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "buyer account")
}

func TestAcceptNegotiation_BuyerBalanceQueryFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(100)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))
	mAcc.ExpectQuery("SELECT available_balance FROM accounts WHERE id").
		WillReturnError(fmt.Errorf("db error"))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "buyer balance")
}

func TestAcceptNegotiation_FindSellerAccountFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(100)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	// findAccount buyer
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))
	mAcc.ExpectQuery("SELECT available_balance FROM accounts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(500)))
	// findAccount seller returns 0 rows
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "seller account")
}

func TestAcceptNegotiation_DeductBuyerPremiumFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(100)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))
	mAcc.ExpectQuery("SELECT available_balance FROM accounts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(500)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	// deduct buyer premium fails
	mAcc.ExpectExec("UPDATE accounts SET balance").
		WillReturnError(fmt.Errorf("db error"))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "deduct premium")
}

func TestAcceptNegotiation_CreditSellerFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(100)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))
	mAcc.ExpectQuery("SELECT available_balance FROM accounts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(500)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance").
		WillReturnResult(sqlmock.NewResult(0, 1)) // deduct buyer succeeds
	mAcc.ExpectExec("UPDATE accounts SET balance").
		WillReturnError(fmt.Errorf("db error")) // credit seller fails
	// compensation: restore buyer
	mAcc.ExpectExec("UPDATE accounts SET balance").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "credit premium")
}

func TestAcceptNegotiation_InsertContractFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(100)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))
	mAcc.ExpectQuery("SELECT available_balance FROM accounts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(500)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectQuery("INSERT INTO otc_contracts").
		WillReturnError(fmt.Errorf("db error"))
	// compensation: restore buyer and seller
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "create contract")
}

func TestAcceptNegotiation_UpdateNegotiationFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id").
		WillReturnRows(acceptNegRow(10, 20, "CLIENT", "CLIENT", "PENDING_SELLER", "AAPL", "USD", 5, 10))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(100)))
	mainMock.ExpectQuery("SELECT COALESCE.*SUM").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))
	mAcc.ExpectQuery("SELECT available_balance FROM accounts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(500)))
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectQuery("INSERT INTO otc_contracts").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mainMock.ExpectExec("UPDATE otc_negotiations").
		WillReturnError(fmt.Errorf("db error"))
	// compensation: restore buyer and seller
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectRollback()
	_, err := s.AcceptNegotiation(context.Background(), &pb.AcceptNegotiationRequest{
		NegotiationId: 1, CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "accept negotiation")
}

// ===== ExerciseContract additional paths =====

func TestExerciseContract_InternalError(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnError(fmt.Errorf("connection reset"))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestExerciseContract_SettlementDatePassed(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	// settlement date is 2 days in the past (beyond 24h grace)
	past := time.Now().Add(-48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, past))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "settlement date")
}

func TestExerciseContract_UnsupportedCurrency(t *testing.T) {
	s, mainMock, _, _, _, _, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{
			"seller_id", "seller_type", "buyer_id", "buyer_type", "status",
			"ticker", "amount", "strike_price", "currency", "settlement_date",
		}).AddRow(int64(10), "CLIENT", int64(20), "CLIENT", "ACTIVE",
			"AAPL", int32(5), float64(100.0), "XYZ", future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "unsupported currency")
}

func TestExerciseContract_ListingIDNotFound(t *testing.T) {
	s, mainMock, _, _, _, _, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	// Return 0 rows → ErrNoRows → listingIDForTicker error
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "ticker")
}

func TestExerciseContract_FindBuyerAccountFails(t *testing.T) {
	s, mainMock, _, _, mAcc, _, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// findAccount buyer returns 0 rows
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 0,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "buyer account")
}

func TestExerciseContract_Step1UpdateFails(t *testing.T) {
	s, mainMock, _, _, mAcc, _, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1: atomic UPDATE returns error
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnError(fmt.Errorf("db error"))
	mainMock.ExpectExec("INSERT INTO otc_saga_log"). // step 1 FAILED
								WillReturnResult(sqlmock.NewResult(1, 1))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "step 1")
}

func TestExerciseContract_Step2InternalError(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1: atomic UPDATE succeeds (1 row)
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 2: atomic UPDATE returns error
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnError(fmt.Errorf("db error"))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 2 FAILED
	// comp1: restore buyer available_balance
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance \\+").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 1 COMPENSATED
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "step 2")
}

func TestExerciseContract_Step3SellerAccountNotFound(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1: atomic UPDATE succeeds
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 2: atomic UPDATE succeeds
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 3: findAccount seller returns 0 rows
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 3 FAILED
	// comp2: retryExec portfolio reserved_amount + sagaLog
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 2 COMPENSATED
	// comp1
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance \\+").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 1 COMPENSATED
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "step 3")
	assert.NoError(t, mainMock.ExpectationsWereMet())
}

func TestExerciseContract_Step3DebitBuyerFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1: atomic UPDATE succeeds
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 2: atomic UPDATE succeeds
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 3: findAccount seller ok, debit buyer fails
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance - ").
		WillReturnError(fmt.Errorf("db error"))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 3 FAILED
	// comp2: retryExec portfolio + sagaLog
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 2 COMPENSATED
	// comp1
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance \\+").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 1 COMPENSATED
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "step 3")
}

func TestExerciseContract_Step3CreditSellerFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1: atomic UPDATE succeeds
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 2: atomic UPDATE succeeds
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 3: findAccount seller ok, debit buyer ok, credit seller fails
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnError(fmt.Errorf("db error"))
	// inner restore buyer balance
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 3 FAILED
	// comp2: retryExec portfolio + sagaLog
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 2 COMPENSATED
	// comp1
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance \\+").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 1 COMPENSATED
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "step 3")
}

func TestExerciseContract_Step4SellerUpdateFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1: atomic UPDATE succeeds
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 2: atomic UPDATE succeeds
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 3
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance - ").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 4: seller portfolio UPDATE fails
	mPort.ExpectExec(`(?s)UPDATE portfolio_entry.*public_amount = GREATEST`).
		WillReturnError(fmt.Errorf("db error"))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 4 FAILED
	// comp3: retryExec buyer restore + seller restore
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance -").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// comp2: retryExec portfolio + sagaLog
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// comp1
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance \\+").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "step 4")
}

func TestExerciseContract_Step4BuyerUpsertFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1: atomic UPDATE succeeds
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 2: atomic UPDATE succeeds
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 3
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance - ").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 4: seller UPDATE ok, DELETE ok, buyer upsert fails
	mPort.ExpectExec(`(?s)UPDATE portfolio_entry.*public_amount = GREATEST`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mPort.ExpectExec("DELETE FROM portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 0))
	mPort.ExpectExec("INSERT INTO portfolio_entry").WillReturnError(fmt.Errorf("db error"))
	// inner comp: restore seller portfolio
	mPort.ExpectExec("UPDATE portfolio_entry SET amount = amount \\+").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 4 COMPENSATED
	// comp3
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance -").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// comp2: retryExec portfolio + sagaLog
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// comp1
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance \\+").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "step 4")
}

func TestExerciseContract_Step5UpdateContractFails(t *testing.T) {
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(contractRow(10, 20, future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1: atomic UPDATE succeeds
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 2: atomic UPDATE succeeds
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 3
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance - ").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 4
	mPort.ExpectExec(`(?s)UPDATE portfolio_entry.*public_amount = GREATEST`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mPort.ExpectExec("DELETE FROM portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 0))
	mPort.ExpectExec("INSERT INTO portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 5: double-check SELECT passes, then tx UPDATE fails
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(5)))
	mainMock.ExpectExec("UPDATE otc_contracts SET status").
		WillReturnError(fmt.Errorf("db error"))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1)) // step 5 FAILED
	// comp4: seller restore + buyer reduce
	mPort.ExpectExec("UPDATE portfolio_entry SET amount = amount \\+").WillReturnResult(sqlmock.NewResult(0, 1))
	mPort.ExpectExec("UPDATE portfolio_entry SET amount = amount -").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// comp3
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance -").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// comp2: retryExec portfolio + sagaLog
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// comp1
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance \\+").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	mainMock.ExpectRollback()
	_, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "step 5")
}

func TestExerciseContract_EmployeeSeller(t *testing.T) {
	// EMPLOYEE seller → portfolioUserID returns 0 (bank shared portfolio)
	s, mainMock, _, _, mAcc, mPort, mSec := newTestServer(t)
	future := time.Now().Add(48 * time.Hour)
	mainMock.ExpectQuery("SELECT step, status FROM otc_saga_log").
		WillReturnRows(sqlmock.NewRows([]string{"step", "status"}))
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{
			"seller_id", "seller_type", "buyer_id", "buyer_type", "status",
			"ticker", "amount", "strike_price", "currency", "settlement_date",
		}).AddRow(int64(5), "EMPLOYEE", int64(20), "CLIENT", "ACTIVE",
			"AAPL", int32(5), float64(100.0), "USD", future))
	mSec.ExpectQuery("SELECT id FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// Step 1: atomic UPDATE succeeds
	mAcc.ExpectExec("UPDATE accounts SET available_balance = available_balance - ").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 2: atomic UPDATE succeeds (portfolioUserID(5, "EMPLOYEE") = 0)
	mPort.ExpectExec("UPDATE portfolio_entry SET reserved_amount").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 3: findAccount seller (portfolioUserID=0)
	mAcc.ExpectQuery("SELECT id FROM accounts WHERE owner_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(200)))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance - ").WillReturnResult(sqlmock.NewResult(0, 1))
	mAcc.ExpectExec("UPDATE accounts SET balance = balance \\+").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 4
	mPort.ExpectExec(`(?s)UPDATE portfolio_entry.*public_amount = GREATEST`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mPort.ExpectExec("DELETE FROM portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 0))
	mPort.ExpectExec("INSERT INTO portfolio_entry").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))
	// Step 5: double-check + EXERCISED inside tx
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(5)))
	mainMock.ExpectExec("UPDATE otc_contracts SET status").WillReturnResult(sqlmock.NewResult(0, 1))
	mainMock.ExpectCommit()
	mainMock.ExpectExec("INSERT INTO otc_saga_log").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.ExerciseContract(context.Background(), &pb.ExerciseContractRequest{
		ContractId: 1, CallerId: 20, CallerType: "CLIENT", BuyerAccountId: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, "EXERCISED", resp.Status)
}

// ===== fetchNegotiationByID internal error =====

func TestGetNegotiation_InternalError(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT id, ticker").
		WillReturnError(fmt.Errorf("connection reset"))
	_, err := s.GetNegotiation(context.Background(), &pb.GetNegotiationRequest{NegotiationId: 1})
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ===== ListNegotiations fetchNegotiationByID error in loop =====

func TestListNegotiations_FetchError(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT id FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	// fetchNegotiationByID returns internal error (non-ErrNoRows)
	mainMock.ExpectQuery("SELECT id, ticker").
		WillReturnError(fmt.Errorf("connection reset"))
	_, err := s.ListNegotiations(context.Background(), &pb.ListNegotiationsRequest{
		CallerId: 5, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ===== CounterOffer: PENDING_SELLER but buyer tries =====

func TestCounterOffer_BuyerTriesWhenPendingSeller(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	// status=PENDING_SELLER but the buyer makes counter offer
	mainMock.ExpectBegin()
	mainMock.ExpectQuery("SELECT seller_id, seller_type, buyer_id, buyer_type, status").
		WillReturnRows(sqlmock.NewRows([]string{"seller_id", "seller_type", "buyer_id", "buyer_type", "status"}).
			AddRow(int64(10), "EMPLOYEE", int64(20), "CLIENT", "PENDING_SELLER"))
	mainMock.ExpectRollback()
	_, err := s.CounterOffer(context.Background(), &pb.CounterOfferRequest{
		NegotiationId: 1, CallerId: 20, CallerType: "CLIENT", SettlementDate: "2027-01-01",
	})
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "waiting for seller")
}

// ===== ListContracts QueryContext error =====

func TestListContracts_QueryError(t *testing.T) {
	s, mainMock, _, _, _, _, _ := newTestServer(t)
	mainMock.ExpectQuery("SELECT .* FROM otc_contracts").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.ListContracts(context.Background(), &pb.ListContractsRequest{
		CallerId: 10, CallerType: "CLIENT",
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ===== GetMarket: securities query fails for a row — continue =====

func TestGetMarket_SecQueryFails(t *testing.T) {
	s, _, _, _, _, mPort, mSec := newTestServer(t)
	mPort.ExpectQuery("SELECT .* FROM portfolio_entry").
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "user_type", "listing_id", "public_amount", "last_modified",
		}).AddRow(int64(99), "CLIENT", int64(42), int32(10), time.Now()))
	// securities query fails → continue (item skipped)
	mSec.ExpectQuery("SELECT l.ticker, l.name, l.price, se.currency").
		WillReturnError(fmt.Errorf("db error"))
	resp, err := s.GetMarket(context.Background(), &pb.GetMarketRequest{
		CallerId: 10, CallerType: "CLIENT",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Items) // item was skipped
	assert.NoError(t, mPort.ExpectationsWereMet())
	assert.NoError(t, mSec.ExpectationsWereMet())
}
