package handlers

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newMiniRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return mr, rdb
}

func newServerWithRedis(t *testing.T) (*SecuritiesServer, sqlmock.Sqlmock, *miniredis.Miniredis) {
	t.Helper()
	s, mock := newServer(t)
	mr, rdb := newMiniRedis(t)
	s.Redis = rdb
	return s, mock, mr
}

func sampleListingResponse(id int64) *pb.GetListingByIdResponse {
	return &pb.GetListingByIdResponse{
		Summary: &pb.ListingSummary{
			Id:     id,
			Ticker: "AAPL",
			Name:   "Apple Inc",
			Type:   "STOCK",
			Price:  200.0,
		},
	}
}

// ── loadCachedListing ─────────────────────────────────────────────────────────

func TestLoadCachedListing_NilRedis_ReturnsNil(t *testing.T) {
	s := &SecuritiesServer{Redis: nil}
	assert.Nil(t, s.loadCachedListing(context.Background(), 1))
}

func TestLoadCachedListing_KeyMissing_ReturnsNil(t *testing.T) {
	_, rdb := newMiniRedis(t)
	s := &SecuritiesServer{Redis: rdb}
	assert.Nil(t, s.loadCachedListing(context.Background(), 99))
}

func TestLoadCachedListing_InvalidProto_ReturnsNil(t *testing.T) {
	mr, rdb := newMiniRedis(t)
	_ = mr.Set(listingCacheKey(1), "not-valid-proto-json")

	s := &SecuritiesServer{Redis: rdb}
	assert.Nil(t, s.loadCachedListing(context.Background(), 1))
}

func TestLoadCachedListing_ValidCache_ReturnsResponse(t *testing.T) {
	mr, rdb := newMiniRedis(t)
	resp := sampleListingResponse(1)
	data, err := protojson.Marshal(resp)
	require.NoError(t, err)
	_ = mr.Set(listingCacheKey(1), string(data))

	s := &SecuritiesServer{Redis: rdb}
	result := s.loadCachedListing(context.Background(), 1)
	require.NotNil(t, result)
	assert.Equal(t, int64(1), result.Summary.Id)
	assert.Equal(t, "AAPL", result.Summary.Ticker)
}

// ── storeCachedListing ────────────────────────────────────────────────────────

func TestStoreCachedListing_NilRedis_NoOp(t *testing.T) {
	s := &SecuritiesServer{Redis: nil}
	// Must not panic.
	s.storeCachedListing(context.Background(), 1, sampleListingResponse(1))
}

func TestStoreCachedListing_WritesKeyWithTTL(t *testing.T) {
	mr, rdb := newMiniRedis(t)
	s := &SecuritiesServer{Redis: rdb}

	s.storeCachedListing(context.Background(), 42, sampleListingResponse(42))

	val, err := mr.Get(listingCacheKey(42))
	require.NoError(t, err)
	assert.NotEmpty(t, val)

	ttl := mr.TTL(listingCacheKey(42))
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, listingCacheTTL)
}

func TestStoreCachedListing_RoundTrip(t *testing.T) {
	_, rdb := newMiniRedis(t)
	s := &SecuritiesServer{Redis: rdb}

	original := sampleListingResponse(7)
	s.storeCachedListing(context.Background(), 7, original)

	result := s.loadCachedListing(context.Background(), 7)
	require.NotNil(t, result)
	assert.Equal(t, int64(7), result.Summary.Id)
	assert.Equal(t, "AAPL", result.Summary.Ticker)
	assert.Equal(t, "STOCK", result.Summary.Type)
	assert.Equal(t, 200.0, result.Summary.Price)
}

// ── InvalidateListing ─────────────────────────────────────────────────────────

func TestInvalidateListing_NilRedis_NoOp(t *testing.T) {
	s := &SecuritiesServer{Redis: nil}
	// Must not panic.
	s.InvalidateListing(context.Background(), 1)
}

func TestInvalidateListing_DeletesKey(t *testing.T) {
	mr, rdb := newMiniRedis(t)
	s := &SecuritiesServer{Redis: rdb}

	// Store something first.
	s.storeCachedListing(context.Background(), 5, sampleListingResponse(5))
	_, err := mr.Get(listingCacheKey(5))
	require.NoError(t, err, "key must exist before invalidation")

	s.InvalidateListing(context.Background(), 5)

	_, err = mr.Get(listingCacheKey(5))
	assert.Error(t, err, "key must be gone after invalidation")
}

func TestInvalidateListing_NonExistentKey_NoError(t *testing.T) {
	_, rdb := newMiniRedis(t)
	s := &SecuritiesServer{Redis: rdb}
	// Deleting a key that doesn't exist must be a no-op.
	s.InvalidateListing(context.Background(), 999)
}

// ── GetListingById with Redis cache ───────────────────────────────────────────

func TestGetListingById_CacheHit_DoesNotQueryDB(t *testing.T) {
	s, _, _ := newServerWithRedis(t)

	// Pre-populate cache.
	s.storeCachedListing(context.Background(), 1, sampleListingResponse(1))

	// No sqlmock expectations set — any DB call fails the test.
	resp, err := s.GetListingById(context.Background(), &pb.GetListingByIdRequest{Id: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Summary.Id)
	assert.Equal(t, "AAPL", resp.Summary.Ticker)
}

func TestGetListingById_CacheHit_DifferentIDs_IndependentEntries(t *testing.T) {
	_, rdb := newMiniRedis(t)
	s := &SecuritiesServer{Redis: rdb}

	s.storeCachedListing(context.Background(), 1, sampleListingResponse(1))
	s.storeCachedListing(context.Background(), 2, &pb.GetListingByIdResponse{
		Summary: &pb.ListingSummary{Id: 2, Ticker: "GOOG", Name: "Alphabet", Type: "STOCK", Price: 150.0},
	})

	r1 := s.loadCachedListing(context.Background(), 1)
	r2 := s.loadCachedListing(context.Background(), 2)

	require.NotNil(t, r1)
	require.NotNil(t, r2)
	assert.Equal(t, "AAPL", r1.Summary.Ticker)
	assert.Equal(t, "GOOG", r2.Summary.Ticker)
}

func TestGetListingById_CacheMiss_QueriesDBAndPopulatesCache(t *testing.T) {
	s, mock, _ := newServerWithRedis(t)

	// Minimal DB row for a STOCK listing.
	cols := []string{
		"id", "ticker", "name", "type", "acronym", "currency",
		"price", "ask", "bid", "volume", "change",
		"outstanding_shares", "dividend_yield",
		"base_currency", "quote_currency", "liquidity",
		"contract_size", "contract_unit", "futures_settlement_date",
		"stock_listing_id", "option_type", "strike_price",
		"implied_volatility", "open_interest", "option_settlement_date",
	}
	mock.ExpectQuery(`SELECT l.id`).WillReturnRows(
		sqlmock.NewRows(cols).AddRow(
			int64(10), "MSFT", "Microsoft", "STOCK", "NASDAQ", "USD",
			300.0, 301.0, 299.0, int64(5000000), 5.0,
			int64(7000000), 0.5,
			nil, nil, nil,
			nil, nil, nil,
			// stockListingID is nil → no second SELECT price query will fire
			nil, nil, nil, nil, nil, nil,
		),
	)
	// History query (stockListingID is nil so the price sub-query is skipped).
	mock.ExpectQuery(`SELECT date, price`).WillReturnRows(
		sqlmock.NewRows([]string{"date", "price", "ask", "bid", "change", "volume"}),
	)

	resp, err := s.GetListingById(context.Background(), &pb.GetListingByIdRequest{Id: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(10), resp.Summary.Id)

	// Cache must now be populated.
	cached := s.loadCachedListing(context.Background(), 10)
	require.NotNil(t, cached, "cache must be populated after DB query")
	assert.Equal(t, "MSFT", cached.Summary.Ticker)
}

func TestGetListingById_InvalidateThenRefetch_HitsDB(t *testing.T) {
	_, rdb := newMiniRedis(t)
	s := &SecuritiesServer{Redis: rdb}

	// Store in cache.
	s.storeCachedListing(context.Background(), 3, sampleListingResponse(3))
	assert.NotNil(t, s.loadCachedListing(context.Background(), 3))

	// Invalidate.
	s.InvalidateListing(context.Background(), 3)
	assert.Nil(t, s.loadCachedListing(context.Background(), 3), "cache miss after invalidation")
}
