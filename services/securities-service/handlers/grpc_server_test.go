package handlers

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var exchangeCols = []string{"id", "name", "acronym", "mic_code", "polity", "currency", "timezone"}

func newServer(t *testing.T) (*SecuritiesServer, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return &SecuritiesServer{DB: db}, mock
}

// ── Ping ──────────────────────────────────────────────────────────────────────

func TestPing(t *testing.T) {
	s, _ := newServer(t)
	resp, err := s.Ping(context.Background(), &pb.PingRequest{})
	require.NoError(t, err)
	assert.Equal(t, "securities-service ok", resp.Message)
}

// ── GetStockExchanges ─────────────────────────────────────────────────────────

func TestGetStockExchanges_Empty(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT id, name, acronym, mic_code").
		WillReturnRows(sqlmock.NewRows(exchangeCols))

	resp, err := s.GetStockExchanges(context.Background(), &pb.GetStockExchangesRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Exchanges)
}

func TestGetStockExchanges_ReturnsAll(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT id, name, acronym, mic_code").
		WillReturnRows(sqlmock.NewRows(exchangeCols).
			AddRow(1, "New York Stock Exchange", "NYSE", "XNYS", "United States", "USD", "America/New_York").
			AddRow(2, "London Stock Exchange", "LSE", "XLON", "United Kingdom", "GBP", "Europe/London"))

	resp, err := s.GetStockExchanges(context.Background(), &pb.GetStockExchangesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Exchanges, 2)
	assert.Equal(t, "XNYS", resp.Exchanges[0].MicCode)
	assert.Equal(t, "XLON", resp.Exchanges[1].MicCode)
}

func TestGetStockExchanges_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT id, name, acronym, mic_code").
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetStockExchanges(context.Background(), &pb.GetStockExchangesRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetStockExchangeByMIC ─────────────────────────────────────────────────────

func TestGetStockExchangeByMIC_Found(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT id, name, acronym, mic_code").
		WithArgs("XNYS").
		WillReturnRows(sqlmock.NewRows(exchangeCols).
			AddRow(1, "New York Stock Exchange", "NYSE", "XNYS", "United States", "USD", "America/New_York"))

	resp, err := s.GetStockExchangeByMIC(context.Background(), &pb.GetStockExchangeByMICRequest{MicCode: "XNYS"})
	require.NoError(t, err)
	assert.Equal(t, "New York Stock Exchange", resp.Exchange.Name)
	assert.Equal(t, "America/New_York", resp.Exchange.Timezone)
}

func TestGetStockExchangeByMIC_NotFound(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT id, name, acronym, mic_code").
		WithArgs("XXXX").
		WillReturnError(sql.ErrNoRows)

	_, err := s.GetStockExchangeByMIC(context.Background(), &pb.GetStockExchangeByMICRequest{MicCode: "XXXX"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetStockExchangeByMIC_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT id, name, acronym, mic_code").
		WithArgs("XNYS").
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetStockExchangeByMIC(context.Background(), &pb.GetStockExchangeByMICRequest{MicCode: "XNYS"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── CreateStockExchange ───────────────────────────────────────────────────────

func TestCreateStockExchange_Success(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("INSERT INTO stock_exchanges").
		WithArgs("New York Stock Exchange", "NYSE", "XNYS", "United States", "USD", "America/New_York").
		WillReturnRows(sqlmock.NewRows(exchangeCols).
			AddRow(1, "New York Stock Exchange", "NYSE", "XNYS", "United States", "USD", "America/New_York"))

	resp, err := s.CreateStockExchange(context.Background(), &pb.CreateStockExchangeRequest{
		Name:     "New York Stock Exchange",
		Acronym:  "NYSE",
		MicCode:  "XNYS",
		Polity:   "United States",
		Currency: "USD",
		Timezone: "America/New_York",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Exchange.Id)
	assert.Equal(t, "XNYS", resp.Exchange.MicCode)
}

func TestCreateStockExchange_DuplicateMIC(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("INSERT INTO stock_exchanges").
		WillReturnError(&pq.Error{Code: "23505"})

	_, err := s.CreateStockExchange(context.Background(), &pb.CreateStockExchangeRequest{
		Name: "NYSE Copy", Acronym: "NYSE", MicCode: "XNYS",
		Polity: "United States", Currency: "USD", Timezone: "America/New_York",
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestCreateStockExchange_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("INSERT INTO stock_exchanges").
		WillReturnError(sql.ErrConnDone)

	_, err := s.CreateStockExchange(context.Background(), &pb.CreateStockExchangeRequest{
		Name: "NYSE", Acronym: "NYSE", MicCode: "XNYS",
		Polity: "United States", Currency: "USD", Timezone: "America/New_York",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── UpdateStockExchange ───────────────────────────────────────────────────────

func TestUpdateStockExchange_Success(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("UPDATE stock_exchanges").
		WithArgs("NYSE Updated", "NYSE", "United States", "USD", "America/New_York", "XNYS").
		WillReturnRows(sqlmock.NewRows(exchangeCols).
			AddRow(1, "NYSE Updated", "NYSE", "XNYS", "United States", "USD", "America/New_York"))

	resp, err := s.UpdateStockExchange(context.Background(), &pb.UpdateStockExchangeRequest{
		MicCode:  "XNYS",
		Name:     "NYSE Updated",
		Acronym:  "NYSE",
		Polity:   "United States",
		Currency: "USD",
		Timezone: "America/New_York",
	})
	require.NoError(t, err)
	assert.Equal(t, "NYSE Updated", resp.Exchange.Name)
}

func TestUpdateStockExchange_NotFound(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("UPDATE stock_exchanges").
		WillReturnError(sql.ErrNoRows)

	_, err := s.UpdateStockExchange(context.Background(), &pb.UpdateStockExchangeRequest{
		MicCode: "XXXX", Name: "X", Acronym: "X", Polity: "X", Currency: "X", Timezone: "UTC",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateStockExchange_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("UPDATE stock_exchanges").
		WillReturnError(sql.ErrConnDone)

	_, err := s.UpdateStockExchange(context.Background(), &pb.UpdateStockExchangeRequest{
		MicCode: "XNYS", Name: "X", Acronym: "X", Polity: "X", Currency: "X", Timezone: "UTC",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── DeleteStockExchange ───────────────────────────────────────────────────────

func TestDeleteStockExchange_Success(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectExec("DELETE FROM stock_exchanges").
		WithArgs("XNYS").
		WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := s.DeleteStockExchange(context.Background(), &pb.DeleteStockExchangeRequest{MicCode: "XNYS"})
	require.NoError(t, err)
}

func TestDeleteStockExchange_NotFound(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectExec("DELETE FROM stock_exchanges").
		WithArgs("XXXX").
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := s.DeleteStockExchange(context.Background(), &pb.DeleteStockExchangeRequest{MicCode: "XXXX"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestDeleteStockExchange_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectExec("DELETE FROM stock_exchanges").
		WillReturnError(sql.ErrConnDone)

	_, err := s.DeleteStockExchange(context.Background(), &pb.DeleteStockExchangeRequest{MicCode: "XNYS"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetTestMode ───────────────────────────────────────────────────────────────

func TestGetTestMode_DefaultFalse(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(false))

	resp, err := s.GetTestMode(context.Background(), &pb.GetTestModeRequest{})
	require.NoError(t, err)
	assert.False(t, resp.Enabled)
}

func TestGetTestMode_True(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(true))

	resp, err := s.GetTestMode(context.Background(), &pb.GetTestModeRequest{})
	require.NoError(t, err)
	assert.True(t, resp.Enabled)
}

func TestGetTestMode_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetTestMode(context.Background(), &pb.GetTestModeRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── SetTestMode ───────────────────────────────────────────────────────────────

func TestSetTestMode_Enable(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectExec("UPDATE settings").
		WithArgs(true).
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, err := s.SetTestMode(context.Background(), &pb.SetTestModeRequest{Enabled: true})
	require.NoError(t, err)
	assert.True(t, resp.Enabled)
}

func TestSetTestMode_Disable(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectExec("UPDATE settings").
		WithArgs(false).
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, err := s.SetTestMode(context.Background(), &pb.SetTestModeRequest{Enabled: false})
	require.NoError(t, err)
	assert.False(t, resp.Enabled)
}

func TestSetTestMode_DBError(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectExec("UPDATE settings").
		WillReturnError(sql.ErrConnDone)

	_, err := s.SetTestMode(context.Background(), &pb.SetTestModeRequest{Enabled: true})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── IsExchangeOpen (test mode) ────────────────────────────────────────────────

func TestIsExchangeOpen_TestModeOn(t *testing.T) {
	s, mock := newServer(t)
	mock.ExpectQuery("SELECT test_mode_enabled FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"test_mode_enabled"}).AddRow(true))

	resp, err := s.IsExchangeOpen(context.Background(), &pb.IsExchangeOpenRequest{MicCode: "XNYS"})
	require.NoError(t, err)
	assert.True(t, resp.IsOpen)
	assert.Equal(t, "test_mode", resp.Segment)
	assert.Equal(t, "XNYS", resp.MicCode)
	// No further DB queries should have been made
	require.NoError(t, mock.ExpectationsWereMet())
}
