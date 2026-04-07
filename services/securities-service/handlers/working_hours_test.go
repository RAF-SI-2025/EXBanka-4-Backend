package handlers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var hoursCols    = []string{"id", "polity", "segment", "open_time", "close_time"}
var holidayCols  = []string{"id", "polity", "holiday_date", "description"}

// ── GetWorkingHours ───────────────────────────────────────────────────────────

func TestGetWorkingHours_Success(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT polity FROM stock_exchanges").
		WithArgs("XNYS").
		WillReturnRows(sqlmock.NewRows([]string{"polity"}).AddRow("United States"))
	mock.ExpectQuery("SELECT id, polity, segment, open_time, close_time").
		WithArgs("United States").
		WillReturnRows(sqlmock.NewRows(hoursCols).
			AddRow(1, "United States", "pre_market", "04:00", "09:30").
			AddRow(2, "United States", "regular", "09:30", "16:00").
			AddRow(3, "United States", "post_market", "16:00", "20:00"))

	resp, err := s.GetWorkingHours(context.Background(), &pb.GetWorkingHoursRequest{MicCode: "XNYS"})
	require.NoError(t, err)
	require.Len(t, resp.Hours, 3)
	assert.Equal(t, "pre_market", resp.Hours[0].Segment)
	assert.Equal(t, "04:00", resp.Hours[0].OpenTime)
	assert.Equal(t, "09:30", resp.Hours[0].CloseTime)
}

func TestGetWorkingHours_ExchangeNotFound(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT polity FROM stock_exchanges").
		WithArgs("XXXX").
		WillReturnError(sql.ErrNoRows)

	_, err := s.GetWorkingHours(context.Background(), &pb.GetWorkingHoursRequest{MicCode: "XXXX"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetWorkingHours_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT polity FROM stock_exchanges").
		WithArgs("XNYS").
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetWorkingHours(context.Background(), &pb.GetWorkingHoursRequest{MicCode: "XNYS"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── SetWorkingHours ───────────────────────────────────────────────────────────

func TestSetWorkingHours_Insert(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("INSERT INTO exchange_working_hours").
		WithArgs("United States", "regular", "09:30", "16:00").
		WillReturnRows(sqlmock.NewRows(hoursCols).
			AddRow(1, "United States", "regular", "09:30", "16:00"))

	resp, err := s.SetWorkingHours(context.Background(), &pb.SetWorkingHoursRequest{
		Polity:    "United States",
		Segment:   "regular",
		OpenTime:  "09:30",
		CloseTime: "16:00",
	})
	require.NoError(t, err)
	assert.Equal(t, "regular", resp.Hours.Segment)
	assert.Equal(t, "09:30", resp.Hours.OpenTime)
}

func TestSetWorkingHours_Upsert(t *testing.T) {
	s, mock := newServer(t)
	// Same polity+segment already exists — ON CONFLICT DO UPDATE returns the updated row
	mock.ExpectQuery("INSERT INTO exchange_working_hours").
		WithArgs("United States", "regular", "09:30", "17:00").
		WillReturnRows(sqlmock.NewRows(hoursCols).
			AddRow(1, "United States", "regular", "09:30", "17:00"))

	resp, err := s.SetWorkingHours(context.Background(), &pb.SetWorkingHoursRequest{
		Polity:    "United States",
		Segment:   "regular",
		OpenTime:  "09:30",
		CloseTime: "17:00",
	})
	require.NoError(t, err)
	assert.Equal(t, "17:00", resp.Hours.CloseTime)
}

func TestSetWorkingHours_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("INSERT INTO exchange_working_hours").
		WillReturnError(sql.ErrConnDone)

	_, err := s.SetWorkingHours(context.Background(), &pb.SetWorkingHoursRequest{
		Polity: "United States", Segment: "regular", OpenTime: "09:30", CloseTime: "16:00",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetHolidays ───────────────────────────────────────────────────────────────

func TestGetHolidays_Success(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT id, polity, holiday_date, COALESCE").
		WithArgs("United States").
		WillReturnRows(sqlmock.NewRows(holidayCols).
			AddRow(1, "United States", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "New Year's Day").
			AddRow(2, "United States", time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC), "Independence Day"))

	resp, err := s.GetHolidays(context.Background(), &pb.GetHolidaysRequest{Polity: "United States"})
	require.NoError(t, err)
	require.Len(t, resp.Holidays, 2)
	assert.Equal(t, "2026-01-01", resp.Holidays[0].HolidayDate)
	assert.Equal(t, "New Year's Day", resp.Holidays[0].Description)
}

func TestGetHolidays_Empty(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT id, polity, holiday_date, COALESCE").
		WithArgs("Narnia").
		WillReturnRows(sqlmock.NewRows(holidayCols))

	resp, err := s.GetHolidays(context.Background(), &pb.GetHolidaysRequest{Polity: "Narnia"})
	require.NoError(t, err)
	assert.Empty(t, resp.Holidays)
}

func TestGetHolidays_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT id, polity, holiday_date, COALESCE").
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetHolidays(context.Background(), &pb.GetHolidaysRequest{Polity: "United States"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── AddHoliday ────────────────────────────────────────────────────────────────

func TestAddHoliday_Success(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("INSERT INTO exchange_holidays").
		WithArgs("United States", "2026-01-01", "New Year's Day").
		WillReturnRows(sqlmock.NewRows(holidayCols).
			AddRow(1, "United States", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "New Year's Day"))

	resp, err := s.AddHoliday(context.Background(), &pb.AddHolidayRequest{
		Polity:      "United States",
		HolidayDate: "2026-01-01",
		Description: "New Year's Day",
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-01-01", resp.Holiday.HolidayDate)
	assert.Equal(t, "New Year's Day", resp.Holiday.Description)
}

func TestAddHoliday_Duplicate(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("INSERT INTO exchange_holidays").
		WillReturnError(&pq.Error{Code: "23505"})

	_, err := s.AddHoliday(context.Background(), &pb.AddHolidayRequest{
		Polity: "United States", HolidayDate: "2026-01-01", Description: "New Year's Day",
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestAddHoliday_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("INSERT INTO exchange_holidays").
		WillReturnError(sql.ErrConnDone)

	_, err := s.AddHoliday(context.Background(), &pb.AddHolidayRequest{
		Polity: "United States", HolidayDate: "2026-01-01",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── DeleteHoliday ─────────────────────────────────────────────────────────────

func TestDeleteHoliday_Success(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectExec("DELETE FROM exchange_holidays").
		WithArgs("United States", "2026-01-01").
		WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := s.DeleteHoliday(context.Background(), &pb.DeleteHolidayRequest{
		Polity: "United States", HolidayDate: "2026-01-01",
	})
	require.NoError(t, err)
}

func TestDeleteHoliday_NotFound(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectExec("DELETE FROM exchange_holidays").
		WithArgs("United States", "2000-01-01").
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := s.DeleteHoliday(context.Background(), &pb.DeleteHolidayRequest{
		Polity: "United States", HolidayDate: "2000-01-01",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestDeleteHoliday_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectExec("DELETE FROM exchange_holidays").
		WillReturnError(sql.ErrConnDone)

	_, err := s.DeleteHoliday(context.Background(), &pb.DeleteHolidayRequest{
		Polity: "United States", HolidayDate: "2026-01-01",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── IsExchangeOpen ────────────────────────────────────────────────────────────

func TestIsExchangeOpen_ExchangeNotFound(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(false))
	mock.ExpectQuery("SELECT timezone, polity FROM stock_exchanges").
		WithArgs("XXXX").
		WillReturnError(sql.ErrNoRows)

	_, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "XXXX"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestIsExchangeOpen_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(false))
	mock.ExpectQuery("SELECT timezone, polity FROM stock_exchanges").
		WithArgs("XNYS").
		WillReturnError(sql.ErrConnDone)

	_, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "XNYS"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestIsExchangeOpen_Holiday(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(false))
	mock.ExpectQuery("SELECT timezone, polity FROM stock_exchanges").
		WithArgs("XNYS").
		WillReturnRows(sqlmock.NewRows([]string{"timezone", "polity"}).
			AddRow("America/New_York", "United States"))
	// Holiday exists → exchange closed
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	resp, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "XNYS"})
	require.NoError(t, err)
	assert.False(t, resp.IsOpen)
	assert.Equal(t, "closed", resp.Segment)
	assert.Equal(t, "XNYS", resp.MicCode)
}

func TestIsExchangeOpen_NoHoursConfigured(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(false))
	mock.ExpectQuery("SELECT timezone, polity FROM stock_exchanges").
		WithArgs("XNYS").
		WillReturnRows(sqlmock.NewRows([]string{"timezone", "polity"}).
			AddRow("America/New_York", "United States"))
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	// No working hours rows → exchange closed
	mock.ExpectQuery("SELECT segment, open_time, close_time FROM exchange_working_hours").
		WithArgs("United States").
		WillReturnRows(sqlmock.NewRows([]string{"segment", "open_time", "close_time"}))

	resp, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "XNYS"})
	require.NoError(t, err)
	assert.False(t, resp.IsOpen)
	assert.Equal(t, "closed", resp.Segment)
}

// ── timeInRange (unit test for the helper) ────────────────────────────────────

func TestTimeInRange(t *testing.T) {
	tests := []struct {
		name  string
		t     string
		open  string
		close string
		want  bool
	}{
		{"inside regular hours", "12:00", "09:30", "16:00", true},
		{"at open boundary", "09:30", "09:30", "16:00", true},
		{"just before close", "15:59", "09:30", "16:00", true},
		{"at close boundary", "16:00", "09:30", "16:00", false},
		{"before open", "08:00", "09:30", "16:00", false},
		{"after close", "20:01", "16:00", "20:00", false},
		{"pre-market open", "05:00", "04:00", "09:30", true},
		{"pre-market at open", "04:00", "04:00", "09:30", true},
		{"midnight outside all", "00:00", "04:00", "20:00", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, timeInRange(tc.t, tc.open, tc.close))
		})
	}
}
