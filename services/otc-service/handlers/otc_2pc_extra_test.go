package handlers

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ── PrepareOtcInterbank: seller-side accept (idemRouting != partnerRouting) ──

func TestPrepareOtcInterbank_SellerAccept_NegotiationNotFound(t *testing.T) {
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	// idemRouting="888" != partnerRouting="444" → falls through to general load
	mOTC.ExpectQuery("SELECT status, ticker, seller_id, seller_type, amount").
		WillReturnRows(sqlmock.NewRows([]string{"status", "ticker", "seller_id", "seller_type", "amount"}))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(99, true, "888"))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
}

func TestPrepareOtcInterbank_SellerAccept_NotAccepted(t *testing.T) {
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	mOTC.ExpectQuery("SELECT status, ticker, seller_id, seller_type, amount").
		WillReturnRows(sqlmock.NewRows([]string{"status", "ticker", "seller_id", "seller_type", "amount"}).
			AddRow("PENDING_BUYER", "AAPL", int64(1), "CLIENT", int32(10)))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(1, true, "888"))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
	assert.Equal(t, "OPTION_USED_OR_EXPIRED", resp.Reason)
}

func TestPrepareOtcInterbank_SellerAccept_ContractExists(t *testing.T) {
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, mOTC, _, _, _, _, _ := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	mOTC.ExpectQuery("SELECT status, ticker, seller_id, seller_type, amount").
		WillReturnRows(sqlmock.NewRows([]string{"status", "ticker", "seller_id", "seller_type", "amount"}).
			AddRow("ACCEPTED", "AAPL", int64(1), "CLIENT", int32(10)))
	mOTC.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(1, true, "888"))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
}

func TestPrepareOtcInterbank_SellerAccept_InsufficientShares(t *testing.T) {
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, mOTC, _, _, _, mPort, mSec := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	mOTC.ExpectQuery("SELECT status, ticker, seller_id, seller_type, amount").
		WillReturnRows(sqlmock.NewRows([]string{"status", "ticker", "seller_id", "seller_type", "amount"}).
			AddRow("ACCEPTED", "AAPL", int64(1), "CLIENT", int32(10)))
	mOTC.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	// amount == stockAmount so no mismatch
	// listingIDForTicker
	mSec.ExpectQuery("SELECT id FROM listing WHERE ticker").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	// free shares = 0 < 10
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(0)))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(1, true, "888"))
	require.NoError(t, err)
	assert.Equal(t, "NO", resp.Vote)
}

func TestPrepareOtcInterbank_SellerAccept_HappyPath(t *testing.T) {
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	s, mOTC, _, _, _, mPort, mSec := newTestServer(t)
	mOTC.ExpectQuery("SELECT cached_vote FROM otc_interbank_tx").
		WillReturnRows(sqlmock.NewRows([]string{"cached_vote"}))
	mOTC.ExpectQuery("SELECT status, ticker, seller_id, seller_type, amount").
		WillReturnRows(sqlmock.NewRows([]string{"status", "ticker", "seller_id", "seller_type", "amount"}).
			AddRow("ACCEPTED", "AAPL", int64(1), "CLIENT", int32(10)))
	mOTC.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mSec.ExpectQuery("SELECT id FROM listing WHERE ticker").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	// enough free shares
	mPort.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(100)))
	mOTC.ExpectExec("INSERT INTO otc_interbank_tx").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.PrepareOtcInterbank(context.Background(), makePrepareOtcReq(1, true, "888"))
	require.NoError(t, err)
	assert.Equal(t, "YES", resp.Vote)
}

// ── CommitOtcInterbank: seller-side ACCEPT ────────────────────────────────────

func TestCommitOtcInterbank_Accept_SellerSide_HappyPath(t *testing.T) {
	s, mOTC, _, _, _, mPort, mSec := newTestServer(t)
	// tx lookup — ACCEPT type with non-INTERBANK seller
	mOTC.ExpectQuery("SELECT id, status, tx_type, negotiation_id, stock_amount").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "tx_type", "negotiation_id", "stock_amount"}).
			AddRow(int64(1), "PENDING", "ACCEPT", int64(7), int32(10)))
	// load negotiation — sellerType="CLIENT" means seller-side
	mOTC.ExpectQuery("SELECT ticker, currency, settlement_date").
		WillReturnRows(sqlmock.NewRows([]string{
			"ticker", "currency", "settlement_date", "seller_id", "seller_type",
			"amount", "price_per_stock", "premium",
		}).AddRow("AAPL", "RSD", "2099-12-31", int64(5), "CLIENT", int32(10), 100.0, 5.0))
	// listingIDForTicker
	mSec.ExpectQuery("SELECT id FROM listing WHERE ticker").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	// INSERT contract
	mOTC.ExpectExec("INSERT INTO otc_contracts").WillReturnResult(sqlmock.NewResult(1, 1))
	// UPDATE portfolio_entry to reserve shares
	mPort.ExpectExec("UPDATE portfolio_entry").WillReturnResult(sqlmock.NewResult(1, 1))
	// UPDATE tx status
	mOTC.ExpectExec("UPDATE otc_interbank_tx SET status='COMMITTED'").WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.CommitOtcInterbank(context.Background(), makeCommitOtcReq())
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ── InterbankAcceptNegotiation: idempotent (already ACCEPTED) ─────────────────

func TestInterbankAcceptNegotiation_AlreadyAccepted(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)

	addLookupByLocalID(mOTC, 42, "ACCEPTED")
	mOTC.ExpectBegin()
	mOTC.ExpectQuery("SELECT id, status, ticker, currency").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "ticker", "currency", "settlement_date",
			"amount", "premium", "price_per_stock", "seller_id", "seller_type",
		}).AddRow(int64(42), "ACCEPTED", "AAPL", "RSD", "2099-12-31",
			int32(10), 5.0, 100.0, int64(7), "CLIENT"))
	mOTC.ExpectCommit()

	resp, err := s.InterbankAcceptNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "42",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ── ExpireContracts: with rows ────────────────────────────────────────────────

func TestExpireContracts_WithRSDRows(t *testing.T) {
	s, mOTC, _, _, _, mPort, _ := newTestServer(t)
	// UPDATE returns expired contracts — RSD currency so convertAmount is no-op
	mOTC.ExpectQuery("UPDATE otc_contracts SET status='EXPIRED'").
		WillReturnRows(sqlmock.NewRows([]string{"id", "buyer_id", "buyer_type", "premium", "currency"}).
			AddRow(int64(1), int64(5), "CLIENT", 10.0, "RSD"))
	// recordOtcTax → INSERT into tax_record (PortfolioDB)
	mPort.ExpectExec("INSERT INTO tax_record").WillReturnResult(sqlmock.NewResult(1, 1))

	s.ExpireContracts() // should not panic
}

// ── sagaFaultHook: compensate fault ──────────────────────────────────────────

func TestSagaFaultHook_CompensateFault(t *testing.T) {
	t.Setenv("OTC_SAGA_TEST_HOOKS", "true")
	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("x-saga-compensate-fail", "C1"))
	counts := map[string]int{}
	err := sagaFaultHook(ctx, "C1", counts)
	assert.Error(t, err)
}

// ── lookupInterbankNegotiation: fallback path ─────────────────────────────────

func TestLookupInterbankNegotiation_NumericIDNotFound_FallsBack(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	// Primary: id=42 not found
	mOTC.ExpectQuery("SELECT id, status FROM otc_negotiations WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}))
	// Fallback: by creator routing + externalId — also not found
	mOTC.ExpectQuery("SELECT id, status FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}))

	// lookupInterbankNegotiation is internal; exercise it via GetNegotiation
	_, err := s.InterbankGetNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "42",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestLookupInterbankNegotiation_NonNumericID_FallsBack(t *testing.T) {
	s, mOTC, _, _, _, _, _ := newTestServer(t)
	// Non-numeric externalId → skips local id query, goes straight to creator key fallback
	mOTC.ExpectQuery("SELECT id, status FROM otc_negotiations").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}))

	_, err := s.InterbankGetNegotiation(context.Background(), &pb.InterbankNegotiationIdRequest{
		RoutingNumber: 444, ExternalId: "abc-external",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ── convertAmount: different currencies ──────────────────────────────────────

func TestConvertAmount_SameCurrency_NoQuery(t *testing.T) {
	result, err := convertAmount(context.Background(), nil, 100.0, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 100.0, result)
}

func TestConvertAmount_DifferentCurrencies_DBError(t *testing.T) {
	excDB, excMock, _ := sqlmock.New()
	defer excDB.Close()
	excMock.ExpectQuery("SELECT middle_rate FROM daily_exchange_rates").
		WillReturnRows(sqlmock.NewRows([]string{"middle_rate"})) // empty → ErrNoRows
	_, err := convertAmount(context.Background(), excDB, 100.0, 1, 2)
	require.Error(t, err)
}
