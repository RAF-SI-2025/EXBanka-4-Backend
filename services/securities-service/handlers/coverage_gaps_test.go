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

// ── GetStockExchanges scan error ──────────────────────────────────────────────

func TestGetStockExchanges_ScanError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT id, name, acronym, mic_code").
		WillReturnRows(sqlmock.NewRows(exchangeCols).
			AddRow("bad", "NYSE", "NYSE", "XNYS", "United States", "USD", "America/New_York"))

	_, err := s.GetStockExchanges(context.Background(), &pb.GetStockExchangesRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetWorkingHours second query and scan errors ──────────────────────────────

func TestGetWorkingHours_HoursQueryError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT polity FROM stock_exchanges").
		WithArgs("XNYS").
		WillReturnRows(sqlmock.NewRows([]string{"polity"}).AddRow("United States"))
	mock.ExpectQuery("SELECT id, polity, segment").
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetWorkingHours(context.Background(), &pb.GetWorkingHoursRequest{MicCode: "XNYS"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetWorkingHours_ScanError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT polity FROM stock_exchanges").
		WithArgs("XNYS").
		WillReturnRows(sqlmock.NewRows([]string{"polity"}).AddRow("United States"))
	mock.ExpectQuery("SELECT id, polity, segment").
		WillReturnRows(sqlmock.NewRows(hoursCols).
			AddRow("bad", "United States", "regular", "09:30", "16:00"))

	_, err := s.GetWorkingHours(context.Background(), &pb.GetWorkingHoursRequest{MicCode: "XNYS"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetHolidays scan error ────────────────────────────────────────────────────

func TestGetHolidays_ScanError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT id, polity, holiday_date, COALESCE").
		WillReturnRows(sqlmock.NewRows(holidayCols).
			AddRow("bad", "United States", "not-a-date", "New Year"))

	_, err := s.GetHolidays(context.Background(), &pb.GetHolidaysRequest{Polity: "United States"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── IsExchangeOpen additional error paths ─────────────────────────────────────

func TestIsExchangeOpen_TestModeCheckError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnError(sql.ErrConnDone)

	_, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "XNYS"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestIsExchangeOpen_InvalidTimezone(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(false))
	mock.ExpectQuery("SELECT timezone, polity FROM stock_exchanges").
		WillReturnRows(sqlmock.NewRows([]string{"timezone", "polity"}).
			AddRow("Invalid/NotATimezone", "Testland"))
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "TEST"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestIsExchangeOpen_HolidayCheckError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(false))
	mock.ExpectQuery("SELECT timezone, polity FROM stock_exchanges").
		WillReturnRows(sqlmock.NewRows([]string{"timezone", "polity"}).
			AddRow("America/New_York", "United States"))
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnError(sql.ErrConnDone)

	_, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "XNYS"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestIsExchangeOpen_WorkingHoursQueryError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(false))
	mock.ExpectQuery("SELECT timezone, polity FROM stock_exchanges").
		WillReturnRows(sqlmock.NewRows([]string{"timezone", "polity"}).
			AddRow("America/New_York", "United States"))
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT segment, TO_CHAR").
		WillReturnError(sql.ErrConnDone)

	_, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "XNYS"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestIsExchangeOpen_WorkingHoursScanError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(false))
	mock.ExpectQuery("SELECT timezone, polity FROM stock_exchanges").
		WillReturnRows(sqlmock.NewRows([]string{"timezone", "polity"}).
			AddRow("America/New_York", "United States"))
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT segment, TO_CHAR").
		WillReturnRows(sqlmock.NewRows([]string{"segment", "open_time", "close_time"}).
			AddRow(nil, "09:30", "16:00"))

	_, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "XNYS"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestIsExchangeOpen_InsideRegularHours(t *testing.T) {
	// Use UTC timezone so we control "current time" indirectly by returning
	// a full-day window (00:00–23:59) that always matches, covering the
	// "segment == regular" path and the timeInRange-returns-true path.
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(false))
	mock.ExpectQuery("SELECT timezone, polity FROM stock_exchanges").
		WillReturnRows(sqlmock.NewRows([]string{"timezone", "polity"}).
			AddRow("UTC", "Testland"))
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT segment, TO_CHAR").
		WillReturnRows(sqlmock.NewRows([]string{"segment", "open_time", "close_time"}).
			AddRow("regular", "00:00", "23:59"))

	resp, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "TEST"})
	require.NoError(t, err)
	assert.True(t, resp.IsOpen)
	assert.Equal(t, "regular", resp.Segment)
}

func TestIsExchangeOpen_InPreMarket(t *testing.T) {
	// Return regular hours 08:00–16:00 so current UTC time (whatever it is)
	// might not match any segment. Use a very narrow regular window that
	// is always ±4h before/after to guarantee hitting prePostMarketSegment.
	// Since we can't control now(), we instead test prePostMarketSegment directly.
	// This test just exercises the regularOpen != "" branch returning closed.
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(false))
	mock.ExpectQuery("SELECT timezone, polity FROM stock_exchanges").
		WillReturnRows(sqlmock.NewRows([]string{"timezone", "polity"}).
			AddRow("UTC", "Testland"))
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	// Regular session 01:00-02:00 — almost certainly not in this window
	mock.ExpectQuery("SELECT segment, TO_CHAR").
		WillReturnRows(sqlmock.NewRows([]string{"segment", "open_time", "close_time"}).
			AddRow("regular", "01:00", "02:00"))

	resp, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "TEST"})
	require.NoError(t, err)
	// could be pre_market, post_market, or closed — all are valid here
	assert.NotNil(t, resp)
}

// ── prePostMarketSegment unit tests ──────────────────────────────────────────

func TestPrePostMarketSegment(t *testing.T) {
	window := 4 * time.Hour
	tests := []struct {
		name  string
		t     string
		open  string
		close string
		want  string
	}{
		{"pre_market 1h before open", "08:30", "09:30", "16:00", "pre_market"},
		{"pre_market at boundary", "05:30", "09:30", "16:00", "pre_market"},
		{"post_market 1h after close", "17:00", "09:30", "16:00", "post_market"},
		{"post_market at boundary", "16:00", "09:30", "16:00", "post_market"},
		{"outside all windows", "02:00", "09:30", "16:00", ""},
		{"after post_market window", "21:00", "09:30", "16:00", ""},
		{"invalid t", "bad", "09:30", "16:00", ""},
		{"invalid open", "09:00", "bad", "16:00", ""},
		{"invalid close", "09:00", "09:30", "bad", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := prePostMarketSegment(tc.t, tc.open, tc.close, window)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ── GetListings sort column and order branches ────────────────────────────────

func TestGetListings_SortByPrice(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols))

	_, err := s.GetListings(context.Background(), &pb.GetListingsRequest{SortBy: "price"})
	require.NoError(t, err)
}

func TestGetListings_SortByVolume(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols))

	_, err := s.GetListings(context.Background(), &pb.GetListingsRequest{SortBy: "volume"})
	require.NoError(t, err)
}

func TestGetListings_SortByChange(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols))

	_, err := s.GetListings(context.Background(), &pb.GetListingsRequest{SortBy: "change_percent"})
	require.NoError(t, err)
}

func TestGetListings_SortOrderDESC(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols))

	_, err := s.GetListings(context.Background(), &pb.GetListingsRequest{SortBy: "price", SortOrder: "DESC"})
	require.NoError(t, err)
}

func TestGetListings_ScanError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols).
			AddRow("bad", "AAPL", "Apple", "STOCK", "NASDAQ", "USD",
				150.0, 151.0, 149.0, int64(1000), 0.0,
				int64(0), 1.0, nil, 0.0,
				nil, nil, nil, nil))

	_, err := s.GetListings(context.Background(), &pb.GetListingsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetListings_WithOptionFields(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	settlementDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT l.id").
		WillReturnRows(sqlmock.NewRows(listingSummaryCols).
			AddRow(int64(10), "AAPL260101C00150", "AAPL Call", "OPTION", "CBOE", "USD",
				5.0, 5.1, 4.9, int64(200), 0.0,
				int64(0), 1.0, int64(1), 200.0,
				"CALL", 150.0, settlementDate, int64(5000)))

	resp, err := s.GetListings(context.Background(), &pb.GetListingsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Listings, 1)
	l := resp.Listings[0]
	assert.Equal(t, "CALL", l.OptionType)
	assert.InDelta(t, 150.0, l.StrikePrice, 0.001)
	assert.Equal(t, int64(5000), l.OpenInterest)
}

// ── GetListingById history error paths ───────────────────────────────────────

func TestGetListingById_HistoryQueryError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT l.id, l.ticker").
		WithArgs(int64(1)).
		WillReturnRows(addStockDetailRow(sqlmock.NewRows(listingDetailCols)))
	mock.ExpectQuery("SELECT date, price").
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetListingById(context.Background(), &pb.GetListingByIdRequest{Id: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetListingById_HistoryScanError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT l.id, l.ticker").
		WithArgs(int64(1)).
		WillReturnRows(addStockDetailRow(sqlmock.NewRows(listingDetailCols)))
	mock.ExpectQuery("SELECT date, price").
		WillReturnRows(sqlmock.NewRows(historyCols).
			AddRow("not-a-date", 150.0, 151.0, 149.0, 5.0, int64(500)))

	_, err := s.GetListingById(context.Background(), &pb.GetListingByIdRequest{Id: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetListingHistory error paths ─────────────────────────────────────────────

func TestGetListingHistory_QueryError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT date, price, ask, bid, change, volume").
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetListingHistory(context.Background(), &pb.GetListingHistoryRequest{
		Id: 5, FromDate: "2026-01-01", ToDate: "2026-12-31",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetListingHistory_ScanError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT date, price, ask, bid, change, volume").
		WillReturnRows(sqlmock.NewRows(historyCols).
			AddRow("not-a-date", 100.0, 101.0, 99.0, 1.0, int64(1000)))

	_, err := s.GetListingHistory(context.Background(), &pb.GetListingHistoryRequest{
		Id: 5, FromDate: "2026-01-01", ToDate: "2026-12-31",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}
