package execution

import (
	"context"
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/models"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_exchange "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/exchange"
	pb_fund "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	pb_loan "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
)

// ── minimal mock clients ──────────────────────────────────────────────────────

type mockEmpClient struct {
	employee *pb_emp.Employee
	err      error
}

func (m *mockEmpClient) GetAllEmployees(ctx context.Context, in *pb_emp.GetAllEmployeesRequest, opts ...grpc.CallOption) (*pb_emp.GetAllEmployeesResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) SearchEmployees(ctx context.Context, in *pb_emp.SearchEmployeesRequest, opts ...grpc.CallOption) (*pb_emp.SearchEmployeesResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) GetEmployeeCredentials(ctx context.Context, in *pb_emp.GetEmployeeCredentialsRequest, opts ...grpc.CallOption) (*pb_emp.GetEmployeeCredentialsResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) CreateEmployee(ctx context.Context, in *pb_emp.CreateEmployeeRequest, opts ...grpc.CallOption) (*pb_emp.CreateEmployeeResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) GetEmployeeById(ctx context.Context, in *pb_emp.GetEmployeeByIdRequest, opts ...grpc.CallOption) (*pb_emp.GetEmployeeByIdResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &pb_emp.GetEmployeeByIdResponse{Employee: m.employee}, nil
}
func (m *mockEmpClient) UpdateEmployee(ctx context.Context, in *pb_emp.UpdateEmployeeRequest, opts ...grpc.CallOption) (*pb_emp.UpdateEmployeeResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) ActivateEmployee(ctx context.Context, in *pb_emp.ActivateEmployeeRequest, opts ...grpc.CallOption) (*pb_emp.ActivateEmployeeResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) GetEmployeeByEmail(ctx context.Context, in *pb_emp.GetEmployeeByEmailRequest, opts ...grpc.CallOption) (*pb_emp.GetEmployeeByEmailResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) UpdatePassword(ctx context.Context, in *pb_emp.UpdatePasswordRequest, opts ...grpc.CallOption) (*pb_emp.UpdatePasswordResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) GetActuaries(ctx context.Context, in *pb_emp.GetActuariesRequest, opts ...grpc.CallOption) (*pb_emp.GetActuariesResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) SetAgentLimit(ctx context.Context, in *pb_emp.SetAgentLimitRequest, opts ...grpc.CallOption) (*pb_emp.SetAgentLimitResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) ResetAgentUsedLimit(ctx context.Context, in *pb_emp.ResetAgentUsedLimitRequest, opts ...grpc.CallOption) (*pb_emp.ResetAgentUsedLimitResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) SetNeedApproval(ctx context.Context, in *pb_emp.SetNeedApprovalRequest, opts ...grpc.CallOption) (*pb_emp.SetNeedApprovalResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) ResetAllActuaryUsedLimits(ctx context.Context, in *pb_emp.ResetAllActuaryUsedLimitsRequest, opts ...grpc.CallOption) (*pb_emp.ResetAllActuaryUsedLimitsResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) GetActuaryPerformers(ctx context.Context, in *pb_emp.GetActuaryPerformersRequest, opts ...grpc.CallOption) (*pb_emp.GetActuaryPerformersResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) GetSupervisors(ctx context.Context, in *pb_emp.GetSupervisorsRequest, opts ...grpc.CallOption) (*pb_emp.GetSupervisorsResponse, error) {
	return nil, nil
}
func (m *mockEmpClient) UpdateLoginAttempts(ctx context.Context, in *pb_emp.UpdateLoginAttemptsRequest, opts ...grpc.CallOption) (*pb_emp.UpdateLoginAttemptsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockEmpClient) LogAuditEvent(ctx context.Context, in *pb_emp.AuditLogRequest, opts ...grpc.CallOption) (*pb_emp.AuditLogResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockEmpClient) ListAuditLogs(ctx context.Context, in *pb_emp.ListAuditLogsRequest, opts ...grpc.CallOption) (*pb_emp.ListAuditLogsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

type mockLoanClient struct {
	loans []*pb_loan.LoanSummary
	err   error
}

func (m *mockLoanClient) GetClientLoans(ctx context.Context, in *pb_loan.GetClientLoansRequest, opts ...grpc.CallOption) (*pb_loan.GetClientLoansResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &pb_loan.GetClientLoansResponse{Loans: m.loans}, nil
}
func (m *mockLoanClient) GetLoanDetails(ctx context.Context, in *pb_loan.GetLoanDetailsRequest, opts ...grpc.CallOption) (*pb_loan.GetLoanDetailsResponse, error) {
	return nil, nil
}
func (m *mockLoanClient) GetLoanInstallments(ctx context.Context, in *pb_loan.GetLoanInstallmentsRequest, opts ...grpc.CallOption) (*pb_loan.GetLoanInstallmentsResponse, error) {
	return nil, nil
}
func (m *mockLoanClient) SubmitLoanApplication(ctx context.Context, in *pb_loan.SubmitLoanApplicationRequest, opts ...grpc.CallOption) (*pb_loan.SubmitLoanApplicationResponse, error) {
	return nil, nil
}
func (m *mockLoanClient) ApproveLoan(ctx context.Context, in *pb_loan.ApproveLoanRequest, opts ...grpc.CallOption) (*pb_loan.ApproveLoanResponse, error) {
	return nil, nil
}
func (m *mockLoanClient) RejectLoan(ctx context.Context, in *pb_loan.RejectLoanRequest, opts ...grpc.CallOption) (*pb_loan.RejectLoanResponse, error) {
	return nil, nil
}
func (m *mockLoanClient) GetAllLoanApplications(ctx context.Context, in *pb_loan.GetAllLoanApplicationsRequest, opts ...grpc.CallOption) (*pb_loan.GetAllLoanApplicationsResponse, error) {
	return nil, nil
}
func (m *mockLoanClient) GetAllLoans(ctx context.Context, in *pb_loan.GetAllLoansRequest, opts ...grpc.CallOption) (*pb_loan.GetAllLoansResponse, error) {
	return nil, nil
}
func (m *mockLoanClient) TriggerInstallments(ctx context.Context, in *pb_loan.TriggerInstallmentsRequest, opts ...grpc.CallOption) (*pb_loan.TriggerInstallmentsResponse, error) {
	return nil, nil
}

type mockExchangeClient struct{}

func (m *mockExchangeClient) GetExchangeRates(ctx context.Context, in *pb_exchange.GetExchangeRatesRequest, opts ...grpc.CallOption) (*pb_exchange.GetExchangeRatesResponse, error) {
	return &pb_exchange.GetExchangeRatesResponse{}, nil
}

type mockExchangeClientError struct{}

func (m *mockExchangeClientError) GetExchangeRates(ctx context.Context, in *pb_exchange.GetExchangeRatesRequest, opts ...grpc.CallOption) (*pb_exchange.GetExchangeRatesResponse, error) {
	return nil, fmt.Errorf("exchange service unavailable")
}
func (m *mockExchangeClientError) ConvertAmount(ctx context.Context, in *pb_exchange.ConvertAmountRequest, opts ...grpc.CallOption) (*pb_exchange.ConvertAmountResponse, error) {
	return nil, nil
}
func (m *mockExchangeClientError) GetExchangeHistory(ctx context.Context, in *pb_exchange.GetExchangeHistoryRequest, opts ...grpc.CallOption) (*pb_exchange.GetExchangeHistoryResponse, error) {
	return nil, nil
}
func (m *mockExchangeClientError) PreviewConversion(ctx context.Context, in *pb_exchange.PreviewConversionRequest, opts ...grpc.CallOption) (*pb_exchange.PreviewConversionResponse, error) {
	return nil, nil
}

type mockSecuritiesClient struct {
	listing *pb_sec.ListingSummary
	err     error
}

func (m *mockSecuritiesClient) Ping(ctx context.Context, in *pb_sec.PingRequest, opts ...grpc.CallOption) (*pb_sec.PingResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) GetStockExchanges(ctx context.Context, in *pb_sec.GetStockExchangesRequest, opts ...grpc.CallOption) (*pb_sec.GetStockExchangesResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) GetStockExchangeByMIC(ctx context.Context, in *pb_sec.GetStockExchangeByMICRequest, opts ...grpc.CallOption) (*pb_sec.GetStockExchangeByMICResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) GetStockExchangeById(ctx context.Context, in *pb_sec.GetStockExchangeByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetStockExchangeByIdResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) CreateStockExchange(ctx context.Context, in *pb_sec.CreateStockExchangeRequest, opts ...grpc.CallOption) (*pb_sec.CreateStockExchangeResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) UpdateStockExchange(ctx context.Context, in *pb_sec.UpdateStockExchangeRequest, opts ...grpc.CallOption) (*pb_sec.UpdateStockExchangeResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) DeleteStockExchange(ctx context.Context, in *pb_sec.DeleteStockExchangeRequest, opts ...grpc.CallOption) (*pb_sec.DeleteStockExchangeResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) GetWorkingHours(ctx context.Context, in *pb_sec.GetWorkingHoursRequest, opts ...grpc.CallOption) (*pb_sec.GetWorkingHoursResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) SetWorkingHours(ctx context.Context, in *pb_sec.SetWorkingHoursRequest, opts ...grpc.CallOption) (*pb_sec.SetWorkingHoursResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) GetHolidays(ctx context.Context, in *pb_sec.GetHolidaysRequest, opts ...grpc.CallOption) (*pb_sec.GetHolidaysResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) AddHoliday(ctx context.Context, in *pb_sec.AddHolidayRequest, opts ...grpc.CallOption) (*pb_sec.AddHolidayResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) DeleteHoliday(ctx context.Context, in *pb_sec.DeleteHolidayRequest, opts ...grpc.CallOption) (*pb_sec.DeleteHolidayResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) IsExchangeOpen(ctx context.Context, in *pb_sec.IsExchangeOpenRequest, opts ...grpc.CallOption) (*pb_sec.IsExchangeOpenResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) GetTestMode(ctx context.Context, in *pb_sec.GetTestModeRequest, opts ...grpc.CallOption) (*pb_sec.GetTestModeResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) SetTestMode(ctx context.Context, in *pb_sec.SetTestModeRequest, opts ...grpc.CallOption) (*pb_sec.SetTestModeResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) GetListings(ctx context.Context, in *pb_sec.GetListingsRequest, opts ...grpc.CallOption) (*pb_sec.GetListingsResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) GetListingById(ctx context.Context, in *pb_sec.GetListingByIdRequest, opts ...grpc.CallOption) (*pb_sec.GetListingByIdResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &pb_sec.GetListingByIdResponse{Summary: m.listing}, nil
}
func (m *mockSecuritiesClient) GetListingHistory(ctx context.Context, in *pb_sec.GetListingHistoryRequest, opts ...grpc.CallOption) (*pb_sec.GetListingHistoryResponse, error) {
	return nil, nil
}
func (m *mockSecuritiesClient) CreatePriceAlert(ctx context.Context, in *pb_sec.CreatePriceAlertRequest, opts ...grpc.CallOption) (*pb_sec.CreatePriceAlertResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockSecuritiesClient) ListPriceAlerts(ctx context.Context, in *pb_sec.ListPriceAlertsRequest, opts ...grpc.CallOption) (*pb_sec.ListPriceAlertsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockSecuritiesClient) DeletePriceAlert(ctx context.Context, in *pb_sec.DeletePriceAlertRequest, opts ...grpc.CallOption) (*pb_sec.DeletePriceAlertResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockExchangeClient) ConvertAmount(ctx context.Context, in *pb_exchange.ConvertAmountRequest, opts ...grpc.CallOption) (*pb_exchange.ConvertAmountResponse, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetExchangeHistory(ctx context.Context, in *pb_exchange.GetExchangeHistoryRequest, opts ...grpc.CallOption) (*pb_exchange.GetExchangeHistoryResponse, error) {
	return nil, nil
}
func (m *mockExchangeClient) PreviewConversion(ctx context.Context, in *pb_exchange.PreviewConversionRequest, opts ...grpc.CallOption) (*pb_exchange.PreviewConversionResponse, error) {
	return nil, nil
}

// ── listingCurrency ───────────────────────────────────────────────────────────

func TestListingCurrency_Success(t *testing.T) {
	secDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	mock.ExpectQuery(`SELECT e\.currency`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))

	s := &Scheduler{SecuritiesDB: secDB}
	code, err := s.listingCurrency(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "USD", code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListingCurrency_NotFound(t *testing.T) {
	secDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	mock.ExpectQuery(`SELECT e\.currency`).
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}))

	s := &Scheduler{SecuritiesDB: secDB}
	_, err = s.listingCurrency(context.Background(), 99)
	assert.Error(t, err)
}

// ── validateMargin ────────────────────────────────────────────────────────────

func TestValidateMargin_Employee_WithMarginPermission(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()

	accMock.ExpectQuery(`SELECT available_balance`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(100.0))

	s := &Scheduler{
		AccountDB:      accDB,
		EmployeeClient: &mockEmpClient{employee: &pb_emp.Employee{Permissions: []string{"AGENT", "MARGIN"}}},
	}
	order := models.Order{AccountID: 42, UserType: "EMPLOYEE", UserID: 1}
	// initialMarginCost > balance but MARGIN permission bypasses it
	assert.True(t, s.validateMargin(context.Background(), order, 50000.0))
}

func TestValidateMargin_Employee_NoMarginPermission_Insufficient(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()

	accMock.ExpectQuery(`SELECT available_balance`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(100.0))

	s := &Scheduler{
		AccountDB:      accDB,
		EmployeeClient: &mockEmpClient{employee: &pb_emp.Employee{Permissions: []string{"AGENT"}}},
	}
	order := models.Order{AccountID: 42, UserType: "EMPLOYEE", UserID: 1}
	assert.False(t, s.validateMargin(context.Background(), order, 50000.0))
}

func TestValidateMargin_Client_SufficientLoan(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()

	accMock.ExpectQuery(`SELECT available_balance`).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(500.0))

	loan := &pb_loan.LoanSummary{Status: "APPROVED", Amount: 20000.0}
	s := &Scheduler{
		AccountDB:  accDB,
		LoanClient: &mockLoanClient{loans: []*pb_loan.LoanSummary{loan}},
	}
	order := models.Order{AccountID: 10, UserType: "CLIENT", UserID: 5}
	assert.True(t, s.validateMargin(context.Background(), order, 15000.0))
}

func TestValidateMargin_Client_InsufficientFunds(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()

	accMock.ExpectQuery(`SELECT available_balance`).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(500.0))

	s := &Scheduler{
		AccountDB:  accDB,
		LoanClient: &mockLoanClient{loans: []*pb_loan.LoanSummary{}},
	}
	order := models.Order{AccountID: 10, UserType: "CLIENT", UserID: 5}
	assert.False(t, s.validateMargin(context.Background(), order, 15000.0))
}

// ── settleAccountAndCommission ────────────────────────────────────────────────

func TestSettle_Buy_SameCurrency(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()

	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()

	// 1. look up account currency_id
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	// 2. resolve currency code → "USD"
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	// 3. debit buyer (same currency, no conversion)
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(-110.0, int64(1), 110.0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// 4. credit commission to bank account: look up currency id
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WithArgs("USD").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	// 5. find bank account
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	// 6. credit bank account
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(10.0, int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{
		AccountDB:      accDB,
		ExchangeDB:     exchDB,
		ExchangeClient: &mockExchangeClient{},
	}
	order := models.Order{AccountID: 1, Direction: "BUY", UserType: "EMPLOYEE"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	require.NoError(t, err)
	assert.NoError(t, accMock.ExpectationsWereMet())
	assert.NoError(t, exchMock.ExpectationsWereMet())
}

func TestSettle_Sell_SameCurrency(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()

	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()

	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	// credit seller: totalPrice - commission = 100 - 10 = 90
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(90.0, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WithArgs("USD").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(10.0, int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{
		AccountDB:      accDB,
		ExchangeDB:     exchDB,
		ExchangeClient: &mockExchangeClient{},
	}
	order := models.Order{AccountID: 1, Direction: "SELL", UserType: "EMPLOYEE"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	require.NoError(t, err)
	assert.NoError(t, accMock.ExpectationsWereMet())
	assert.NoError(t, exchMock.ExpectationsWereMet())
}

func TestSettle_Buy_InsufficientFunds(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()

	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()

	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	// UPDATE returns 0 rows (insufficient funds)
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(-110.0, int64(1), 110.0).
		WillReturnResult(sqlmock.NewResult(0, 0))

	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB, ExchangeClient: &mockExchangeClient{}}
	order := models.Order{AccountID: 1, Direction: "BUY", UserType: "EMPLOYEE"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient funds")
}

// ── declineExpiredOrders ──────────────────────────────────────────────────────

func TestDeclineExpiredOrders_NoPendingOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`SELECT id, user_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "user_type", "asset_id", "order_type", "quantity", "contract_size",
			"price_per_unit", "limit_value", "stop_value", "direction", "status", "approved_by",
			"is_done", "last_modification", "remaining_portions", "after_hours", "is_aon", "is_margin", "account_id", "fund_id",
		}))

	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	s := &Scheduler{DB: db, SecuritiesDB: secDB}
	s.declineExpiredOrders()
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.NoError(t, secMock.ExpectationsWereMet())
}

func TestDeclineExpiredOrders_ExpiredFutures(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	pastDate := time.Now().UTC().AddDate(0, 0, -2).Format("2006-01-02")

	mock.ExpectQuery(`SELECT id, user_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "user_type", "asset_id", "order_type", "quantity", "contract_size",
			"price_per_unit", "limit_value", "stop_value", "direction", "status", "approved_by",
			"is_done", "last_modification", "remaining_portions", "after_hours", "is_aon", "is_margin", "account_id", "fund_id",
		}).AddRow(
			int64(7), int64(1), "EMPLOYEE", int64(10), "MARKET", int32(1), int32(1),
			100.0, nil, nil, "BUY", "PENDING", nil,
			false, time.Now(), int32(1), false, false, false, int64(1), int64(0),
		))

	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	secMock.ExpectQuery(`SELECT settlement_date`).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"settlement_date"}).AddRow(pastDate))

	// DeclineOrder: fetch order status, then update
	mock.ExpectQuery(`SELECT id, user_id`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "user_type", "asset_id", "order_type", "quantity", "contract_size",
			"price_per_unit", "limit_value", "stop_value", "direction", "status", "approved_by",
			"is_done", "last_modification", "remaining_portions", "after_hours", "is_aon", "is_margin", "account_id", "fund_id",
		}).AddRow(
			int64(7), int64(1), "EMPLOYEE", int64(10), "MARKET", int32(1), int32(1),
			100.0, nil, nil, "BUY", "PENDING", nil,
			false, time.Now(), int32(1), false, false, false, int64(1), int64(0),
		))
	mock.ExpectExec(`UPDATE orders SET status`).
		WithArgs("DECLINED", nil, sqlmock.AnyArg(), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{DB: db, SecuritiesDB: secDB}
	s.declineExpiredOrders()
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.NoError(t, secMock.ExpectationsWereMet())
}

func TestDeclineExpiredOrders_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectQuery(`SELECT id, user_id`).WillReturnError(fmt.Errorf("db error"))
	s := &Scheduler{DB: db}
	s.declineExpiredOrders()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeclineExpiredOrders_NoSettlementDate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectQuery(`SELECT id, user_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "user_type", "asset_id", "order_type", "quantity", "contract_size",
			"price_per_unit", "limit_value", "stop_value", "direction", "status", "approved_by",
			"is_done", "last_modification", "remaining_portions", "after_hours", "is_aon", "is_margin", "account_id", "fund_id",
		}).AddRow(
			int64(1), int64(1), "EMPLOYEE", int64(10), "MARKET", int32(1), int32(1),
			100.0, nil, nil, "BUY", "PENDING", nil,
			false, time.Now(), int32(1), false, false, false, int64(1), int64(0),
		))
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()
	secMock.ExpectQuery(`SELECT settlement_date`).
		WillReturnRows(sqlmock.NewRows([]string{"settlement_date"}))
	s2 := &Scheduler{DB: db, SecuritiesDB: secDB}
	s2.declineExpiredOrders()
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.NoError(t, secMock.ExpectationsWereMet())
}

func TestDeclineExpiredOrders_DeclineError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	pastDate := time.Now().UTC().AddDate(0, 0, -2).Format("2006-01-02")
	mock.ExpectQuery(`SELECT id, user_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "user_type", "asset_id", "order_type", "quantity", "contract_size",
			"price_per_unit", "limit_value", "stop_value", "direction", "status", "approved_by",
			"is_done", "last_modification", "remaining_portions", "after_hours", "is_aon", "is_margin", "account_id", "fund_id",
		}).AddRow(
			int64(7), int64(1), "EMPLOYEE", int64(10), "MARKET", int32(1), int32(1),
			100.0, nil, nil, "BUY", "PENDING", nil,
			false, time.Now(), int32(1), false, false, false, int64(1), int64(0),
		))
	secDB2, secMock2, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB2.Close() }()
	secMock2.ExpectQuery(`SELECT settlement_date`).
		WillReturnRows(sqlmock.NewRows([]string{"settlement_date"}).AddRow(pastDate))
	mock.ExpectQuery(`SELECT id, user_id`).WithArgs(int64(7)).WillReturnError(fmt.Errorf("db error"))
	s3 := &Scheduler{DB: db, SecuritiesDB: secDB2}
	s3.declineExpiredOrders()
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── validateMargin balance error ──────────────────────────────────────────────

func TestValidateMargin_AccountBalanceError(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	accMock.ExpectQuery(`SELECT available_balance`).
		WithArgs(int64(42)).WillReturnError(fmt.Errorf("db error"))
	s := &Scheduler{
		AccountDB:  accDB,
		LoanClient: &mockLoanClient{loans: []*pb_loan.LoanSummary{}},
	}
	order := models.Order{AccountID: 42, UserType: "CLIENT", UserID: 5}
	assert.False(t, s.validateMargin(context.Background(), order, 50000.0))
}

// ── settleAccountAndCommission additional error paths ─────────────────────────

func TestSettle_ExchangeClientError_Continues(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB, ExchangeClient: &mockExchangeClientError{}}
	order := models.Order{AccountID: 1, Direction: "SELL", UserType: "EMPLOYEE"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	require.NoError(t, err)
}

func TestSettle_AccountCurrencyIDError(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).WillReturnError(fmt.Errorf("db error"))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "BUY"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account currency_id")
}

func TestSettle_CurrencyCodeError(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).WillReturnError(fmt.Errorf("db error"))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "BUY"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "currency code")
}

func TestSettle_Buy_ExecError(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnError(fmt.Errorf("db error"))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "BUY"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	assert.Error(t, err)
}

func TestSettle_Sell_ExecError(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnError(fmt.Errorf("db error"))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "SELL"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	assert.Error(t, err)
}

func TestSettle_ZeroCommission_NoBank(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "SELL"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 0.0, "USD")
	require.NoError(t, err)
	assert.NoError(t, accMock.ExpectationsWereMet())
}

func TestSettle_CurrencyIDLookupError(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).WillReturnError(fmt.Errorf("db error"))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "SELL"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	assert.Error(t, err)
}

func TestSettle_BankAccountIDLookupError(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).WillReturnError(fmt.Errorf("db error"))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "SELL"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	assert.Error(t, err)
}

func TestSettle_BankAccountUpdateError(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnError(fmt.Errorf("db error"))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "SELL"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	assert.Error(t, err)
}

// ── settleAccountAndCommission: currency conversion paths ─────────────────────

func TestSettle_Buy_DifferentCurrency_AccountRSD(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(1)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	exchMock.ExpectQuery(`SELECT selling_rate FROM daily_exchange_rates`).
		WillReturnRows(sqlmock.NewRows([]string{"selling_rate"}).AddRow(float64(118.0)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "BUY", UserType: "CLIENT"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	require.NoError(t, err)
}

func TestSettle_Buy_DifferentCurrency_SecurityRSD(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("EUR"))
	exchMock.ExpectQuery(`SELECT buying_rate FROM daily_exchange_rates`).
		WillReturnRows(sqlmock.NewRows([]string{"buying_rate"}).AddRow(float64(117.0)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "BUY", UserType: "EMPLOYEE"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "RSD")
	require.NoError(t, err)
}

func TestSettle_Buy_DifferentCurrency_BothForeign(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("EUR"))
	exchMock.ExpectQuery(`SELECT selling_rate FROM daily_exchange_rates`).
		WillReturnRows(sqlmock.NewRows([]string{"selling_rate"}).AddRow(float64(118.0)))
	exchMock.ExpectQuery(`SELECT buying_rate FROM daily_exchange_rates`).
		WillReturnRows(sqlmock.NewRows([]string{"buying_rate"}).AddRow(float64(117.0)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "BUY", UserType: "CLIENT"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	require.NoError(t, err)
}

func TestSettle_Sell_DifferentCurrency_AccountRSD(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(1)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	exchMock.ExpectQuery(`SELECT buying_rate FROM daily_exchange_rates`).
		WillReturnRows(sqlmock.NewRows([]string{"buying_rate"}).AddRow(float64(116.0)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "SELL", UserType: "CLIENT"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	require.NoError(t, err)
}

func TestSettle_Sell_DifferentCurrency_SecurityRSD(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("EUR"))
	exchMock.ExpectQuery(`SELECT selling_rate FROM daily_exchange_rates`).
		WillReturnRows(sqlmock.NewRows([]string{"selling_rate"}).AddRow(float64(119.0)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "SELL", UserType: "EMPLOYEE"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "RSD")
	require.NoError(t, err)
}

func TestSettle_Sell_DifferentCurrency_BothForeign(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("EUR"))
	exchMock.ExpectQuery(`SELECT buying_rate FROM daily_exchange_rates`).
		WillReturnRows(sqlmock.NewRows([]string{"buying_rate"}).AddRow(float64(116.0)))
	exchMock.ExpectQuery(`SELECT selling_rate FROM daily_exchange_rates`).
		WillReturnRows(sqlmock.NewRows([]string{"selling_rate"}).AddRow(float64(119.0)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "SELL", UserType: "CLIENT"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	require.NoError(t, err)
}

func TestSettle_DifferentCurrency_RateError(t *testing.T) {
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(1)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	exchMock.ExpectQuery(`SELECT selling_rate FROM daily_exchange_rates`).
		WillReturnError(fmt.Errorf("no rate"))
	s := &Scheduler{AccountDB: accDB, ExchangeDB: exchDB}
	order := models.Order{AccountID: 1, Direction: "BUY"}
	_, err = s.settleAccountAndCommission(context.Background(), order, 100.0, 10.0, "USD")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exchange rate")
}

// ── executeOrder early-return paths ──────────────────────────────────────────

func TestExecuteOrder_GetListingByIdError(t *testing.T) {
	s := &Scheduler{
		SecuritiesClient: &mockSecuritiesClient{err: fmt.Errorf("rpc error")},
	}
	order := models.Order{AssetID: 1, RemainingPortions: 5}
	s.executeOrder(order)
}

func TestExecuteOrder_MarginFails_Decline(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, InitialMarginCost: 50000.0}
	mock.ExpectExec(`UPDATE orders SET status`).
		WithArgs("DECLINED", nil, sqlmock.AnyArg(), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	accMock.ExpectQuery(`SELECT available_balance`).
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(100)))
	s := &Scheduler{
		DB:               db,
		AccountDB:        accDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing},
		LoanClient:       &mockLoanClient{loans: []*pb_loan.LoanSummary{}},
	}
	order := models.Order{ID: 1, AssetID: 10, IsMargin: true, AccountID: 5, UserType: "CLIENT", RemainingPortions: 1}
	s.executeOrder(order)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteOrder_ListingCurrencyError(t *testing.T) {
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()
	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0}
	secMock.ExpectQuery(`SELECT e\.currency`).WillReturnError(fmt.Errorf("db error"))
	s := &Scheduler{
		SecuritiesDB:     secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing},
	}
	order := models.Order{AssetID: 10, IsMargin: false, RemainingPortions: 5}
	s.executeOrder(order)
	assert.NoError(t, secMock.ExpectationsWereMet())
}

func TestExecuteOrder_MarginDecline_UpdateError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, InitialMarginCost: 50000.0}
	mock.ExpectExec(`UPDATE orders SET status`).
		WithArgs("DECLINED", nil, sqlmock.AnyArg(), int64(1)).
		WillReturnError(fmt.Errorf("db error"))
	accMock.ExpectQuery(`SELECT available_balance`).
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(float64(100)))
	s := &Scheduler{
		DB:               db,
		AccountDB:        accDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing},
		LoanClient:       &mockLoanClient{loans: []*pb_loan.LoanSummary{}},
	}
	order := models.Order{ID: 1, AssetID: 10, IsMargin: true, AccountID: 5, UserType: "CLIENT", RemainingPortions: 1}
	s.executeOrder(order)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── processOrders ─────────────────────────────────────────────────────────────

func TestProcessOrders_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectQuery(`APPROVED`).WillReturnError(fmt.Errorf("db error"))
	s := &Scheduler{DB: db}
	s.processOrders()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProcessOrders_EmptyOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectQuery(`APPROVED`).WillReturnRows(sqlmock.NewRows([]string{
		"id", "user_id", "user_type", "asset_id", "order_type", "quantity", "contract_size",
		"price_per_unit", "limit_value", "stop_value", "direction", "status", "approved_by",
		"is_done", "last_modification", "remaining_portions", "after_hours", "is_aon", "is_margin", "account_id", "fund_id",
	}))
	s := &Scheduler{DB: db}
	s.processOrders()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProcessOrders_AlreadyInProgress(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectQuery(`APPROVED`).WillReturnRows(sqlmock.NewRows([]string{
		"id", "user_id", "user_type", "asset_id", "order_type", "quantity", "contract_size",
		"price_per_unit", "limit_value", "stop_value", "direction", "status", "approved_by",
		"is_done", "last_modification", "remaining_portions", "after_hours", "is_aon", "is_margin", "account_id", "fund_id",
	}).AddRow(
		int64(42), int64(1), "EMPLOYEE", int64(10), "MARKET", int32(1), int32(1),
		100.0, nil, nil, "BUY", "APPROVED", nil,
		false, time.Now(), int32(1), false, false, false, int64(1), int64(0),
	))
	s := &Scheduler{DB: db}
	s.inProgress.Store(int64(42), true)
	s.processOrders()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProcessOrders_LaunchesGoroutine(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectQuery(`APPROVED`).WillReturnRows(sqlmock.NewRows([]string{
		"id", "user_id", "user_type", "asset_id", "order_type", "quantity", "contract_size",
		"price_per_unit", "limit_value", "stop_value", "direction", "status", "approved_by",
		"is_done", "last_modification", "remaining_portions", "after_hours", "is_aon", "is_margin", "account_id", "fund_id",
	}).AddRow(
		int64(7), int64(1), "EMPLOYEE", int64(10), "MARKET", int32(1), int32(1),
		100.0, nil, nil, "BUY", "APPROVED", nil,
		false, time.Now(), int32(1), false, false, false, int64(1), int64(0),
	))
	s := &Scheduler{
		DB:               db,
		SecuritiesClient: &mockSecuritiesClient{err: fmt.Errorf("listing error")},
	}
	s.processOrders()
	time.Sleep(20 * time.Millisecond)
	_, loaded := s.inProgress.Load(int64(7))
	assert.False(t, loaded)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── FillInterval ──────────────────────────────────────────────────────────────

func TestFillInterval_SmallVolume(t *testing.T) {
	// volume < remainingQuantity → fillsPerDay = 0 → clamped to 1
	d := FillInterval(1, 5, false)
	assert.GreaterOrEqual(t, d, time.Duration(0))
}

func TestFillInterval_ZeroVolume(t *testing.T) {
	d := FillInterval(0, 5, false)
	assert.Equal(t, 5*time.Second, d)
}

func TestFillInterval_AfterHours(t *testing.T) {
	d := FillInterval(100, 1, true)
	assert.GreaterOrEqual(t, d, 30*time.Minute)
}

// ── mock fund and portfolio clients ──────────────────────────────────────────

type mockFundClient struct {
	updateHoldingErr error
	checkPendingErr  error
}

func (m *mockFundClient) Ping(ctx context.Context, in *pb_fund.PingRequest, opts ...grpc.CallOption) (*pb_fund.PingResponse, error) {
	return nil, nil
}
func (m *mockFundClient) CreateFund(ctx context.Context, in *pb_fund.CreateFundRequest, opts ...grpc.CallOption) (*pb_fund.FundResponse, error) {
	return nil, nil
}
func (m *mockFundClient) ListFunds(ctx context.Context, in *pb_fund.ListFundsRequest, opts ...grpc.CallOption) (*pb_fund.ListFundsResponse, error) {
	return nil, nil
}
func (m *mockFundClient) GetFund(ctx context.Context, in *pb_fund.GetFundRequest, opts ...grpc.CallOption) (*pb_fund.FundResponse, error) {
	return nil, nil
}
func (m *mockFundClient) UpdateFund(ctx context.Context, in *pb_fund.UpdateFundRequest, opts ...grpc.CallOption) (*pb_fund.FundResponse, error) {
	return nil, nil
}
func (m *mockFundClient) DeleteFund(ctx context.Context, in *pb_fund.DeleteFundRequest, opts ...grpc.CallOption) (*pb_fund.DeleteFundResponse, error) {
	return nil, nil
}
func (m *mockFundClient) InvestFund(ctx context.Context, in *pb_fund.InvestFundRequest, opts ...grpc.CallOption) (*pb_fund.FundResponse, error) {
	return nil, nil
}
func (m *mockFundClient) WithdrawFund(ctx context.Context, in *pb_fund.WithdrawFundRequest, opts ...grpc.CallOption) (*pb_fund.WithdrawFundResponse, error) {
	return nil, nil
}
func (m *mockFundClient) CheckPendingWithdrawals(ctx context.Context, in *pb_fund.CheckPendingWithdrawalsRequest, opts ...grpc.CallOption) (*pb_fund.CheckPendingWithdrawalsResponse, error) {
	return nil, m.checkPendingErr
}
func (m *mockFundClient) GetBankPositions(ctx context.Context, in *pb_fund.GetBankPositionsRequest, opts ...grpc.CallOption) (*pb_fund.GetBankPositionsResponse, error) {
	return nil, nil
}
func (m *mockFundClient) GetMyPositions(ctx context.Context, in *pb_fund.GetMyPositionsRequest, opts ...grpc.CallOption) (*pb_fund.GetMyPositionsResponse, error) {
	return nil, nil
}
func (m *mockFundClient) TransferFundsByManager(ctx context.Context, in *pb_fund.TransferFundsByManagerRequest, opts ...grpc.CallOption) (*pb_fund.TransferFundsByManagerResponse, error) {
	return nil, nil
}
func (m *mockFundClient) ValidateFundAccount(ctx context.Context, in *pb_fund.ValidateFundAccountRequest, opts ...grpc.CallOption) (*pb_fund.ValidateFundAccountResponse, error) {
	return nil, nil
}
func (m *mockFundClient) UpdateFundHolding(ctx context.Context, in *pb_fund.UpdateFundHoldingRequest, opts ...grpc.CallOption) (*pb_fund.UpdateFundHoldingResponse, error) {
	return nil, m.updateHoldingErr
}
func (m *mockFundClient) GetFundPortfolio(ctx context.Context, in *pb_fund.GetFundPortfolioRequest, opts ...grpc.CallOption) (*pb_fund.GetFundPortfolioResponse, error) {
	return nil, nil
}
func (m *mockFundClient) GetFundPerformanceHistory(ctx context.Context, in *pb_fund.GetFundPerformanceRequest, opts ...grpc.CallOption) (*pb_fund.GetFundPerformanceResponse, error) {
	return nil, nil
}
func (m *mockFundClient) GetAveragePerformance(ctx context.Context, in *pb_fund.GetAveragePerformanceRequest, opts ...grpc.CallOption) (*pb_fund.GetAveragePerformanceResponse, error) {
	return nil, nil
}
func (m *mockFundClient) ProcessFundDividend(ctx context.Context, in *pb_fund.ProcessFundDividendRequest, opts ...grpc.CallOption) (*pb_fund.ProcessFundDividendResponse, error) {
	return nil, nil
}

type mockPortfolioClient struct {
	updateHoldingErr error
}

func (m *mockPortfolioClient) UpdateHolding(ctx context.Context, in *pb_portfolio.UpdateHoldingRequest, opts ...grpc.CallOption) (*pb_portfolio.UpdateHoldingResponse, error) {
	return nil, m.updateHoldingErr
}
func (m *mockPortfolioClient) GetPortfolio(ctx context.Context, in *pb_portfolio.GetPortfolioRequest, opts ...grpc.CallOption) (*pb_portfolio.GetPortfolioResponse, error) {
	return nil, nil
}
func (m *mockPortfolioClient) GetProfit(ctx context.Context, in *pb_portfolio.GetProfitRequest, opts ...grpc.CallOption) (*pb_portfolio.GetProfitResponse, error) {
	return nil, nil
}
func (m *mockPortfolioClient) SetPublicAmount(ctx context.Context, in *pb_portfolio.SetPublicAmountRequest, opts ...grpc.CallOption) (*pb_portfolio.SetPublicAmountResponse, error) {
	return nil, nil
}
func (m *mockPortfolioClient) SetPublicMode(ctx context.Context, in *pb_portfolio.SetPublicModeRequest, opts ...grpc.CallOption) (*pb_portfolio.SetPublicModeResponse, error) {
	return nil, nil
}
func (m *mockPortfolioClient) GetMyTax(ctx context.Context, in *pb_portfolio.GetMyTaxRequest, opts ...grpc.CallOption) (*pb_portfolio.GetMyTaxResponse, error) {
	return nil, nil
}
func (m *mockPortfolioClient) GetTaxList(ctx context.Context, in *pb_portfolio.GetTaxListRequest, opts ...grpc.CallOption) (*pb_portfolio.GetTaxListResponse, error) {
	return nil, nil
}
func (m *mockPortfolioClient) CollectTax(ctx context.Context, in *pb_portfolio.CollectTaxRequest, opts ...grpc.CallOption) (*pb_portfolio.CollectTaxResponse, error) {
	return nil, nil
}
func (m *mockPortfolioClient) CollectTaxForUser(ctx context.Context, in *pb_portfolio.CollectTaxForUserRequest, opts ...grpc.CallOption) (*pb_portfolio.CollectTaxForUserResponse, error) {
	return nil, nil
}
func (m *mockPortfolioClient) CreateWatchlist(ctx context.Context, in *pb_portfolio.CreateWatchlistRequest, opts ...grpc.CallOption) (*pb_portfolio.WatchlistResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockPortfolioClient) ListWatchlists(ctx context.Context, in *pb_portfolio.ListWatchlistsRequest, opts ...grpc.CallOption) (*pb_portfolio.ListWatchlistsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockPortfolioClient) DeleteWatchlist(ctx context.Context, in *pb_portfolio.DeleteWatchlistRequest, opts ...grpc.CallOption) (*pb_portfolio.DeleteWatchlistResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockPortfolioClient) AddWatchlistItem(ctx context.Context, in *pb_portfolio.AddWatchlistItemRequest, opts ...grpc.CallOption) (*pb_portfolio.WatchlistItemResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockPortfolioClient) RemoveWatchlistItem(ctx context.Context, in *pb_portfolio.RemoveWatchlistItemRequest, opts ...grpc.CallOption) (*pb_portfolio.RemoveWatchlistItemResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockPortfolioClient) GetWatchlistItems(ctx context.Context, in *pb_portfolio.GetWatchlistItemsRequest, opts ...grpc.CallOption) (*pb_portfolio.GetWatchlistItemsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockPortfolioClient) CreateDividendPayout(ctx context.Context, in *pb_portfolio.CreateDividendPayoutRequest, opts ...grpc.CallOption) (*pb_portfolio.DividendPayoutResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockPortfolioClient) GetDividendHistory(ctx context.Context, in *pb_portfolio.GetDividendHistoryRequest, opts ...grpc.CallOption) (*pb_portfolio.GetDividendHistoryResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockPortfolioClient) GetHoldersByListing(ctx context.Context, in *pb_portfolio.GetHoldersByListingRequest, opts ...grpc.CallOption) (*pb_portfolio.GetHoldersByListingResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// ── executeOrder: full happy-path loop (1 portion, no fund/portfolio) ────────

func TestExecuteOrder_FullLoop_Happy(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, Volume: 1000, Type: "STOCK"}

	// listingCurrency
	secMock.ExpectQuery(`SELECT e\.currency`).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))

	// settleAccountAndCommission (BUY, USD→USD, totalPrice=100, commission=7)
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(-107.0, int64(1), 107.0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(7.0, int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// InsertPortion
	dbMock.ExpectExec(`INSERT INTO order_portions`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// UpdateRemainingPortions (remaining=0)
	dbMock.ExpectExec(`UPDATE orders SET remaining_portions`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// SetOrderDone
	dbMock.ExpectExec(`UPDATE orders SET is_done`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{
		DB:               db,
		AccountDB:        accDB,
		ExchangeDB:       exchDB,
		SecuritiesDB:     secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing},
		ExchangeClient:   &mockExchangeClient{},
	}
	order := models.Order{
		ID: 1, AssetID: 10, IsMargin: false, IsAON: false,
		Direction: "BUY", OrderType: "MARKET",
		ContractSize: 1, Quantity: 1, RemainingPortions: 1,
		AccountID: 1, UserType: "EMPLOYEE", FundID: 0,
	}
	s.executeOrder(order)
	assert.NoError(t, dbMock.ExpectationsWereMet())
	assert.NoError(t, accMock.ExpectationsWereMet())
	assert.NoError(t, exchMock.ExpectationsWereMet())
	assert.NoError(t, secMock.ExpectationsWereMet())
}

func TestExecuteOrder_FullLoop_WithPortfolioClient(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, Volume: 1000, Type: "STOCK"}

	secMock.ExpectQuery(`SELECT e\.currency`).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(-107.0, int64(1), 107.0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(7.0, int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec(`INSERT INTO order_portions`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectExec(`UPDATE orders SET remaining_portions`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec(`UPDATE orders SET is_done`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{
		DB:               db,
		AccountDB:        accDB,
		ExchangeDB:       exchDB,
		SecuritiesDB:     secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing},
		ExchangeClient:   &mockExchangeClient{},
		PortfolioClient:  &mockPortfolioClient{},
	}
	order := models.Order{
		ID: 1, AssetID: 10, IsMargin: false, IsAON: false,
		Direction: "BUY", OrderType: "MARKET",
		ContractSize: 1, Quantity: 1, RemainingPortions: 1,
		AccountID: 1, UserType: "EMPLOYEE", FundID: 0,
	}
	s.executeOrder(order)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestExecuteOrder_FullLoop_WithFundClient_Sell(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, Volume: 1000, Type: "STOCK"}

	secMock.ExpectQuery(`SELECT e\.currency`).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	// SELL uses bid=99: totalPrice=99, commission=7, credit=99-7=92
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(92.0, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(7.0, int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec(`INSERT INTO order_portions`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectExec(`UPDATE orders SET remaining_portions`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec(`UPDATE orders SET is_done`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{
		DB:               db,
		AccountDB:        accDB,
		ExchangeDB:       exchDB,
		SecuritiesDB:     secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing},
		ExchangeClient:   &mockExchangeClient{},
		FundClient:       &mockFundClient{},
	}
	order := models.Order{
		ID: 1, AssetID: 10, IsMargin: false, IsAON: false,
		Direction: "SELL", OrderType: "MARKET",
		ContractSize: 1, Quantity: 1, RemainingPortions: 1,
		AccountID: 1, UserType: "EMPLOYEE", FundID: 5,
	}
	s.executeOrder(order)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestExecuteOrder_SettlementFails_InLoop(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, Volume: 1000}

	secMock.ExpectQuery(`SELECT e\.currency`).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	// settleAccountAndCommission fails on first DB query
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnError(fmt.Errorf("db error"))
	// UpdateOrderStatus after settle failure
	dbMock.ExpectExec(`UPDATE orders SET status`).
		WithArgs("DECLINED", nil, sqlmock.AnyArg(), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{
		DB:               db,
		AccountDB:        accDB,
		ExchangeDB:       exchDB,
		SecuritiesDB:     secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing},
	}
	order := models.Order{
		ID: 1, AssetID: 10, IsMargin: false, IsAON: false,
		Direction: "BUY", OrderType: "MARKET",
		ContractSize: 1, Quantity: 1, RemainingPortions: 1,
		AccountID: 1, UserType: "EMPLOYEE",
	}
	s.executeOrder(order)
	assert.NoError(t, dbMock.ExpectationsWereMet())
	assert.NoError(t, secMock.ExpectationsWereMet())
}

// helper: BUY MARKET happy-path settle mocks (USD=USD, totalPrice=100, commission=7)
func expectBuySettle(accMock, exchMock sqlmock.Sqlmock) {
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(-107.0, int64(1), 107.0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(7.0, int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func TestExecuteOrder_Loop_LimitStopValuePointers(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, Volume: 1000, Type: "STOCK"}
	secMock.ExpectQuery(`SELECT e\.currency`).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	expectBuySettle(accMock, exchMock)
	dbMock.ExpectExec(`INSERT INTO order_portions`).WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectExec(`UPDATE orders SET remaining_portions`).WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec(`UPDATE orders SET is_done`).WillReturnResult(sqlmock.NewResult(0, 1))

	lv, sv := 200.0, 50.0
	s := &Scheduler{DB: db, AccountDB: accDB, ExchangeDB: exchDB, SecuritiesDB: secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing}, ExchangeClient: &mockExchangeClient{}}
	order := models.Order{
		ID: 1, AssetID: 10, IsMargin: false, IsAON: false,
		Direction: "BUY", OrderType: "MARKET",
		ContractSize: 1, Quantity: 1, RemainingPortions: 1,
		AccountID: 1, UserType: "EMPLOYEE", FundID: 0,
		LimitValue: &lv, StopValue: &sv,
	}
	s.executeOrder(order)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestExecuteOrder_Loop_IsAON(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, Volume: 1000, Type: "STOCK"}
	secMock.ExpectQuery(`SELECT e\.currency`).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	expectBuySettle(accMock, exchMock)
	dbMock.ExpectExec(`INSERT INTO order_portions`).WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectExec(`UPDATE orders SET remaining_portions`).WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec(`UPDATE orders SET is_done`).WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{DB: db, AccountDB: accDB, ExchangeDB: exchDB, SecuritiesDB: secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing}, ExchangeClient: &mockExchangeClient{}}
	order := models.Order{
		ID: 1, AssetID: 10, IsMargin: false, IsAON: true,
		Direction: "BUY", OrderType: "MARKET",
		ContractSize: 1, Quantity: 1, RemainingPortions: 1,
		AccountID: 1, UserType: "EMPLOYEE", FundID: 0,
	}
	s.executeOrder(order)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestExecuteOrder_Loop_FundUpdateError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, Volume: 1000, Type: "STOCK"}
	secMock.ExpectQuery(`SELECT e\.currency`).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	expectBuySettle(accMock, exchMock)
	dbMock.ExpectExec(`INSERT INTO order_portions`).WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectExec(`UPDATE orders SET remaining_portions`).WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec(`UPDATE orders SET is_done`).WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{DB: db, AccountDB: accDB, ExchangeDB: exchDB, SecuritiesDB: secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing}, ExchangeClient: &mockExchangeClient{},
		FundClient: &mockFundClient{updateHoldingErr: fmt.Errorf("fund error")}}
	order := models.Order{
		ID: 1, AssetID: 10, IsMargin: false, IsAON: false,
		Direction: "BUY", OrderType: "MARKET",
		ContractSize: 1, Quantity: 1, RemainingPortions: 1,
		AccountID: 1, UserType: "EMPLOYEE", FundID: 5,
	}
	s.executeOrder(order)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestExecuteOrder_Loop_PortfolioUpdateError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, Volume: 1000, Type: "STOCK"}
	secMock.ExpectQuery(`SELECT e\.currency`).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	expectBuySettle(accMock, exchMock)
	dbMock.ExpectExec(`INSERT INTO order_portions`).WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectExec(`UPDATE orders SET remaining_portions`).WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec(`UPDATE orders SET is_done`).WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{DB: db, AccountDB: accDB, ExchangeDB: exchDB, SecuritiesDB: secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing}, ExchangeClient: &mockExchangeClient{},
		PortfolioClient: &mockPortfolioClient{updateHoldingErr: fmt.Errorf("portfolio error")}}
	order := models.Order{
		ID: 1, AssetID: 10, IsMargin: false, IsAON: false,
		Direction: "BUY", OrderType: "MARKET",
		ContractSize: 1, Quantity: 1, RemainingPortions: 1,
		AccountID: 1, UserType: "EMPLOYEE", FundID: 0,
	}
	s.executeOrder(order)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestExecuteOrder_Loop_UpdateRemainingError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, Volume: 1000, Type: "STOCK"}
	secMock.ExpectQuery(`SELECT e\.currency`).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	expectBuySettle(accMock, exchMock)
	dbMock.ExpectExec(`INSERT INTO order_portions`).WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectExec(`UPDATE orders SET remaining_portions`).WillReturnError(fmt.Errorf("db error"))
	dbMock.ExpectExec(`UPDATE orders SET is_done`).WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{DB: db, AccountDB: accDB, ExchangeDB: exchDB, SecuritiesDB: secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing}, ExchangeClient: &mockExchangeClient{}}
	order := models.Order{
		ID: 1, AssetID: 10, IsMargin: false, IsAON: false,
		Direction: "BUY", OrderType: "MARKET",
		ContractSize: 1, Quantity: 1, RemainingPortions: 1,
		AccountID: 1, UserType: "EMPLOYEE", FundID: 0,
	}
	s.executeOrder(order)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestExecuteOrder_Loop_SetOrderDoneError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, Volume: 1000, Type: "STOCK"}
	secMock.ExpectQuery(`SELECT e\.currency`).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	expectBuySettle(accMock, exchMock)
	dbMock.ExpectExec(`INSERT INTO order_portions`).WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectExec(`UPDATE orders SET remaining_portions`).WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec(`UPDATE orders SET is_done`).WillReturnError(fmt.Errorf("db error"))

	s := &Scheduler{DB: db, AccountDB: accDB, ExchangeDB: exchDB, SecuritiesDB: secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing}, ExchangeClient: &mockExchangeClient{}}
	order := models.Order{
		ID: 1, AssetID: 10, IsMargin: false, IsAON: false,
		Direction: "BUY", OrderType: "MARKET",
		ContractSize: 1, Quantity: 1, RemainingPortions: 1,
		AccountID: 1, UserType: "EMPLOYEE", FundID: 0,
	}
	s.executeOrder(order)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestExecuteOrder_Loop_CheckPendingWithdrawalsError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	accDB, accMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = accDB.Close() }()
	exchDB, exchMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = exchDB.Close() }()
	secDB, secMock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = secDB.Close() }()

	listing := &pb_sec.ListingSummary{Ask: 100.0, Bid: 99.0, Volume: 1000, Type: "STOCK"}
	secMock.ExpectQuery(`SELECT e\.currency`).
		WillReturnRows(sqlmock.NewRows([]string{"currency"}).AddRow("USD"))
	// SELL uses bid=99: totalPrice=99, commission=7, credit=92
	accMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchMock.ExpectQuery(`SELECT code FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("USD"))
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(92.0, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	exchMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	accMock.ExpectQuery(`SELECT id FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	accMock.ExpectExec(`UPDATE accounts`).
		WithArgs(7.0, int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec(`INSERT INTO order_portions`).WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectExec(`UPDATE orders SET remaining_portions`).WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec(`UPDATE orders SET is_done`).WillReturnResult(sqlmock.NewResult(0, 1))

	s := &Scheduler{DB: db, AccountDB: accDB, ExchangeDB: exchDB, SecuritiesDB: secDB,
		SecuritiesClient: &mockSecuritiesClient{listing: listing}, ExchangeClient: &mockExchangeClient{},
		FundClient: &mockFundClient{checkPendingErr: fmt.Errorf("fund error")}}
	order := models.Order{
		ID: 1, AssetID: 10, IsMargin: false, IsAON: false,
		Direction: "SELL", OrderType: "MARKET",
		ContractSize: 1, Quantity: 1, RemainingPortions: 1,
		AccountID: 1, UserType: "EMPLOYEE", FundID: 5,
	}
	s.executeOrder(order)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}
