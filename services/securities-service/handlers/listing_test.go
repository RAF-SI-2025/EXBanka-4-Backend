package handlers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Column sets for row builders
var listingSummaryCols = []string{
	"id", "ticker", "name", "type", "acronym", "currency",
	"price", "ask", "bid", "volume", "change",
	"outstanding_shares", "contract_size", "stock_listing_id", "stock_price",
	"option_type", "strike_price", "settlement_date", "open_interest",
}

var listingDetailCols = []string{
	"id", "ticker", "name", "type", "acronym", "currency",
	"price", "ask", "bid", "volume", "change",
	"outstanding_shares", "dividend_yield",
	"base_currency", "quote_currency", "liquidity",
	"contract_size", "contract_unit", "futures_settlement_date",
	"stock_listing_id", "option_type", "strike_price",
	"implied_volatility", "open_interest", "option_settlement_date",
}

var historyCols = []string{"date", "price", "ask", "bid", "change", "volume"}

// ── GetListings ────────────────────────────────────────────────────────────────

func TestGetListings_Empty(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols))

	resp, err := s.GetListings(context.Background(), &pb.GetListingsRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Listings)
	assert.Equal(t, int64(0), resp.TotalElements)
	assert.Equal(t, int32(0), resp.TotalPages)
}

func TestGetListings_StockDerivedFields(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols).
			// price=200, change=10 → prev=190, changePercent=5.263...
			// maintenanceMargin(STOCK) = 0.5*200 = 100
			// nominalValue(STOCK) = 200
			AddRow(1, "AAPL", "Apple Inc", "STOCK", "NASDAQ", "USD",
				200.0, 202.0, 198.0, int64(1000000), 10.0,
				int64(5000000), 1.0, nil, 0.0,
				nil, nil, nil, nil))

	resp, err := s.GetListings(context.Background(), &pb.GetListingsRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, resp.Listings, 1)

	l := resp.Listings[0]
	assert.Equal(t, int64(1), l.Id)
	assert.Equal(t, "AAPL", l.Ticker)
	assert.Equal(t, "STOCK", l.Type)
	assert.Equal(t, "NASDAQ", l.ExchangeAcronym)
	assert.InDelta(t, 100.0, l.MaintenanceMargin, 0.001)
	assert.InDelta(t, 110.0, l.InitialMarginCost, 0.001)
	assert.InDelta(t, 200.0, l.NominalValue, 0.001)
	// changePercent = (100 * 10) / (200 - 10) = 1000/190 ≈ 5.263
	assert.InDelta(t, 5.263, l.ChangePercent, 0.01)
}

func TestGetListings_ForexDerivedFields(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols).
			// maintenanceMargin(FOREX_PAIR) = 1000 * 1.1 * 0.10 = 110
			// nominalValue(FOREX_PAIR) = 1000 * 1.1 = 1100
			AddRow(2, "EUR/USD", "Euro / US Dollar", "FOREX_PAIR", "FOREX", "USD",
				1.1, 1.101, 1.099, int64(0), 0.0,
				int64(0), 1.0, nil, 0.0,
				nil, nil, nil, nil))

	resp, err := s.GetListings(context.Background(), &pb.GetListingsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Listings, 1)
	l := resp.Listings[0]
	assert.InDelta(t, 110.0, l.MaintenanceMargin, 0.001)
	assert.InDelta(t, 1100.0, l.NominalValue, 0.001)
}

func TestGetListings_FuturesDerivedFields(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols).
			// contractSize=1000, price=80 → maintenanceMargin = 1000*80*0.10 = 8000
			AddRow(3, "CLJ25", "Crude Oil Jul 2025", "FUTURES_CONTRACT", "NYMEX", "USD",
				80.0, 80.5, 79.5, int64(50000), 0.0,
				int64(0), 1000.0, nil, 0.0,
				nil, nil, nil, nil))

	resp, err := s.GetListings(context.Background(), &pb.GetListingsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Listings, 1)
	l := resp.Listings[0]
	assert.InDelta(t, 8000.0, l.MaintenanceMargin, 0.001)
	assert.InDelta(t, 80000.0, l.NominalValue, 0.001)
}

func TestGetListings_OptionDerivedFields(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols).
			// stock_listing_id=1, stockPrice=200 → maintenanceMargin = 100*0.5*200 = 10000
			AddRow(4, "AAPL240101C00150000", "AAPL Call", "OPTION", "CBOE", "USD",
				5.0, 5.1, 4.9, int64(200), 0.0,
				int64(0), 1.0, int64(1), 200.0,
				nil, nil, nil, nil))

	resp, err := s.GetListings(context.Background(), &pb.GetListingsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Listings, 1)
	l := resp.Listings[0]
	assert.InDelta(t, 10000.0, l.MaintenanceMargin, 0.001)
	assert.InDelta(t, 500.0, l.NominalValue, 0.001) // 100 * 5.0
}

func TestGetListings_DefaultPagination(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols))

	// Zero values should default to page=1, pageSize=20
	resp, err := s.GetListings(context.Background(), &pb.GetListingsRequest{Page: 0, PageSize: 0})
	require.NoError(t, err)
	assert.Empty(t, resp.Listings)
}

func TestGetListings_Pagination(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols))

	resp, err := s.GetListings(context.Background(), &pb.GetListingsRequest{Page: 2, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(25), resp.TotalElements)
	assert.Equal(t, int32(3), resp.TotalPages) // ceil(25/10) = 3
}

func TestGetListings_CountDBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").WillReturnError(sql.ErrConnDone)

	_, err := s.GetListings(context.Background(), &pb.GetListingsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetListings_QueryDBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT l.id").WillReturnError(sql.ErrConnDone)

	_, err := s.GetListings(context.Background(), &pb.GetListingsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetListingById ─────────────────────────────────────────────────────────────

func addStockDetailRow(rows *sqlmock.Rows) *sqlmock.Rows {
	return rows.AddRow(
		int64(1), "AAPL", "Apple Inc", "STOCK", "NASDAQ", "USD",
		150.0, 151.0, 149.0, int64(500000), 5.0,
		// stock
		int64(1000000), 0.02,
		// forex (nil)
		nil, nil, nil,
		// futures (nil)
		nil, nil, nil,
		// option (nil)
		nil, nil, nil, nil, nil, nil,
	)
}

func TestGetListingById_Stock(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT l.id, l.ticker").
		WithArgs(int64(1)).
		WillReturnRows(addStockDetailRow(sqlmock.NewRows(listingDetailCols)))
	// history query
	mock.ExpectQuery("SELECT date, price").
		WillReturnRows(sqlmock.NewRows(historyCols).
			AddRow(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), 150.0, 151.0, 149.0, 5.0, int64(500000)).
			AddRow(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 145.0, 146.0, 144.0, 3.0, int64(400000)))

	resp, err := s.GetListingById(context.Background(), &pb.GetListingByIdRequest{Id: 1})
	require.NoError(t, err)
	require.NotNil(t, resp.Summary)
	assert.Equal(t, "AAPL", resp.Summary.Ticker)
	assert.Equal(t, "STOCK", resp.Summary.Type)
	assert.InDelta(t, 75.0, resp.Summary.MaintenanceMargin, 0.001) // 0.5*150

	require.Len(t, resp.PriceHistory, 2)
	assert.Equal(t, "2025-01-02", resp.PriceHistory[0].Date)
	assert.Equal(t, "2025-01-01", resp.PriceHistory[1].Date)

	stockDetail, ok := resp.Detail.(*pb.GetListingByIdResponse_Stock)
	require.True(t, ok, "expected Stock detail oneof")
	assert.Equal(t, int64(1000000), stockDetail.Stock.OutstandingShares)
	assert.InDelta(t, 0.02, stockDetail.Stock.DividendYield, 0.0001)
	assert.InDelta(t, 150_000_000.0, stockDetail.Stock.MarketCap, 0.001) // 1M * 150
}

func TestGetListingById_Forex(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT l.id, l.ticker").
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows(listingDetailCols).AddRow(
			int64(2), "EUR/USD", "Euro / US Dollar", "FOREX_PAIR", "FOREX", "USD",
			1.1, 1.101, 1.099, int64(0), 0.0,
			// stock (nil)
			nil, nil,
			// forex
			"EUR", "USD", "HIGH",
			// futures (nil)
			nil, nil, nil,
			// option (nil)
			nil, nil, nil, nil, nil, nil,
		))
	mock.ExpectQuery("SELECT date, price").
		WillReturnRows(sqlmock.NewRows(historyCols))

	resp, err := s.GetListingById(context.Background(), &pb.GetListingByIdRequest{Id: 2})
	require.NoError(t, err)
	assert.Equal(t, "EUR/USD", resp.Summary.Ticker)

	forexDetail, ok := resp.Detail.(*pb.GetListingByIdResponse_Forex)
	require.True(t, ok, "expected Forex detail oneof")
	assert.Equal(t, "EUR", forexDetail.Forex.BaseCurrency)
	assert.Equal(t, "USD", forexDetail.Forex.QuoteCurrency)
	assert.Equal(t, "HIGH", forexDetail.Forex.Liquidity)
	assert.InDelta(t, 1100.0, forexDetail.Forex.NominalValue, 0.001) // 1000 * 1.1
}

func TestGetListingById_Futures(t *testing.T) {
	s, mock := newServer(t)
	settleDate := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT l.id, l.ticker").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows(listingDetailCols).AddRow(
			int64(3), "CLJ25", "Crude Oil", "FUTURES_CONTRACT", "NYMEX", "USD",
			80.0, 80.5, 79.5, int64(50000), 0.0,
			// stock (nil)
			nil, nil,
			// forex (nil)
			nil, nil, nil,
			// futures
			1000.0, "Barrel", settleDate,
			// option (nil)
			nil, nil, nil, nil, nil, nil,
		))
	mock.ExpectQuery("SELECT date, price").
		WillReturnRows(sqlmock.NewRows(historyCols))

	resp, err := s.GetListingById(context.Background(), &pb.GetListingByIdRequest{Id: 3})
	require.NoError(t, err)

	futuresDetail, ok := resp.Detail.(*pb.GetListingByIdResponse_Futures)
	require.True(t, ok, "expected Futures detail oneof")
	assert.InDelta(t, 1000.0, futuresDetail.Futures.ContractSize, 0.001)
	assert.Equal(t, "Barrel", futuresDetail.Futures.ContractUnit)
	assert.Equal(t, "2025-06-30", futuresDetail.Futures.SettlementDate)
}

func TestGetListingById_Option(t *testing.T) {
	s, mock := newServer(t)
	settleDate := time.Date(2025, 3, 21, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT l.id, l.ticker").
		WithArgs(int64(4)).
		WillReturnRows(sqlmock.NewRows(listingDetailCols).AddRow(
			int64(4), "AAPL250321C00150000", "AAPL Call 150", "OPTION", "CBOE", "USD",
			5.0, 5.2, 4.8, int64(200), 0.0,
			// stock (nil)
			nil, nil,
			// forex (nil)
			nil, nil, nil,
			// futures (nil)
			nil, nil, nil,
			// option
			int64(1), "CALL", 150.0, 0.25, int64(5000), settleDate,
		))
	// Underlying stock price lookup
	mock.ExpectQuery("SELECT price FROM listing WHERE id").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(200.0))
	mock.ExpectQuery("SELECT date, price").
		WillReturnRows(sqlmock.NewRows(historyCols))

	resp, err := s.GetListingById(context.Background(), &pb.GetListingByIdRequest{Id: 4})
	require.NoError(t, err)

	// maintenanceMargin = 100 * 0.5 * 200 = 10000
	assert.InDelta(t, 10000.0, resp.Summary.MaintenanceMargin, 0.001)

	optDetail, ok := resp.Detail.(*pb.GetListingByIdResponse_Option)
	require.True(t, ok, "expected Option detail oneof")
	assert.Equal(t, int64(1), optDetail.Option.StockListingId)
	assert.Equal(t, "CALL", optDetail.Option.OptionType)
	assert.InDelta(t, 150.0, optDetail.Option.StrikePrice, 0.001)
	assert.InDelta(t, 0.25, optDetail.Option.ImpliedVolatility, 0.0001)
	assert.Equal(t, int64(5000), optDetail.Option.OpenInterest)
	assert.Equal(t, "2025-03-21", optDetail.Option.SettlementDate)
}

func TestGetListingById_NotFound(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT l.id, l.ticker").
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	_, err := s.GetListingById(context.Background(), &pb.GetListingByIdRequest{Id: 999})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetListingById_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT l.id, l.ticker").
		WithArgs(int64(1)).
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetListingById(context.Background(), &pb.GetListingByIdRequest{Id: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetListingById_EmptyHistory(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT l.id, l.ticker").
		WithArgs(int64(1)).
		WillReturnRows(addStockDetailRow(sqlmock.NewRows(listingDetailCols)))
	mock.ExpectQuery("SELECT date, price").
		WillReturnRows(sqlmock.NewRows(historyCols)) // no rows

	resp, err := s.GetListingById(context.Background(), &pb.GetListingByIdRequest{Id: 1})
	require.NoError(t, err)
	assert.Empty(t, resp.PriceHistory)
}

// ── GetListingHistory ──────────────────────────────────────────────────────────

func TestGetListingHistory_Found(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT date, price").
		WillReturnRows(sqlmock.NewRows(historyCols).
			AddRow(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 100.0, 101.0, 99.0, 2.0, int64(300000)).
			AddRow(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), 102.0, 103.0, 101.0, 2.0, int64(350000)).
			AddRow(time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), 104.0, 105.0, 103.0, 2.0, int64(400000)))

	resp, err := s.GetListingHistory(context.Background(), &pb.GetListingHistoryRequest{
		Id: 1, FromDate: "2025-01-01", ToDate: "2025-01-03",
	})
	require.NoError(t, err)
	require.Len(t, resp.History, 3)
	assert.Equal(t, "2025-01-01", resp.History[0].Date)
	assert.Equal(t, "2025-01-03", resp.History[2].Date)
	assert.InDelta(t, 100.0, resp.History[0].Price, 0.001)
}

func TestGetListingHistory_NotFound(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err := s.GetListingHistory(context.Background(), &pb.GetListingHistoryRequest{
		Id: 999, FromDate: "2025-01-01", ToDate: "2025-01-31",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetListingHistory_ExistsDBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(1)).
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetListingHistory(context.Background(), &pb.GetListingHistoryRequest{
		Id: 1, FromDate: "2025-01-01", ToDate: "2025-01-31",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetListingHistory_EmptyRange(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT date, price").
		WillReturnRows(sqlmock.NewRows(historyCols)) // no rows in range

	resp, err := s.GetListingHistory(context.Background(), &pb.GetListingHistoryRequest{
		Id: 1, FromDate: "2020-01-01", ToDate: "2020-01-31",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.History)
}

// ── Helper function unit tests ─────────────────────────────────────────────────

func TestComputeMaintenanceMargin(t *testing.T) {
	tests := []struct {
		name         string
		lType        string
		price        float64
		outshares    int64
		contractSize float64
		stockPrice   float64
		want         float64
	}{
		{"STOCK 50% of price", "STOCK", 200.0, 1000, 1.0, 0, 100.0},
		{"STOCK zero price", "STOCK", 0.0, 0, 1.0, 0, 0.0},
		{"FOREX_PAIR 10% of 1000*price", "FOREX_PAIR", 1.1, 0, 1.0, 0, 110.0},
		{"FUTURES contractSize*price*10%", "FUTURES_CONTRACT", 80.0, 0, 1000.0, 0, 8000.0},
		{"OPTION 100*50%*stockPrice", "OPTION", 5.0, 0, 1.0, 200.0, 10000.0},
		{"unknown type returns 0", "UNKNOWN", 100.0, 0, 1.0, 0, 0.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeMaintenanceMargin(tc.lType, tc.price, tc.outshares, tc.contractSize, tc.stockPrice)
			assert.InDelta(t, tc.want, got, 0.001)
		})
	}
}

func TestListingChangePercent(t *testing.T) {
	tests := []struct {
		name   string
		price  float64
		change float64
		want   float64
	}{
		{"positive change", 200.0, 10.0, 5.263}, // 100*10/190
		{"negative change", 95.0, -5.0, -5.0},   // 100*(-5)/100
		{"zero change", 100.0, 0.0, 0.0},
		{"zero prev price guard", 5.0, 5.0, 0.0}, // prev = 0 → guard
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := listingChangePercent(tc.price, tc.change)
			assert.InDelta(t, tc.want, got, 0.01)
		})
	}
}

func TestListingNominalValue(t *testing.T) {
	tests := []struct {
		lType        string
		price        float64
		contractSize float64
		want         float64
	}{
		{"STOCK", 150.0, 1.0, 150.0},
		{"FOREX_PAIR", 1.1, 1.0, 1100.0},
		{"FUTURES_CONTRACT", 80.0, 1000.0, 80000.0},
		{"OPTION", 5.0, 1.0, 500.0},
		{"unknown", 100.0, 1.0, 100.0},
	}
	for _, tc := range tests {
		t.Run(tc.lType, func(t *testing.T) {
			got := listingNominalValue(tc.lType, tc.price, tc.contractSize)
			assert.InDelta(t, tc.want, got, 0.001)
		})
	}
}
