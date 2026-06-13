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
	"google.golang.org/grpc/status"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// interbankNegColumns returns the columns expected by fetchInterbankNegotiationByID.
func interbankNegColumns() []string {
	return []string{
		"id", "ticker", "amount", "price_per_stock", "settlement_date", "premium", "currency", "status",
		"buyer_routing_number", "buyer_external_id",
		"seller_routing_number", "seller_external_id",
		"creator_routing_number", "creator_external_id",
	}
}

// addFetchInterbankRows sets up the fetchInterbankNegotiationByID mock expectation.
func addFetchInterbankRows(m sqlmock.Sqlmock, id int64, negStatus string) {
	m.ExpectQuery("SELECT id, ticker, amount").
		WillReturnRows(sqlmock.NewRows(interbankNegColumns()).
			AddRow(id, "AAPL", int32(10), 100.0, "2026-12-31", 5.0, "RSD", negStatus,
				sql.NullInt32{Int32: 444, Valid: true}, sql.NullString{String: "42", Valid: true},
				sql.NullInt32{Int32: 888, Valid: true}, sql.NullString{String: "7", Valid: true},
				sql.NullInt32{Int32: 444, Valid: true}, sql.NullString{String: "42", Valid: true},
			))
}

// addLookupByLocalID adds a mock expectation for the primary key lookup in lookupInterbankNegotiation.
func addLookupByLocalID(m sqlmock.Sqlmock, id int64, negStatus string) {
	m.ExpectQuery("SELECT id, status FROM otc_negotiations WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(id, negStatus))
}

// addLookupNotFound adds mock expectations for a lookup that finds nothing (both paths return no rows).
func addLookupNotFound(m sqlmock.Sqlmock) {
	// Primary: by local ID
	m.ExpectQuery("SELECT id, status FROM otc_negotiations WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}))
	// Fallback: by creator key
	m.ExpectQuery("SELECT id, status FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}))
}

// ── CreateInterbankNegotiation ────────────────────────────────────────────────

func TestCreateInterbankNegotiation_NoTicker(t *testing.T) {
	s, _, _, _, _, _, _ := newTestServer(t)
	_, err := s.CreateInterbankNegotiation(context.Background(), &pb.CreateInterbankNegotiationRequest{
		Amount: 10, SettlementDate: "2026-12-31",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateInterbankNegotiation_ZeroAmount(t *testing.T) {
	s, _, _, _, _, _, _ := newTestServer(t)
	_, err := s.CreateInterbankNegotiation(context.Background(), &pb.CreateInterbankNegotiationRequest{
		Ticker: "AAPL", SettlementDate: "2026-12-31",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateInterbankNegotiation_InvalidSellerID(t *testing.T) {
	s, _, _, _, _, _, _ := newTestServer(t)
	_, err := s.CreateInterbankNegotiation(context.Background(), &pb.CreateInterbankNegotiationRequest{
		Ticker: "AAPL", Amount: 10, SettlementDate: "2026-12-31",
		SellerExternalId: "not-a-number",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateInterbankNegotiation_Idempotent(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)

	// idempotency check: lookup by creatorExtId="42" (numeric, so tries local ID first)
	addLookupByLocalID(mOTC, 42, "PENDING_SELLER")
	addFetchInterbankRows(mOTC, 42, "PENDING_SELLER")

	resp, err := s.CreateInterbankNegotiation(context.Background(), &pb.CreateInterbankNegotiationRequest{
		Ticker: "AAPL", Amount: 10, SettlementDate: "2026-12-31",
		SellerExternalId:  "7",
		CreatorRoutingNumber: 444, CreatorExternalId: "42",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.LocalId)
}

func TestCreateInterbankNegotiation_InsertFails(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)

	// Idempotency lookup: numeric "7" tries local ID 7 first — not found, then creator key not found
	addLookupNotFound(mOTC)
	// INSERT fails
	mOTC.ExpectQuery("INSERT INTO otc_negotiations").
		WillReturnError(sql.ErrConnDone)

	_, err := s.CreateInterbankNegotiation(context.Background(), &pb.CreateInterbankNegotiationRequest{
		Ticker: "AAPL", Amount: 10, SettlementDate: "2026-12-31",
		SellerExternalId: "7",
		CreatorRoutingNumber: 444, CreatorExternalId: "9999",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── InterbankCounterOffer ─────────────────────────────────────────────────────

func TestInterbankCounterOffer_NotFound(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	addLookupNotFound(mOTC)

	_, err := s.InterbankCounterOffer(context.Background(), &pb.InterbankCounterOfferRequest{
		RoutingNumber: 444, ExternalId: "9999",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestInterbankCounterOffer_TerminalState(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	addLookupByLocalID(mOTC, 42, "ACCEPTED")

	_, err := s.InterbankCounterOffer(context.Background(), &pb.InterbankCounterOfferRequest{
		RoutingNumber: 444, ExternalId: "42",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestInterbankCounterOffer_NotBuyersTurn(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	addLookupByLocalID(mOTC, 42, "PENDING_SELLER")

	_, err := s.InterbankCounterOffer(context.Background(), &pb.InterbankCounterOfferRequest{
		RoutingNumber: 444, ExternalId: "42",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestInterbankCounterOffer_Success(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	addLookupByLocalID(mOTC, 42, "PENDING_BUYER")
	mOTC.ExpectExec("UPDATE otc_negotiations").WillReturnResult(sqlmock.NewResult(1, 1))
	addFetchInterbankRows(mOTC, 42, "PENDING_SELLER")

	resp, err := s.InterbankCounterOffer(context.Background(), &pb.InterbankCounterOfferRequest{
		RoutingNumber: 444, ExternalId: "42",
		PricePerUnit: 110, Premium: 5, Amount: 10, SettlementDate: "2026-12-31",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.LocalId)
}

// ── InterbankGetNegotiation ───────────────────────────────────────────────────

func TestInterbankGetNegotiation_NotFound(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	addLookupNotFound(mOTC)

	_, err := s.InterbankGetNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "9999",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestInterbankGetNegotiation_Success(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	addLookupByLocalID(mOTC, 42, "PENDING_SELLER")
	addFetchInterbankRows(mOTC, 42, "PENDING_SELLER")

	resp, err := s.InterbankGetNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "42",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.LocalId)
	assert.Equal(t, "AAPL", resp.Ticker)
}

// ── InterbankDeleteNegotiation ────────────────────────────────────────────────

func TestInterbankDeleteNegotiation_NotFound(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	addLookupNotFound(mOTC)

	_, err := s.InterbankDeleteNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "9999",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestInterbankDeleteNegotiation_AlreadyAccepted(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	addLookupByLocalID(mOTC, 42, "ACCEPTED")

	_, err := s.InterbankDeleteNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "42",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestInterbankDeleteNegotiation_AlreadyRejected_Idempotent(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	addLookupByLocalID(mOTC, 42, "REJECTED")

	resp, err := s.InterbankDeleteNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "42",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestInterbankDeleteNegotiation_Success(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	addLookupByLocalID(mOTC, 42, "PENDING_SELLER")
	mOTC.ExpectExec("UPDATE otc_negotiations SET status = 'REJECTED'").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.InterbankDeleteNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "42",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ── InterbankAcceptNegotiation ────────────────────────────────────────────────

func TestInterbankAcceptNegotiation_InvalidID(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectBegin()
	mOTC.ExpectRollback()

	_, err := s.InterbankAcceptNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "not-a-number",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestInterbankAcceptNegotiation_NotFound(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)

	mOTC.ExpectBegin()
	mOTC.ExpectQuery("SELECT id, status, ticker, currency").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "ticker", "currency", "settlement_date",
			"amount", "premium", "price_per_stock", "seller_id", "seller_type",
		}))
	mOTC.ExpectRollback()

	_, err := s.InterbankAcceptNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "42",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestInterbankAcceptNegotiation_NotBuyersTurn(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)

	mOTC.ExpectBegin()
	mOTC.ExpectQuery("SELECT id, status, ticker, currency").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "ticker", "currency", "settlement_date",
			"amount", "premium", "price_per_stock", "seller_id", "seller_type",
		}).AddRow(int64(42), "PENDING_SELLER", "AAPL", "RSD", "2026-12-31",
			int32(10), 5.0, 100.0, int64(7), "CLIENT"))
	mOTC.ExpectRollback()

	_, err := s.InterbankAcceptNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "42",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}
