package handlers_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func newOrderServer(t *testing.T) (*handlers.OrderServer, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	employeeDB, empMock, err := sqlmock.New()
	require.NoError(t, err)
	srv := &handlers.OrderServer{DB: db, EmployeeDB: employeeDB}
	t.Cleanup(func() { _ = db.Close(); _ = employeeDB.Close() })
	return srv, dbMock, empMock
}

func newOrderServerWithSecDB(t *testing.T) (*handlers.OrderServer, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	employeeDB, empMock, err := sqlmock.New()
	require.NoError(t, err)
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	accountDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	srv := &handlers.OrderServer{DB: db, EmployeeDB: employeeDB, SecuritiesDB: secDB, AccountDB: accountDB, PortfolioClient: &mockPortfolioClient{}}
	t.Cleanup(func() { _ = db.Close(); _ = employeeDB.Close(); _ = secDB.Close(); _ = accountDB.Close() })
	return srv, dbMock, empMock, secMock, accMock
}

// orderCols is the ordered list of columns returned by all order SELECT queries.
var orderCols = []string{
	"id", "user_id", "user_type", "asset_id", "order_type",
	"quantity", "contract_size", "price_per_unit", "limit_value", "stop_value",
	"direction", "status", "approved_by", "is_done", "last_modification",
	"remaining_portions", "after_hours", "is_aon", "is_margin", "account_id", "fund_id",
}

// addOrderRow appends a fully-populated order row to a sqlmock Rows object.
func addOrderRow(rows *sqlmock.Rows, id, userID int64, userType string, assetID int64,
	orderType string, quantity, contractSize int32, pricePerUnit float64,
	limitVal, stopVal interface{}, direction, orderStatus string, approvedBy interface{},
	isDone bool, ts time.Time, remaining int32, afterHours, isAON, isMargin bool, accountID int64,
) *sqlmock.Rows {
	return rows.AddRow(
		id, userID, userType, assetID, orderType,
		quantity, contractSize, pricePerUnit, limitVal, stopVal,
		direction, orderStatus, approvedBy, isDone, ts,
		remaining, afterHours, isAON, isMargin, accountID, int64(0),
	)
}

// mockSecClient is a configurable mock for pb_sec.SecuritiesServiceClient.
type mockSecClient struct {
	getListingById   func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error)
	getWorkingHours  func(ctx context.Context, in *pb_sec.GetWorkingHoursRequest, opts ...grpc.CallOption) (*pb_sec.GetWorkingHoursResponse, error)
	getExchangeByMIC func(ctx context.Context, in *pb_sec.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb_sec.GetStockExchangeByMICResponse, error)
	isExchangeOpen   func(ctx context.Context, in *pb_sec.IsExchangeOpenRequest, opts ...grpc.CallOption) (*pb_sec.IsExchangeOpenResponse, error)
}

func (m *mockSecClient) Ping(ctx context.Context, in *pb_sec.PingRequest, opts ...grpc.CallOption) (*pb_sec.PingResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) GetStockExchanges(ctx context.Context, in *pb_sec.GetStockExchangesRequest, opts ...grpc.CallOption) (*pb_sec.GetStockExchangesResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) GetStockExchangeByMIC(ctx context.Context, in *pb_sec.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb_sec.GetStockExchangeByMICResponse, error) {
	if m.getExchangeByMIC != nil {
		return m.getExchangeByMIC(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) GetStockExchangeById(ctx context.Context, in *pb_sec.GetStockExchangeByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetStockExchangeByIdResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) CreateStockExchange(ctx context.Context, in *pb_sec.CreateStockExchangeRequest, opts ...grpc.CallOption) (*pb_sec.CreateStockExchangeResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) UpdateStockExchange(ctx context.Context, in *pb_sec.UpdateStockExchangeRequest, opts ...grpc.CallOption) (*pb_sec.UpdateStockExchangeResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) DeleteStockExchange(ctx context.Context, in *pb_sec.DeleteStockExchangeRequest, opts ...grpc.CallOption) (*pb_sec.DeleteStockExchangeResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) GetWorkingHours(ctx context.Context, in *pb_sec.GetWorkingHoursRequest, opts ...grpc.CallOption) (*pb_sec.GetWorkingHoursResponse, error) {
	if m.getWorkingHours != nil {
		return m.getWorkingHours(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) SetWorkingHours(ctx context.Context, in *pb_sec.SetWorkingHoursRequest, opts ...grpc.CallOption) (*pb_sec.SetWorkingHoursResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) GetHolidays(ctx context.Context, in *pb_sec.GetHolidaysRequest, opts ...grpc.CallOption) (*pb_sec.GetHolidaysResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) AddHoliday(ctx context.Context, in *pb_sec.AddHolidayRequest, opts ...grpc.CallOption) (*pb_sec.AddHolidayResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) DeleteHoliday(ctx context.Context, in *pb_sec.DeleteHolidayRequest, opts ...grpc.CallOption) (*pb_sec.DeleteHolidayResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) IsExchangeOpen(ctx context.Context, in *pb_sec.IsExchangeOpenRequest, opts ...grpc.CallOption) (*pb_sec.IsExchangeOpenResponse, error) {
	if m.isExchangeOpen != nil {
		return m.isExchangeOpen(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) GetTestMode(ctx context.Context, in *pb_sec.GetTestModeRequest, opts ...grpc.CallOption) (*pb_sec.GetTestModeResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) SetTestMode(ctx context.Context, in *pb_sec.SetTestModeRequest, opts ...grpc.CallOption) (*pb_sec.SetTestModeResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) GetListings(ctx context.Context, in *pb_sec.GetListingsRequest, opts ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) GetListingById(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
	if m.getListingById != nil {
		return m.getListingById(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) GetListingHistory(ctx context.Context, in *pb_sec.GetListingHistoryRequest, opts ...grpc.CallOption) (*pb_sec.GetListingHistoryResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) CreatePriceAlert(ctx context.Context, in *pb_sec.CreatePriceAlertRequest, opts ...grpc.CallOption) (*pb_sec.CreatePriceAlertResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) ListPriceAlerts(ctx context.Context, in *pb_sec.ListPriceAlertsRequest, opts ...grpc.CallOption) (*pb_sec.ListPriceAlertsResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockSecClient) DeletePriceAlert(ctx context.Context, in *pb_sec.DeletePriceAlertRequest, opts ...grpc.CallOption) (*pb_sec.DeletePriceAlertResponse, error) {
	return nil, fmt.Errorf("not mocked")
}

// mockPortfolioClient is a configurable mock for pb_portfolio.PortfolioServiceClient.
type mockPortfolioClient struct {
	getPortfolio func(ctx context.Context, in *pb_portfolio.GetPortfolioRequest, opts ...grpc.CallOption) (*pb_portfolio.GetPortfolioResponse, error)
}

func (m *mockPortfolioClient) UpdateHolding(ctx context.Context, in *pb_portfolio.UpdateHoldingRequest, opts ...grpc.CallOption) (*pb_portfolio.UpdateHoldingResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) GetPortfolio(ctx context.Context, in *pb_portfolio.GetPortfolioRequest, opts ...grpc.CallOption) (*pb_portfolio.GetPortfolioResponse, error) {
	if m.getPortfolio != nil {
		return m.getPortfolio(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) GetProfit(ctx context.Context, in *pb_portfolio.GetProfitRequest, opts ...grpc.CallOption) (*pb_portfolio.GetProfitResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) SetPublicAmount(ctx context.Context, in *pb_portfolio.SetPublicAmountRequest, opts ...grpc.CallOption) (*pb_portfolio.SetPublicAmountResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) GetMyTax(ctx context.Context, in *pb_portfolio.GetMyTaxRequest, opts ...grpc.CallOption) (*pb_portfolio.GetMyTaxResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) GetTaxList(ctx context.Context, in *pb_portfolio.GetTaxListRequest, opts ...grpc.CallOption) (*pb_portfolio.GetTaxListResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) CollectTax(ctx context.Context, in *pb_portfolio.CollectTaxRequest, opts ...grpc.CallOption) (*pb_portfolio.CollectTaxResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) CollectTaxForUser(ctx context.Context, in *pb_portfolio.CollectTaxForUserRequest, opts ...grpc.CallOption) (*pb_portfolio.CollectTaxForUserResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) SetPublicMode(ctx context.Context, in *pb_portfolio.SetPublicModeRequest, opts ...grpc.CallOption) (*pb_portfolio.SetPublicModeResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) CreateWatchlist(ctx context.Context, in *pb_portfolio.CreateWatchlistRequest, opts ...grpc.CallOption) (*pb_portfolio.WatchlistResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) ListWatchlists(ctx context.Context, in *pb_portfolio.ListWatchlistsRequest, opts ...grpc.CallOption) (*pb_portfolio.ListWatchlistsResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) DeleteWatchlist(ctx context.Context, in *pb_portfolio.DeleteWatchlistRequest, opts ...grpc.CallOption) (*pb_portfolio.DeleteWatchlistResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) AddWatchlistItem(ctx context.Context, in *pb_portfolio.AddWatchlistItemRequest, opts ...grpc.CallOption) (*pb_portfolio.WatchlistItemResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) RemoveWatchlistItem(ctx context.Context, in *pb_portfolio.RemoveWatchlistItemRequest, opts ...grpc.CallOption) (*pb_portfolio.RemoveWatchlistItemResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) GetWatchlistItems(ctx context.Context, in *pb_portfolio.GetWatchlistItemsRequest, opts ...grpc.CallOption) (*pb_portfolio.GetWatchlistItemsResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) CreateDividendPayout(ctx context.Context, in *pb_portfolio.CreateDividendPayoutRequest, opts ...grpc.CallOption) (*pb_portfolio.DividendPayoutResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) GetDividendHistory(ctx context.Context, in *pb_portfolio.GetDividendHistoryRequest, opts ...grpc.CallOption) (*pb_portfolio.GetDividendHistoryResponse, error) {
	return nil, fmt.Errorf("not mocked")
}
func (m *mockPortfolioClient) GetHoldersByListing(ctx context.Context, in *pb_portfolio.GetHoldersByListingRequest, opts ...grpc.CallOption) (*pb_portfolio.GetHoldersByListingResponse, error) {
	return nil, fmt.Errorf("not mocked")
}

// ─────────────────────────────────────────────
// Ping
// ─────────────────────────────────────────────

func TestPing(t *testing.T) {
	srv := &handlers.OrderServer{}
	resp, err := srv.Ping(context.Background(), &pb.PingRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "order-service OK", resp.Message)
}

// ─────────────────────────────────────────────
// GetOrderById
// ─────────────────────────────────────────────

func TestGetOrderById_NotFound(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	dbMock.ExpectQuery("SELECT id, user_id").
		WillReturnError(sql.ErrNoRows)

	_, err := srv.GetOrderById(context.Background(), &pb.GetOrderByIdRequest{Id: 99})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetOrderById_DBError(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	dbMock.ExpectQuery("SELECT id, user_id").
		WillReturnError(sql.ErrConnDone)

	_, err := srv.GetOrderById(context.Background(), &pb.GetOrderByIdRequest{Id: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetOrderById_Happy(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 1, 10, "CLIENT", 5, "MARKET", 100, 1, 150.0, nil, nil, "BUY", "APPROVED", nil, false, ts, 100, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)

	resp, err := srv.GetOrderById(context.Background(), &pb.GetOrderByIdRequest{Id: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Order.Id)
	assert.Equal(t, int64(10), resp.Order.UserId)
	assert.Equal(t, "MARKET", resp.Order.OrderType)
	assert.Equal(t, "APPROVED", resp.Order.Status)
	assert.Equal(t, "BUY", resp.Order.Direction)
}

// ─────────────────────────────────────────────
// ListOrders
// ─────────────────────────────────────────────

func TestListOrders_DBError(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnError(sql.ErrConnDone)

	_, err := srv.ListOrders(context.Background(), &pb.ListOrdersRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestListOrders_Empty(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(sqlmock.NewRows(orderCols))

	resp, err := srv.ListOrders(context.Background(), &pb.ListOrdersRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Orders)
}

func TestListOrders_Happy(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 1, 10, "EMPLOYEE", 5, "LIMIT", 50, 1, 200.0, func() *float64 { v := 205.0; return &v }(), nil, "SELL", "PENDING", nil, false, ts, 50, false, false, false, 10)
	addOrderRow(rows, 2, 11, "CLIENT", 6, "MARKET", 10, 1, 100.0, nil, nil, "BUY", "APPROVED", nil, false, ts, 10, false, false, false, 11)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)

	resp, err := srv.ListOrders(context.Background(), &pb.ListOrdersRequest{Status: "ALL"})
	require.NoError(t, err)
	require.Len(t, resp.Orders, 2)
	assert.Equal(t, int64(1), resp.Orders[0].Id)
	assert.Equal(t, "PENDING", resp.Orders[0].Status)
	assert.Equal(t, int64(2), resp.Orders[1].Id)
	assert.Equal(t, "APPROVED", resp.Orders[1].Status)
}

// ─────────────────────────────────────────────
// ApproveOrder
// ─────────────────────────────────────────────

func TestApproveOrder_NotFound(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnError(sql.ErrNoRows)

	_, err := srv.ApproveOrder(context.Background(), &pb.ApproveOrderRequest{OrderId: 99, SupervisorId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestApproveOrder_DBError(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnError(sql.ErrConnDone)

	_, err := srv.ApproveOrder(context.Background(), &pb.ApproveOrderRequest{OrderId: 1, SupervisorId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestApproveOrder_NotPending(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Now()

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 1, 10, "CLIENT", 5, "MARKET", 10, 1, 100.0, nil, nil, "BUY", "APPROVED", nil, false, ts, 10, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)

	_, err := srv.ApproveOrder(context.Background(), &pb.ApproveOrderRequest{OrderId: 1, SupervisorId: 2})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestApproveOrder_UpdateError(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Now()

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 1, 10, "CLIENT", 5, "MARKET", 10, 1, 100.0, nil, nil, "BUY", "PENDING", nil, false, ts, 10, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)
	dbMock.ExpectExec("UPDATE orders SET status").WillReturnError(sql.ErrConnDone)

	_, err := srv.ApproveOrder(context.Background(), &pb.ApproveOrderRequest{OrderId: 1, SupervisorId: 2})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestApproveOrder_Happy_ClientOrder(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Now()

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 1, 10, "CLIENT", 5, "MARKET", 10, 1, 100.0, nil, nil, "BUY", "PENDING", nil, false, ts, 10, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)
	dbMock.ExpectExec("UPDATE orders SET status").WillReturnResult(sqlmock.NewResult(1, 1))
	// CLIENT user → IsActuary not queried

	_, err := srv.ApproveOrder(context.Background(), &pb.ApproveOrderRequest{OrderId: 1, SupervisorId: 2})
	require.NoError(t, err)
}

func TestApproveOrder_Happy_EmployeeActuary(t *testing.T) {
	srv, dbMock, empMock := newOrderServer(t)
	ts := time.Now()

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 1, 20, "EMPLOYEE", 5, "MARKET", 10, 1, 100.0, nil, nil, "BUY", "PENDING", nil, false, ts, 10, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)
	dbMock.ExpectExec("UPDATE orders SET status").WillReturnResult(sqlmock.NewResult(1, 1))
	// IsActuary query
	empMock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// DeductActuaryUsedLimit
	empMock.ExpectExec("UPDATE actuary_info SET used_limit").WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := srv.ApproveOrder(context.Background(), &pb.ApproveOrderRequest{OrderId: 1, SupervisorId: 5})
	require.NoError(t, err)
}

// ─────────────────────────────────────────────
// DeclineOrder
// ─────────────────────────────────────────────

func TestDeclineOrder_NotFound(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnError(sql.ErrNoRows)

	_, err := srv.DeclineOrder(context.Background(), &pb.DeclineOrderRequest{OrderId: 99, SupervisorId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestDeclineOrder_DBError(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnError(sql.ErrConnDone)

	_, err := srv.DeclineOrder(context.Background(), &pb.DeclineOrderRequest{OrderId: 1, SupervisorId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestDeclineOrder_NotPending(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Now()

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 1, 10, "CLIENT", 5, "MARKET", 10, 1, 100.0, nil, nil, "BUY", "DECLINED", nil, false, ts, 0, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)

	_, err := srv.DeclineOrder(context.Background(), &pb.DeclineOrderRequest{OrderId: 1, SupervisorId: 2})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestDeclineOrder_Happy(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Now()

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 1, 10, "CLIENT", 5, "MARKET", 10, 1, 100.0, nil, nil, "BUY", "PENDING", nil, false, ts, 10, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)
	dbMock.ExpectExec("UPDATE orders SET status").WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := srv.DeclineOrder(context.Background(), &pb.DeclineOrderRequest{OrderId: 1, SupervisorId: 2})
	require.NoError(t, err)
}

// ─────────────────────────────────────────────
// CancelOrder / CancelOrderPortions
// ─────────────────────────────────────────────

func TestCancelOrder_NotFound(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnError(sql.ErrNoRows)

	_, err := srv.CancelOrder(context.Background(), &pb.CancelOrderRequest{OrderId: 99, UserId: 10})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCancelOrder_DBError(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnError(sql.ErrConnDone)

	_, err := srv.CancelOrder(context.Background(), &pb.CancelOrderRequest{OrderId: 1, UserId: 10})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCancelOrder_AlreadyDone(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Now()

	rows := sqlmock.NewRows(orderCols)
	// is_done=true → FailedPrecondition
	addOrderRow(rows, 1, 10, "CLIENT", 5, "MARKET", 10, 1, 100.0, nil, nil, "BUY", "APPROVED", nil, true, ts, 0, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)

	_, err := srv.CancelOrder(context.Background(), &pb.CancelOrderRequest{OrderId: 1, UserId: 10})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestCancelOrder_WrongOwner(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Now()

	rows := sqlmock.NewRows(orderCols)
	// user_id=10, but request userId=99 → PermissionDenied
	addOrderRow(rows, 1, 10, "CLIENT", 5, "MARKET", 10, 1, 100.0, nil, nil, "BUY", "APPROVED", nil, false, ts, 10, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)

	_, err := srv.CancelOrder(context.Background(), &pb.CancelOrderRequest{OrderId: 1, UserId: 99})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestCancelOrder_CancelExecError(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Now()

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 1, 10, "CLIENT", 5, "MARKET", 10, 1, 100.0, nil, nil, "BUY", "APPROVED", nil, false, ts, 10, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)
	dbMock.ExpectExec("UPDATE orders SET is_done").WillReturnError(sql.ErrConnDone)

	_, err := srv.CancelOrder(context.Background(), &pb.CancelOrderRequest{OrderId: 1, UserId: 10})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCancelOrder_Happy(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Now()

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 1, 10, "CLIENT", 5, "MARKET", 10, 1, 100.0, nil, nil, "BUY", "APPROVED", nil, false, ts, 10, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)
	dbMock.ExpectExec("UPDATE orders SET is_done").WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := srv.CancelOrder(context.Background(), &pb.CancelOrderRequest{OrderId: 1, UserId: 10})
	require.NoError(t, err)
}

func TestCancelOrderPortions_Happy(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Now()

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 2, 20, "EMPLOYEE", 7, "LIMIT", 50, 1, 300.0, nil, nil, "SELL", "APPROVED", nil, false, ts, 30, false, false, false, 55)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)
	dbMock.ExpectExec("UPDATE orders SET is_done").WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := srv.CancelOrderPortions(context.Background(), &pb.CancelOrderPortionsRequest{OrderId: 2, UserId: 20})
	require.NoError(t, err)
}

// ─────────────────────────────────────────────
// CreateOrder
// ─────────────────────────────────────────────

func TestCreateOrder_SecuritiesClientError(t *testing.T) {
	srv, _, _ := newOrderServer(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return nil, fmt.Errorf("securities unavailable")
		},
	}

	_, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateOrder_InsertError(t *testing.T) {
	srv, dbMock, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	// CLIENT BUY → balance check passes
	accMock.ExpectQuery("SELECT available_balance FROM accounts").WillReturnRows(
		sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(999999)),
	)
	// After-hours check: mic_code lookup fails → afterHours=false (non-fatal)
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	// INSERT fails
	dbMock.ExpectQuery("INSERT INTO orders").WillReturnError(sql.ErrConnDone)

	_, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateOrder_Happy_Client_Market(t *testing.T) {
	srv, dbMock, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	// CLIENT BUY → balance check passes
	accMock.ExpectQuery("SELECT available_balance FROM accounts").WillReturnRows(
		sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(999999)),
	)
	// After-hours: MIC lookup fails → non-fatal
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	// CLIENT user → no actuary query
	dbMock.ExpectQuery("INSERT INTO orders").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(7)),
	)

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(7), resp.OrderId)
	assert.Equal(t, "MARKET", resp.OrderType)
	assert.Equal(t, "APPROVED", resp.Status) // CLIENT always APPROVED
	assert.Greater(t, resp.ApproximatePrice, float64(0))
}

func TestCreateOrder_Happy_Employee_Actuary_Pending(t *testing.T) {
	srv, dbMock, empMock, secMock, _ := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	// After-hours: MIC lookup fails → non-fatal
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	// EMPLOYEE user → actuary info lookup: need_approval=true → PENDING
	empMock.ExpectQuery("SELECT limit_amount, used_limit, need_approval").WillReturnRows(
		sqlmock.NewRows([]string{"limit_amount", "used_limit", "need_approval"}).
			AddRow(float64(10000), float64(9000), true),
	)
	dbMock.ExpectQuery("INSERT INTO orders").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(8)),
	)

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 20, UserType: "EMPLOYEE", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(8), resp.OrderId)
	assert.Equal(t, "PENDING", resp.Status) // actuary with need_approval=true
}

func TestCreateOrder_Happy_Employee_Supervisor_Approved(t *testing.T) {
	srv, dbMock, empMock, secMock, _ := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	// EMPLOYEE but not an actuary (ErrNoRows → supervisor)
	empMock.ExpectQuery("SELECT limit_amount, used_limit, need_approval").WillReturnError(sql.ErrNoRows)
	dbMock.ExpectQuery("INSERT INTO orders").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(9)),
	)

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 30, UserType: "EMPLOYEE", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(9), resp.OrderId)
	assert.Equal(t, "APPROVED", resp.Status) // supervisor → always APPROVED
}

// determineOrderType coverage: LIMIT, STOP, STOP_LIMIT variants

func TestCreateOrder_LimitOrder(t *testing.T) {
	srv, dbMock, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	accMock.ExpectQuery("SELECT available_balance FROM accounts").WillReturnRows(
		sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(999999)),
	)
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	dbMock.ExpectQuery("INSERT INTO orders").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(10)),
	)

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42,
		Direction: "BUY", LimitValue: 200.0,
	})
	require.NoError(t, err)
	assert.Equal(t, "LIMIT", resp.OrderType)
}

func TestCreateOrder_StopOrder(t *testing.T) {
	srv, dbMock, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	accMock.ExpectQuery("SELECT available_balance FROM accounts").WillReturnRows(
		sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(999999)),
	)
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	dbMock.ExpectQuery("INSERT INTO orders").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(11)),
	)

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42,
		Direction: "BUY", StopValue: 140.0,
	})
	require.NoError(t, err)
	assert.Equal(t, "STOP", resp.OrderType)
}

func TestCreateOrder_StopLimitOrder(t *testing.T) {
	srv, dbMock, _, secMock, _ := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	srv.PortfolioClient = &mockPortfolioClient{
		getPortfolio: func(ctx context.Context, in *pb_portfolio.GetPortfolioRequest, opts ...grpc.CallOption) (*pb_portfolio.GetPortfolioResponse, error) {
			return &pb_portfolio.GetPortfolioResponse{
				Entries: []*pb_portfolio.PortfolioEntry{{ListingId: 5, Amount: 100}},
			}, nil
		},
	}
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	dbMock.ExpectQuery("INSERT INTO orders").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(12)),
	)

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42,
		Direction: "SELL", LimitValue: 145.0, StopValue: 140.0,
	})
	require.NoError(t, err)
	assert.Equal(t, "STOP_LIMIT", resp.OrderType)
}

func TestCreateOrder_InsufficientHoldings(t *testing.T) {
	srv, _, _, _, _ := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	srv.PortfolioClient = &mockPortfolioClient{
		getPortfolio: func(ctx context.Context, in *pb_portfolio.GetPortfolioRequest, opts ...grpc.CallOption) (*pb_portfolio.GetPortfolioResponse, error) {
			// Asset 5 not held at all
			return &pb_portfolio.GetPortfolioResponse{Entries: nil}, nil
		},
	}

	_, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "SELL",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestCreateOrder_Happy_Client_Sell(t *testing.T) {
	srv, dbMock, _, secMock, _ := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	srv.PortfolioClient = &mockPortfolioClient{
		getPortfolio: func(ctx context.Context, in *pb_portfolio.GetPortfolioRequest, opts ...grpc.CallOption) (*pb_portfolio.GetPortfolioResponse, error) {
			return &pb_portfolio.GetPortfolioResponse{
				Entries: []*pb_portfolio.PortfolioEntry{{ListingId: 5, Amount: 50}},
			}, nil
		},
	}
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	dbMock.ExpectQuery("INSERT INTO orders").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(16)),
	)

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "SELL",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(16), resp.OrderId)
	assert.Equal(t, "APPROVED", resp.Status)
}

// checkAfterHours coverage: MIC found but working hours call fails / no regular segment / exchange error

func TestCreateOrder_AfterHours_WorkingHoursError(t *testing.T) {
	srv, dbMock, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
		getWorkingHours: func(ctx context.Context, in *pb_sec.GetWorkingHoursRequest, opts ...grpc.CallOption) (*pb_sec.GetWorkingHoursResponse, error) {
			return nil, fmt.Errorf("working hours unavailable")
		},
	}
	// CLIENT BUY → balance check passes
	accMock.ExpectQuery("SELECT available_balance FROM accounts").WillReturnRows(
		sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(999999)),
	)
	// MIC lookup succeeds
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnRows(
		sqlmock.NewRows([]string{"mic_code"}).AddRow("XNYS"),
	)
	dbMock.ExpectQuery("INSERT INTO orders").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(13)),
	)

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(13), resp.OrderId)
}

func TestCreateOrder_AfterHours_NoRegularSegment(t *testing.T) {
	srv, dbMock, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
		getWorkingHours: func(ctx context.Context, in *pb_sec.GetWorkingHoursRequest, opts ...grpc.CallOption) (*pb_sec.GetWorkingHoursResponse, error) {
			return &pb_sec.GetWorkingHoursResponse{
				Hours: []*pb_sec.ExchangeWorkingHours{
					{Segment: "extended", OpenTime: "04:00", CloseTime: "20:00"},
				},
			}, nil
		},
	}
	accMock.ExpectQuery("SELECT available_balance FROM accounts").WillReturnRows(
		sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(999999)),
	)
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnRows(
		sqlmock.NewRows([]string{"mic_code"}).AddRow("XNYS"),
	)
	dbMock.ExpectQuery("INSERT INTO orders").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(14)),
	)

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(14), resp.OrderId)
}

func TestCreateOrder_AfterHours_ExchangeByMICError(t *testing.T) {
	srv, dbMock, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
		getWorkingHours: func(ctx context.Context, in *pb_sec.GetWorkingHoursRequest, opts ...grpc.CallOption) (*pb_sec.GetWorkingHoursResponse, error) {
			return &pb_sec.GetWorkingHoursResponse{
				Hours: []*pb_sec.ExchangeWorkingHours{
					{Segment: "regular", OpenTime: "09:30", CloseTime: "16:00"},
				},
			}, nil
		},
		getExchangeByMIC: func(ctx context.Context, in *pb_sec.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb_sec.GetStockExchangeByMICResponse, error) {
			return nil, fmt.Errorf("exchange lookup failed")
		},
	}
	accMock.ExpectQuery("SELECT available_balance FROM accounts").WillReturnRows(
		sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(999999)),
	)
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnRows(
		sqlmock.NewRows([]string{"mic_code"}).AddRow("XNYS"),
	)
	dbMock.ExpectQuery("INSERT INTO orders").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(15)),
	)

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(15), resp.OrderId)
}

// orderToProto coverage: non-nil optional fields (LimitValue, StopValue, ApprovedBy)

func TestGetOrderById_NonNilOptionalFields(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	ts := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	limitVal := float64(205.0)
	stopVal := float64(195.0)
	approvedBy := int64(7)

	rows := sqlmock.NewRows(orderCols)
	addOrderRow(rows, 5, 10, "EMPLOYEE", 3, "STOP_LIMIT", 20, 1, 200.0, limitVal, stopVal, "SELL", "APPROVED", approvedBy, false, ts, 20, false, false, false, 42)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)

	resp, err := srv.GetOrderById(context.Background(), &pb.GetOrderByIdRequest{Id: 5})
	require.NoError(t, err)
	assert.Equal(t, float64(205.0), resp.Order.LimitValue)
	assert.Equal(t, float64(195.0), resp.Order.StopValue)
	assert.Equal(t, int64(7), resp.Order.ApprovedBy)
}

// ─────────────────────────────────────────────
// Exchange open check (checkExchangeStatus)
// ─────────────────────────────────────────────

func TestCreateOrder_ExchangeClosed_Rejected(t *testing.T) {
	srv, _, _, secMock, _ := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
		isExchangeOpen: func(_ context.Context, _ *pb_sec.IsExchangeOpenRequest, _ ...grpc.CallOption) (*pb_sec.IsExchangeOpenResponse, error) {
			return &pb_sec.IsExchangeOpenResponse{IsOpen: false, Segment: "closed"}, nil
		},
	}
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").
		WillReturnRows(sqlmock.NewRows([]string{"mic_code"}).AddRow("XNYS"))

	_, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, err.Error(), "exchange is currently closed")
}

func TestCreateOrder_ExchangeOpen_TestMode_Allowed(t *testing.T) {
	srv, dbMock, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
		isExchangeOpen: func(_ context.Context, _ *pb_sec.IsExchangeOpenRequest, _ ...grpc.CallOption) (*pb_sec.IsExchangeOpenResponse, error) {
			return &pb_sec.IsExchangeOpenResponse{IsOpen: true, Segment: "test_mode"}, nil
		},
	}
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").
		WillReturnRows(sqlmock.NewRows([]string{"mic_code"}).AddRow("XNYS"))
	accMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(999999)))
	dbMock.ExpectQuery("INSERT INTO orders").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(20)))

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(20), resp.OrderId)
}

func TestCreateOrder_ExchangeOpen_Regular_Allowed(t *testing.T) {
	srv, dbMock, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
		isExchangeOpen: func(_ context.Context, _ *pb_sec.IsExchangeOpenRequest, _ ...grpc.CallOption) (*pb_sec.IsExchangeOpenResponse, error) {
			return &pb_sec.IsExchangeOpenResponse{IsOpen: true, Segment: "regular"}, nil
		},
	}
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").
		WillReturnRows(sqlmock.NewRows([]string{"mic_code"}).AddRow("XNYS"))
	accMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(999999)))
	dbMock.ExpectQuery("INSERT INTO orders").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(21)))

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(21), resp.OrderId)
}

func TestCreateOrder_IsExchangeOpenError_FailOpen(t *testing.T) {
	srv, dbMock, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
		isExchangeOpen: func(_ context.Context, _ *pb_sec.IsExchangeOpenRequest, _ ...grpc.CallOption) (*pb_sec.IsExchangeOpenResponse, error) {
			return nil, fmt.Errorf("securities-service unavailable")
		},
	}
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").
		WillReturnRows(sqlmock.NewRows([]string{"mic_code"}).AddRow("XNYS"))
	accMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(999999)))
	dbMock.ExpectQuery("INSERT INTO orders").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(22)))

	// When IsExchangeOpen errors, order should still be created (fail open)
	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(22), resp.OrderId)
}

// CancelOrderPortions error path

func TestCancelOrderPortions_NotFound(t *testing.T) {
	srv, dbMock, _ := newOrderServer(t)
	dbMock.ExpectQuery("SELECT id, user_id").WillReturnError(sql.ErrNoRows)

	_, err := srv.CancelOrderPortions(context.Background(), &pb.CancelOrderPortionsRequest{OrderId: 99, UserId: 10})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ── GetActuaryProfits ─────────────────────────────────────────────────────────

func newOrderServerWithExchangeDB(t *testing.T) (*handlers.OrderServer, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	srv := &handlers.OrderServer{DB: db, SecuritiesDB: secDB, ExchangeDB: exchDB}
	t.Cleanup(func() { _ = db.Close(); _ = secDB.Close(); _ = exchDB.Close() })
	return srv, dbMock, secMock, exchMock
}

func TestGetActuaryProfits_NoOrders(t *testing.T) {
	srv, dbMock, secMock, _ := newOrderServerWithExchangeDB(t)

	dbMock.ExpectQuery(`SELECT o\.user_id, o\.asset_id`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "asset_id", "net_pnl"}))

	resp, err := srv.GetActuaryProfits(context.Background(), &pb.GetActuaryProfitsRequest{UserIds: []int64{1, 2}})
	require.NoError(t, err)
	assert.Empty(t, resp.Profits)
	require.NoError(t, dbMock.ExpectationsWereMet())
	require.NoError(t, secMock.ExpectationsWereMet())
}

func TestGetActuaryProfits_RSDProfit(t *testing.T) {
	srv, dbMock, secMock, _ := newOrderServerWithExchangeDB(t)

	// Two rows for two users, both using asset_id=10 (RSD exchange)
	rows := sqlmock.NewRows([]string{"user_id", "asset_id", "net_pnl"}).
		AddRow(int64(1), int64(10), 300.0).
		AddRow(int64(2), int64(10), 1000.0)
	dbMock.ExpectQuery(`SELECT o\.user_id, o\.asset_id`).WillReturnRows(rows)
	// SecuritiesDB: currency lookup for asset 10 → RSD (queried once because asset 10 is unique)
	secMock.ExpectQuery(`SELECT e\.currency FROM listing`).WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("RSD"))

	resp, err := srv.GetActuaryProfits(context.Background(), &pb.GetActuaryProfitsRequest{UserIds: []int64{1, 2}})
	require.NoError(t, err)
	require.Len(t, resp.Profits, 2)

	profitMap := map[int64]float64{}
	for _, p := range resp.Profits {
		profitMap[p.UserId] = p.ProfitRsd
	}
	assert.InDelta(t, 300.0, profitMap[1], 0.01)
	assert.InDelta(t, 1000.0, profitMap[2], 0.01)
	require.NoError(t, dbMock.ExpectationsWereMet())
	require.NoError(t, secMock.ExpectationsWereMet())
}

func TestGetActuaryProfits_USDConversion(t *testing.T) {
	srv, dbMock, secMock, exchMock := newOrderServerWithExchangeDB(t)

	rows := sqlmock.NewRows([]string{"user_id", "asset_id", "net_pnl"}).
		AddRow(int64(1), int64(20), 100.0)
	dbMock.ExpectQuery(`SELECT o\.user_id, o\.asset_id`).WillReturnRows(rows)
	secMock.ExpectQuery(`SELECT e\.currency FROM listing`).WithArgs(int64(20)).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	exchMock.ExpectQuery(`SELECT selling_rate FROM daily_exchange_rates`).
		WillReturnRows(sqlmock.NewRows([]string{"selling_rate"}).AddRow(117.0))

	resp, err := srv.GetActuaryProfits(context.Background(), &pb.GetActuaryProfitsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Profits, 1)
	assert.InDelta(t, 11700.0, resp.Profits[0].ProfitRsd, 0.01)
	require.NoError(t, dbMock.ExpectationsWereMet())
	require.NoError(t, secMock.ExpectationsWereMet())
	require.NoError(t, exchMock.ExpectationsWereMet())
}

func TestGetActuaryProfits_DBError(t *testing.T) {
	srv, dbMock, _, _ := newOrderServerWithExchangeDB(t)

	dbMock.ExpectQuery(`SELECT o\.user_id, o\.asset_id`).WillReturnError(sql.ErrConnDone)

	_, err := srv.GetActuaryProfits(context.Background(), &pb.GetActuaryProfitsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetActuaryProfits_ScanError(t *testing.T) {
	srv, dbMock, _, _ := newOrderServerWithExchangeDB(t)

	// Return a row with bad type for user_id to trigger scan error.
	dbMock.ExpectQuery(`SELECT o\.user_id, o\.asset_id`).WillReturnRows(
		sqlmock.NewRows([]string{"user_id", "asset_id", "net_pnl"}).
			AddRow("bad", int64(5), float64(100)),
	)

	_, err := srv.GetActuaryProfits(context.Background(), &pb.GetActuaryProfitsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetActuaryProfits_DuplicateCurrencyCached(t *testing.T) {
	// Two assets with USD → rate fetched once, second is served from cache (line 421-422 covered).
	srv, dbMock, secMock, exchMock := newOrderServerWithExchangeDB(t)

	dbMock.ExpectQuery(`SELECT o\.user_id, o\.asset_id`).WillReturnRows(
		sqlmock.NewRows([]string{"user_id", "asset_id", "net_pnl"}).
			AddRow(int64(1), int64(5), float64(100)).
			AddRow(int64(1), int64(6), float64(200)),
	)
	// Both assets map to USD
	secMock.ExpectQuery(`SELECT e\.currency FROM listing`).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	secMock.ExpectQuery(`SELECT e\.currency FROM listing`).
		WithArgs(int64(6)).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	// Rate fetched once (first USD), second is cached
	exchMock.ExpectQuery(`SELECT selling_rate FROM daily_exchange_rates`).
		WillReturnRows(sqlmock.NewRows([]string{"selling_rate"}).AddRow(float64(118.0)))

	resp, err := srv.GetActuaryProfits(context.Background(), &pb.GetActuaryProfitsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Profits, 1)
}

func TestCreateOrder_FuturesListing(t *testing.T) {
	srv, dbMock, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, ExchangeAcronym: "NYSE"},
				Detail:  &pb_sec.GetListingByIdResponse_Futures{Futures: &pb_sec.FuturesDetail{ContractSize: 50}},
			}, nil
		},
	}
	accMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(999999)))
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	dbMock.ExpectQuery("INSERT INTO orders").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(30)))

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 2, AccountId: 42, Direction: "BUY",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(30), resp.OrderId)
	// contractSize=50, pricePerUnit=100, quantity=2 → approxPrice=10000
	assert.InDelta(t, 10000.0, resp.ApproximatePrice, 1.0)
}

func TestCreateOrder_ClientBuy_BalanceQueryError(t *testing.T) {
	srv, _, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	accMock.ExpectQuery("SELECT available_balance FROM accounts").WillReturnError(fmt.Errorf("db error"))

	_, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateOrder_ClientBuy_InsufficientFunds(t *testing.T) {
	srv, _, _, secMock, accMock := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	// Balance much lower than approxPrice (150 * 10 = 1500)
	accMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(1)))

	_, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "BUY",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, err.Error(), "insufficient funds")
}

func TestCreateOrder_Sell_PortfolioError(t *testing.T) {
	srv, _, _, secMock, _ := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 150.0, Bid: 148.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	srv.PortfolioClient = &mockPortfolioClient{
		getPortfolio: func(_ context.Context, _ *pb_portfolio.GetPortfolioRequest, _ ...grpc.CallOption) (*pb_portfolio.GetPortfolioResponse, error) {
			return nil, fmt.Errorf("portfolio service down")
		},
	}
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)

	_, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 1, UserType: "CLIENT", AssetId: 5, Quantity: 10, AccountId: 42, Direction: "SELL",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "failed to fetch portfolio")
}

func TestCreateOrder_Employee_CurrencyConversion(t *testing.T) {
	// EMPLOYEE order, non-RSD listing currency → approxPriceRSD = approxPrice * rate.
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	empDB, empMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = empDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()

	srv := &handlers.OrderServer{
		DB: db, EmployeeDB: empDB, SecuritiesDB: secDB, ExchangeDB: exchDB,
		PortfolioClient: &mockPortfolioClient{},
		SecuritiesClient: &mockSecClient{
			getListingById: func(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
				return &pb_sec.GetListingByIdResponse{
					Summary: &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, ExchangeAcronym: "NYSE"},
				}, nil
			},
		},
	}

	// After-hours check
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	// EMPLOYEE: listing currency = USD (non-RSD)
	secMock.ExpectQuery("SELECT e.currency FROM listing").
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	// Exchange rate = 118
	exchMock.ExpectQuery("SELECT selling_rate FROM daily_exchange_rates").
		WillReturnRows(sqlmock.NewRows([]string{"selling_rate"}).AddRow(float64(118.0)))
	// Actuary info: needApproval=false, will auto-approve
	empMock.ExpectQuery("SELECT limit_amount, used_limit, need_approval").
		WillReturnRows(sqlmock.NewRows([]string{"limit_amount", "used_limit", "need_approval"}).
			AddRow(float64(1000000), float64(0), false))
	// Insert order
	dbMock.ExpectQuery("INSERT INTO orders").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(50)))
	// DeductActuaryUsedLimit (auto-approved EMPLOYEE BUY)
	empMock.ExpectExec("UPDATE actuary_info SET used_limit").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 20, UserType: "EMPLOYEE", AssetId: 5, Quantity: 1, AccountId: 42, Direction: "BUY",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(50), resp.OrderId)
	assert.Equal(t, "APPROVED", resp.Status)
}

func TestCreateOrder_Actuary_AutoApproved_Buy_DeductError(t *testing.T) {
	srv, dbMock, empMock, secMock, _ := newOrderServerWithSecDB(t)
	srv.SecuritiesClient = &mockSecClient{
		getListingById: func(_ context.Context, _ *pb_sec.GetListingByIdRequest, _ ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
			return &pb_sec.GetListingByIdResponse{
				Summary: &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, ExchangeAcronym: "NYSE"},
			}, nil
		},
	}
	secMock.ExpectQuery("SELECT mic_code FROM stock_exchanges").WillReturnError(sql.ErrNoRows)
	// Actuary info: no approval needed, lots of limit → auto-approve
	empMock.ExpectQuery("SELECT limit_amount, used_limit, need_approval").
		WillReturnRows(sqlmock.NewRows([]string{"limit_amount", "used_limit", "need_approval"}).
			AddRow(float64(1000000), float64(0), false))
	dbMock.ExpectQuery("INSERT INTO orders").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(51)))
	// DeductActuaryUsedLimit returns error (non-fatal)
	empMock.ExpectExec("UPDATE actuary_info SET used_limit").
		WillReturnError(fmt.Errorf("db error"))

	resp, err := srv.CreateOrder(context.Background(), &pb.CreateOrderRequest{
		UserId: 20, UserType: "EMPLOYEE", AssetId: 5, Quantity: 1, AccountId: 42, Direction: "BUY",
	})
	// Error is non-fatal, order still succeeds
	require.NoError(t, err)
	assert.Equal(t, int64(51), resp.OrderId)
	assert.Equal(t, "APPROVED", resp.Status)
}
