package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb_account "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	pb_order "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
)

// ---- mock AccountServiceClient ----

type mockAccountClient struct {
	createAccountFn func(context.Context, *pb_account.CreateAccountRequest, ...grpc.CallOption) (*pb_account.CreateAccountResponse, error)
}

func (m *mockAccountClient) CreateAccount(ctx context.Context, in *pb_account.CreateAccountRequest, opts ...grpc.CallOption) (*pb_account.CreateAccountResponse, error) {
	if m.createAccountFn != nil {
		return m.createAccountFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockAccountClient) GetMyAccounts(ctx context.Context, in *pb_account.GetMyAccountsRequest, opts ...grpc.CallOption) (*pb_account.GetMyAccountsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockAccountClient) GetAccount(ctx context.Context, in *pb_account.GetAccountRequest, opts ...grpc.CallOption) (*pb_account.GetAccountResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockAccountClient) RenameAccount(ctx context.Context, in *pb_account.RenameAccountRequest, opts ...grpc.CallOption) (*pb_account.RenameAccountResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockAccountClient) GetAllAccounts(ctx context.Context, in *pb_account.GetAllAccountsRequest, opts ...grpc.CallOption) (*pb_account.GetAllAccountsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockAccountClient) UpdateAccountLimits(ctx context.Context, in *pb_account.UpdateAccountLimitsRequest, opts ...grpc.CallOption) (*pb_account.UpdateAccountLimitsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockAccountClient) DeleteAccount(ctx context.Context, in *pb_account.DeleteAccountRequest, opts ...grpc.CallOption) (*pb_account.DeleteAccountResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockAccountClient) GetBankAccounts(ctx context.Context, in *pb_account.GetBankAccountsRequest, opts ...grpc.CallOption) (*pb_account.GetBankAccountsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// ---- mock OrderServiceClient ----

type mockOrderClient struct {
	createOrderFn func(context.Context, *pb_order.CreateOrderRequest, ...grpc.CallOption) (*pb_order.CreateOrderResponse, error)
}

func (m *mockOrderClient) Ping(ctx context.Context, in *pb_order.PingRequest, opts ...grpc.CallOption) (*pb_order.PingResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockOrderClient) CreateOrder(ctx context.Context, in *pb_order.CreateOrderRequest, opts ...grpc.CallOption) (*pb_order.CreateOrderResponse, error) {
	if m.createOrderFn != nil {
		return m.createOrderFn(ctx, in, opts...)
	}
	return &pb_order.CreateOrderResponse{}, nil
}
func (m *mockOrderClient) ListOrders(ctx context.Context, in *pb_order.ListOrdersRequest, opts ...grpc.CallOption) (*pb_order.ListOrdersResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockOrderClient) GetOrderById(ctx context.Context, in *pb_order.GetOrderByIdRequest, opts ...grpc.CallOption) (*pb_order.GetOrderByIdResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockOrderClient) ApproveOrder(ctx context.Context, in *pb_order.ApproveOrderRequest, opts ...grpc.CallOption) (*pb_order.ApproveOrderResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockOrderClient) DeclineOrder(ctx context.Context, in *pb_order.DeclineOrderRequest, opts ...grpc.CallOption) (*pb_order.DeclineOrderResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockOrderClient) CancelOrder(ctx context.Context, in *pb_order.CancelOrderRequest, opts ...grpc.CallOption) (*pb_order.CancelOrderResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockOrderClient) CancelOrderPortions(ctx context.Context, in *pb_order.CancelOrderPortionsRequest, opts ...grpc.CallOption) (*pb_order.CancelOrderPortionsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockOrderClient) GetActuaryProfits(ctx context.Context, in *pb_order.GetActuaryProfitsRequest, opts ...grpc.CallOption) (*pb_order.GetActuaryProfitsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// ---- helpers ----

// newFundServer creates a FundServer with mocked DBs and no OrderClient.
func newFundServer(t *testing.T, acctClient pb_account.AccountServiceClient) (*FundServer, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	return newFundServerFull(t, acctClient, nil)
}

// newFundServerFull creates a FundServer with mocked DBs and an optional OrderClient.
func newFundServerFull(t *testing.T, acctClient pb_account.AccountServiceClient, orderClient pb_order.OrderServiceClient) (*FundServer, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	fundDB, fundMock, err := sqlmock.New()
	require.NoError(t, err)
	accountDB, accountDBMock, err := sqlmock.New()
	require.NoError(t, err)
	employeeDB, empMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = fundDB.Close()
		_ = accountDB.Close()
		_ = employeeDB.Close()
	})
	return &FundServer{
		DB:            fundDB,
		AccountDB:     accountDB,
		EmployeeDB:    employeeDB,
		AccountClient: acctClient,
		OrderClient:   orderClient,
	}, fundMock, accountDBMock, empMock
}

// fundDetailColumns returns the columns scanned by fetchFundByID.
func fundDetailColumns() []string {
	return []string{"id", "name", "description", "minimum_contribution", "manager_id",
		"liquid_assets", "account_id", "created_at", "active"}
}

// addFetchFundRows sets up mock expectations for fetchFundByID with includeAccountNumber=true.
// It expects: SELECT fund, SUM profit, employee name, account number.
func addFetchFundRows(fundMock, accountDBMock, empMock sqlmock.Sqlmock,
	id int64, name string, accountID int64, managerID int64) {
	now := time.Now()
	fundMock.ExpectQuery("SELECT id, name, description").
		WillReturnRows(sqlmock.NewRows(fundDetailColumns()).
			AddRow(id, name,
				sql.NullString{String: "A test fund", Valid: true},
				float64(1000), managerID,
				float64(500000),
				sql.NullInt64{Int64: accountID, Valid: true},
				now, true))
	// SUM of invested
	fundMock.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(float64(0)))
	// manager name
	empMock.ExpectQuery("SELECT first_name").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Jane Manager"))
	// account number
	accountDBMock.ExpectQuery("SELECT account_number").
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("123-456-78"))
}

// addFetchFundRowsNoAccount sets up mock expectations for fetchFundByID with includeAccountNumber=false
// (used in ListFunds).
func addFetchFundRowsNoAccount(fundMock, empMock sqlmock.Sqlmock,
	id int64, name string, accountID int64, managerID int64) {
	now := time.Now()
	fundMock.ExpectQuery("SELECT id, name, description").
		WillReturnRows(sqlmock.NewRows(fundDetailColumns()).
			AddRow(id, name,
				sql.NullString{String: "A test fund", Valid: true},
				float64(1000), managerID,
				float64(500000),
				sql.NullInt64{Int64: accountID, Valid: true},
				now, true))
	// SUM of invested
	fundMock.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(float64(0)))
	// manager name
	empMock.ExpectQuery("SELECT first_name").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Jane Manager"))
}

// ---- TestCreateFund_Happy ----

func TestCreateFund_Happy(t *testing.T) {
	acct := &mockAccountClient{
		createAccountFn: func(_ context.Context, _ *pb_account.CreateAccountRequest, _ ...grpc.CallOption) (*pb_account.CreateAccountResponse, error) {
			return &pb_account.CreateAccountResponse{
				Account: &pb_account.AccountResponse{
					Id:            100,
					AccountNumber: "123-456-78",
				},
			}, nil
		},
	}
	s, fundMock, accountDBMock, empMock := newFundServer(t, acct)

	// permissions check for manager
	empMock.ExpectQuery("SELECT array_to_string").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow("SUPERVISOR"))

	// INSERT fund returns id=1
	fundMock.ExpectQuery("INSERT INTO investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	// fetchFundByID with includeAccountNumber=true
	addFetchFundRows(fundMock, accountDBMock, empMock, 1, "Test Fund", 100, 5)

	resp, err := s.CreateFund(context.Background(), &pb.CreateFundRequest{
		Name:                "Test Fund",
		Description:         "A test fund",
		MinimumContribution: 1000.0,
		ManagerId:           5,
		CreatedById:         1,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Id)
	assert.Equal(t, "Test Fund", resp.Name)
	assert.Equal(t, int64(100), resp.AccountId)
}

// ---- TestCreateFund_MissingName ----

func TestCreateFund_MissingName(t *testing.T) {
	acct := &mockAccountClient{}
	s, _, _, _ := newFundServer(t, acct)

	_, err := s.CreateFund(context.Background(), &pb.CreateFundRequest{
		Name:      "",
		ManagerId: 5,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// ---- TestCreateFund_DuplicateName ----

func TestCreateFund_DuplicateName(t *testing.T) {
	acct := &mockAccountClient{
		createAccountFn: func(_ context.Context, _ *pb_account.CreateAccountRequest, _ ...grpc.CallOption) (*pb_account.CreateAccountResponse, error) {
			return &pb_account.CreateAccountResponse{
				Account: &pb_account.AccountResponse{Id: 101, AccountNumber: "222-333-44"},
			}, nil
		},
	}
	s, fundMock, _, empMock := newFundServer(t, acct)

	// permissions check for manager
	empMock.ExpectQuery("SELECT array_to_string").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow("SUPERVISOR"))

	// INSERT returns unique violation
	fundMock.ExpectQuery("INSERT INTO investment_funds").
		WillReturnError(fmt.Errorf("pq: duplicate key value violates unique constraint: 23505"))

	_, err := s.CreateFund(context.Background(), &pb.CreateFundRequest{
		Name:                "Existing Fund",
		MinimumContribution: 500,
		ManagerId:           5,
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

// ---- TestListFunds_Empty ----

func TestListFunds_Empty(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery("SELECT id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	resp, err := s.ListFunds(context.Background(), &pb.ListFundsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Funds, 0)
}

// ---- TestListFunds_WithManagerFilter ----

func TestListFunds_WithManagerFilter(t *testing.T) {
	s, fundMock, _, empMock := newFundServer(t, &mockAccountClient{})

	// Returns 1 fund ID
	fundMock.ExpectQuery("SELECT id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	// fetchFundByID with includeAccountNumber=false (ListFunds)
	addFetchFundRowsNoAccount(fundMock, empMock, 1, "Manager Fund", 100, 3)

	resp, err := s.ListFunds(context.Background(), &pb.ListFundsRequest{ManagerIdFilter: 3})
	require.NoError(t, err)
	assert.Len(t, resp.Funds, 1)
}

// ---- TestDeleteFund_HasPositions ----

func TestDeleteFund_HasPositions(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	// COUNT returns 1 (active position exists)
	fundMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	_, err := s.DeleteFund(context.Background(), &pb.DeleteFundRequest{Id: 1})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

// ---- TestDeleteFund_NotFound ----

func TestDeleteFund_NotFound(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	// COUNT returns 0 (no active positions)
	fundMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

	// UPDATE returns 0 rows affected
	fundMock.ExpectExec("UPDATE investment_funds SET active = false").
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := s.DeleteFund(context.Background(), &pb.DeleteFundRequest{Id: 99})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ---- TestDeleteFund_Happy ----

func TestDeleteFund_Happy(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	// COUNT returns 0
	fundMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

	// UPDATE returns 1 row affected
	fundMock.ExpectExec("UPDATE investment_funds SET active = false").
		WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := s.DeleteFund(context.Background(), &pb.DeleteFundRequest{Id: 1})
	require.NoError(t, err)
}

// ---- additional: TestListFunds_Happy (two funds) ----

func TestListFunds_Happy(t *testing.T) {
	s, fundMock, _, empMock := newFundServer(t, &mockAccountClient{})

	// Two fund IDs returned
	fundMock.ExpectQuery("SELECT id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)).AddRow(int64(2)))

	// fetchFundByID for fund 1
	addFetchFundRowsNoAccount(fundMock, empMock, 1, "Fund A", 100, 5)
	// fetchFundByID for fund 2
	addFetchFundRowsNoAccount(fundMock, empMock, 2, "Fund B", 101, 5)

	resp, err := s.ListFunds(context.Background(), &pb.ListFundsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Funds, 2)
}

// ---- TestGetFund_NotFound ----

func TestGetFund_NotFound(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery("SELECT id, name, description").
		WillReturnRows(sqlmock.NewRows(fundDetailColumns()))

	_, err := s.GetFund(context.Background(), &pb.GetFundRequest{Id: 999})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ---- TestGetFund_Happy ----

func TestGetFund_Happy(t *testing.T) {
	s, fundMock, accountDBMock, empMock := newFundServer(t, &mockAccountClient{})

	addFetchFundRows(fundMock, accountDBMock, empMock, 1, "Happy Fund", 100, 5)

	resp, err := s.GetFund(context.Background(), &pb.GetFundRequest{Id: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Id)
	assert.Equal(t, "Happy Fund", resp.Name)
	assert.Greater(t, resp.FundValue, float64(0))
}

// ---- TestUpdateFund_NotFound ----

func TestUpdateFund_NotFound(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	// UPDATE executes OK
	fundMock.ExpectExec("UPDATE investment_funds").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// fetchFundByID returns no rows
	fundMock.ExpectQuery("SELECT id, name, description").
		WillReturnRows(sqlmock.NewRows(fundDetailColumns()))

	_, err := s.UpdateFund(context.Background(), &pb.UpdateFundRequest{
		Id:        999,
		Name:      "No Fund",
		ManagerId: 5,
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ---- TestUpdateFund_Happy ----

func TestUpdateFund_Happy(t *testing.T) {
	s, fundMock, accountDBMock, empMock := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectExec("UPDATE investment_funds").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// fetchFundByID with includeAccountNumber=true
	addFetchFundRows(fundMock, accountDBMock, empMock, 1, "Updated Fund", 100, 5)

	resp, err := s.UpdateFund(context.Background(), &pb.UpdateFundRequest{
		Id:                  1,
		Name:                "Updated Fund",
		Description:         "new desc",
		MinimumContribution: 2000.0,
		ManagerId:           5,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Fund", resp.Name)
}

// ---- TestCreateFund_AccountClientError ----

func TestCreateFund_AccountClientError(t *testing.T) {
	acct := &mockAccountClient{
		createAccountFn: func(_ context.Context, _ *pb_account.CreateAccountRequest, _ ...grpc.CallOption) (*pb_account.CreateAccountResponse, error) {
			return nil, fmt.Errorf("account service unavailable")
		},
	}
	s, _, _, empMock := newFundServer(t, acct)

	// permissions check for manager
	empMock.ExpectQuery("SELECT array_to_string").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow("SUPERVISOR"))

	_, err := s.CreateFund(context.Background(), &pb.CreateFundRequest{
		Name:                "New Fund",
		MinimumContribution: 1000.0,
		ManagerId:           5,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- TestPing ----

func TestPing_FundService(t *testing.T) {
	s, _, _, _ := newFundServer(t, &mockAccountClient{})
	resp, err := s.Ping(context.Background(), &pb.PingRequest{})
	require.NoError(t, err)
	assert.Equal(t, "fund-service ok", resp.Message)
}

// ---- TestCreateFund_MissingManagerId ----

func TestCreateFund_MissingManagerId(t *testing.T) {
	s, _, _, _ := newFundServer(t, &mockAccountClient{})
	_, err := s.CreateFund(context.Background(), &pb.CreateFundRequest{
		Name:      "Fund Without Manager",
		ManagerId: 0,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// ---- TestCreateFund_DBError (non-unique internal error) ----

func TestCreateFund_DBError(t *testing.T) {
	acct := &mockAccountClient{
		createAccountFn: func(_ context.Context, _ *pb_account.CreateAccountRequest, _ ...grpc.CallOption) (*pb_account.CreateAccountResponse, error) {
			return &pb_account.CreateAccountResponse{
				Account: &pb_account.AccountResponse{Id: 200, AccountNumber: "999-000-01"},
			}, nil
		},
	}
	s, fundMock, _, empMock := newFundServer(t, acct)

	// permissions check for manager
	empMock.ExpectQuery("SELECT array_to_string").
		WillReturnRows(sqlmock.NewRows([]string{"permissions"}).AddRow("SUPERVISOR"))

	fundMock.ExpectQuery("INSERT INTO investment_funds").
		WillReturnError(fmt.Errorf("connection reset by peer"))

	_, err := s.CreateFund(context.Background(), &pb.CreateFundRequest{
		Name:                "Error Fund",
		MinimumContribution: 1000.0,
		ManagerId:           5,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- TestListFunds_DBError ----

func TestListFunds_DBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT id FROM investment_funds").
		WillReturnError(sql.ErrConnDone)

	_, err := s.ListFunds(context.Background(), &pb.ListFundsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- TestUpdateFund_IdZero ----

func TestUpdateFund_IdZero(t *testing.T) {
	s, _, _, _ := newFundServer(t, &mockAccountClient{})
	_, err := s.UpdateFund(context.Background(), &pb.UpdateFundRequest{Id: 0, Name: "X", ManagerId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// ---- TestUpdateFund_DBError ----

func TestUpdateFund_DBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectExec("UPDATE investment_funds").
		WillReturnError(fmt.Errorf("connection reset by peer"))

	_, err := s.UpdateFund(context.Background(), &pb.UpdateFundRequest{
		Id:        1,
		Name:      "Some Fund",
		ManagerId: 5,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- TestUpdateFund_DuplicateName ----

func TestUpdateFund_DuplicateName(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectExec("UPDATE investment_funds").
		WillReturnError(fmt.Errorf("pq: duplicate key value violates unique constraint: 23505"))

	_, err := s.UpdateFund(context.Background(), &pb.UpdateFundRequest{
		Id:        2,
		Name:      "Existing Fund",
		ManagerId: 5,
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

// ---- TestDeleteFund_IdZero ----

func TestDeleteFund_IdZero(t *testing.T) {
	s, _, _, _ := newFundServer(t, &mockAccountClient{})
	_, err := s.DeleteFund(context.Background(), &pb.DeleteFundRequest{Id: 0})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// ---- TestDeleteFund_CountDBError ----

func TestDeleteFund_CountDBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT COUNT").
		WillReturnError(sql.ErrConnDone)

	_, err := s.DeleteFund(context.Background(), &pb.DeleteFundRequest{Id: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetBankPositions ──────────────────────────────────────────────────────────

func TestGetBankPositions_Empty(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery(`SELECT cfp\.fund_id`).
		WillReturnRows(sqlmock.NewRows([]string{"fund_id", "bank_invested", "name", "fund_value", "manager_id", "total_all_invested"}))

	resp, err := s.GetBankPositions(context.Background(), &pb.GetBankPositionsRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Positions)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestGetBankPositions_Happy(t *testing.T) {
	// Bank invested 1000, total all invested 4000, fund value 8000.
	// bankShareRSD = (1000/4000)*8000 = 2000
	// bankSharePercent = (2000/8000)*100 = 25%
	// profitRSD = 2000 - 1000 = 1000
	s, fundMock, _, empMock := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery(`SELECT cfp\.fund_id`).
		WillReturnRows(sqlmock.NewRows([]string{"fund_id", "bank_invested", "name", "fund_value", "manager_id", "total_all_invested"}).
			AddRow(int64(1), 1000.0, "RAF Growth Fund", 8000.0, int64(5), 4000.0))
	empMock.ExpectQuery(`SELECT first_name`).
		WillReturnRows(sqlmock.NewRows([]string{"manager_name"}).AddRow("Ana Jovanovic"))

	resp, err := s.GetBankPositions(context.Background(), &pb.GetBankPositionsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Positions, 1)
	pos := resp.Positions[0]
	assert.Equal(t, int64(1), pos.FundId)
	assert.Equal(t, "RAF Growth Fund", pos.FundName)
	assert.Equal(t, "Ana Jovanovic", pos.ManagerName)
	assert.InDelta(t, 25.0, pos.BankSharePercent, 0.01)
	assert.InDelta(t, 2000.0, pos.BankShareRsd, 0.01)
	assert.InDelta(t, 1000.0, pos.ProfitRsd, 0.01)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestGetBankPositions_ZeroFundValue(t *testing.T) {
	// Fund has no value yet — bankSharePercent should be 0, not divide-by-zero.
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery(`SELECT cfp\.fund_id`).
		WillReturnRows(sqlmock.NewRows([]string{"fund_id", "bank_invested", "name", "fund_value", "manager_id", "total_all_invested"}).
			AddRow(int64(2), 500.0, "Empty Fund", 0.0, int64(0), 500.0))

	resp, err := s.GetBankPositions(context.Background(), &pb.GetBankPositionsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Positions, 1)
	assert.InDelta(t, 0.0, resp.Positions[0].BankSharePercent, 0.001)
	assert.InDelta(t, 0.0, resp.Positions[0].BankShareRsd, 0.001)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestGetBankPositions_DBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery(`SELECT cfp\.fund_id`).WillReturnError(sql.ErrConnDone)

	_, err := s.GetBankPositions(context.Background(), &pb.GetBankPositionsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── ValidateFundAccount ───────────────────────────────────────────────────────

func TestValidateFundAccount_Happy_Liquid(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery("SELECT account_id, manager_id, liquid_assets").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "manager_id", "liquid_assets"}).
			AddRow(int64(42), int64(5), float64(100000)))

	resp, err := s.ValidateFundAccount(context.Background(), &pb.ValidateFundAccountRequest{
		FundId: 1, ManagerId: 5, RequiredAmount: 50000,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.AccountId)
	assert.True(t, resp.IsLiquid)
	assert.InDelta(t, 100000.0, resp.LiquidAssets, 0.01)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestValidateFundAccount_InsufficientLiquidity(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery("SELECT account_id, manager_id, liquid_assets").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "manager_id", "liquid_assets"}).
			AddRow(int64(42), int64(5), float64(10000)))

	resp, err := s.ValidateFundAccount(context.Background(), &pb.ValidateFundAccountRequest{
		FundId: 1, ManagerId: 5, RequiredAmount: 50000,
	})
	require.NoError(t, err)
	assert.False(t, resp.IsLiquid)
	assert.InDelta(t, 10000.0, resp.LiquidAssets, 0.01)
}

func TestValidateFundAccount_WrongManager(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery("SELECT account_id, manager_id, liquid_assets").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "manager_id", "liquid_assets"}).
			AddRow(int64(42), int64(99), float64(100000)))

	_, err := s.ValidateFundAccount(context.Background(), &pb.ValidateFundAccountRequest{
		FundId: 1, ManagerId: 5, RequiredAmount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestValidateFundAccount_NotFound(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery("SELECT account_id, manager_id, liquid_assets").
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	_, err := s.ValidateFundAccount(context.Background(), &pb.ValidateFundAccountRequest{
		FundId: 999, ManagerId: 5, RequiredAmount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ── UpdateFundHolding ─────────────────────────────────────────────────────────

func TestUpdateFundHolding_Buy(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WithArgs(500.0, int64(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO fund_portfolio_positions").
		WithArgs(int64(1), int64(10), int64(5), 100.0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectCommit()

	_, err := s.UpdateFundHolding(context.Background(), &pb.UpdateFundHoldingRequest{
		FundId: 1, ListingId: 10, Quantity: 5, Price: 100.0, Direction: "BUY",
	})
	require.NoError(t, err)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestUpdateFundHolding_Sell(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets \\+").
		WithArgs(300.0, int64(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE fund_portfolio_positions SET quantity = quantity -").
		WithArgs(int64(3), int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("DELETE FROM fund_portfolio_positions").
		WithArgs(int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	fundMock.ExpectCommit()

	_, err := s.UpdateFundHolding(context.Background(), &pb.UpdateFundHoldingRequest{
		FundId: 1, ListingId: 10, Quantity: 3, Price: 100.0, Direction: "SELL",
	})
	require.NoError(t, err)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestTransferFundsByManager_Happy(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectExec("UPDATE investment_funds SET manager_id").
		WithArgs(int64(99), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 2))

	resp, err := s.TransferFundsByManager(context.Background(), &pb.TransferFundsByManagerRequest{
		OldManagerId: 7,
		NewManagerId: 99,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.FundsTransferred)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestTransferFundsByManager_NoFunds(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectExec("UPDATE investment_funds SET manager_id").
		WithArgs(int64(99), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	resp, err := s.TransferFundsByManager(context.Background(), &pb.TransferFundsByManagerRequest{
		OldManagerId: 7,
		NewManagerId: 99,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.FundsTransferred)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestGetMyPositions_Empty(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery("SELECT cfp.fund_id").
		WithArgs(int64(42), "CLIENT").
		WillReturnRows(sqlmock.NewRows([]string{
			"fund_id", "total_invested_amount", "name", "description",
			"fund_value", "minimum_contribution", "total_all_invested",
		}))

	resp, err := s.GetMyPositions(context.Background(), &pb.GetMyPositionsRequest{
		ClientId: 42, ClientType: "CLIENT",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Positions)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestGetMyPositions_Happy(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	// client invested 10000, total all invested is also 10000 (sole investor), fund value 50000
	fundMock.ExpectQuery("SELECT cfp.fund_id").
		WithArgs(int64(1), "CLIENT").
		WillReturnRows(sqlmock.NewRows([]string{
			"fund_id", "total_invested_amount", "name", "description",
			"fund_value", "minimum_contribution", "total_all_invested",
		}).AddRow(int64(7), float64(10000), "RAF Growth Fund", "A growth fund", float64(50000), float64(1000), float64(10000)))

	resp, err := s.GetMyPositions(context.Background(), &pb.GetMyPositionsRequest{
		ClientId: 1, ClientType: "CLIENT",
	})
	require.NoError(t, err)
	require.Len(t, resp.Positions, 1)

	pos := resp.Positions[0]
	assert.Equal(t, int64(7), pos.FundId)
	assert.Equal(t, "RAF Growth Fund", pos.FundName)
	assert.Equal(t, float64(10000), pos.TotalInvestedAmount)
	// currentPositionValue = (10000/10000) * 50000 = 50000
	assert.InDelta(t, 50000.0, pos.CurrentPositionValue, 0.01)
	// fundPercentage = (50000/50000) * 100 = 100
	assert.InDelta(t, 100.0, pos.FundPercentage, 0.01)
	// profit = 50000 - 10000 = 40000
	assert.InDelta(t, 40000.0, pos.Profit, 0.01)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

// ── GetFundPortfolio ──────────────────────────────────────────────────────────

func TestGetFundPortfolio_Empty(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	cols := []string{"listing_id", "quantity", "average_cost", "acquisition_date"}
	fundMock.ExpectQuery("SELECT listing_id, quantity, average_cost, acquisition_date").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows(cols))

	resp, err := s.GetFundPortfolio(context.Background(), &pb.GetFundPortfolioRequest{FundId: 1})
	require.NoError(t, err)
	assert.Empty(t, resp.Positions)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestGetFundPortfolio_Happy(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	acqDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	cols := []string{"listing_id", "quantity", "average_cost", "acquisition_date"}
	rows := sqlmock.NewRows(cols).AddRow(int64(10), 50.0, 125.0, acqDate)
	fundMock.ExpectQuery("SELECT listing_id, quantity, average_cost, acquisition_date").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	resp, err := s.GetFundPortfolio(context.Background(), &pb.GetFundPortfolioRequest{FundId: 1})
	require.NoError(t, err)
	require.Len(t, resp.Positions, 1)
	assert.Equal(t, int64(10), resp.Positions[0].ListingId)
	assert.InDelta(t, 50.0, resp.Positions[0].Quantity, 0.001)
	assert.InDelta(t, 125.0, resp.Positions[0].AverageCost, 0.001)
	assert.Equal(t, "2026-04-01", resp.Positions[0].AcquisitionDate)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

// ── WithdrawFund ──────────────────────────────────────────────────────────────

func TestWithdrawFund_Case1_Immediate(t *testing.T) {
	s, fundMock, accountDBMock, empMock := newFundServer(t, &mockAccountClient{})

	// SELECT fund: liquid_assets=500000, active=true, account_id=42
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(500000.0, true, int64(42)))
	// SELECT position
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	// Debit fund account
	accountDBMock.ExpectExec("UPDATE accounts SET balance").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Credit destination account
	accountDBMock.ExpectExec("UPDATE accounts SET balance").
		WithArgs(1000.0, int64(99)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// DB tx
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE client_fund_positions SET total_invested_amount").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectCommit()
	// fetchFundByID
	addFetchFundRows(fundMock, accountDBMock, empMock, 1, "Test Fund", 100, 5)

	resp, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT",
		DestinationAccountId: 99, Amount: 1000.0,
	})
	require.NoError(t, err)
	assert.False(t, resp.Pending)
	assert.NotNil(t, resp.Fund)
	assert.Equal(t, int64(1), resp.Fund.Id)
}

func TestWithdrawFund_Case1_WithdrawAll(t *testing.T) {
	// amount=0 means withdraw full position
	s, fundMock, accountDBMock, empMock := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(500000.0, true, int64(42)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(2500.0))
	// Debit fund account
	accountDBMock.ExpectExec("UPDATE accounts SET balance").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Credit destination account
	accountDBMock.ExpectExec("UPDATE accounts SET balance").
		WithArgs(2500.0, int64(99)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE client_fund_positions SET total_invested_amount").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectCommit()
	addFetchFundRows(fundMock, accountDBMock, empMock, 1, "Test Fund", 100, 5)

	resp, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT",
		DestinationAccountId: 99, Amount: 0, // withdraw all
	})
	require.NoError(t, err)
	assert.False(t, resp.Pending)
	assert.NotNil(t, resp.Fund)
}

func TestWithdrawFund_Case2_ClientAutoLiquidate(t *testing.T) {
	var createOrderCalled bool
	order := &mockOrderClient{
		createOrderFn: func(_ context.Context, req *pb_order.CreateOrderRequest, _ ...grpc.CallOption) (*pb_order.CreateOrderResponse, error) {
			createOrderCalled = true
			assert.Equal(t, "SELL", req.Direction)
			assert.Equal(t, int64(1), req.FundId)
			return &pb_order.CreateOrderResponse{}, nil
		},
	}
	s, fundMock, _, _ := newFundServerFull(t, &mockAccountClient{}, order)

	// liquid_assets=100 < amount=1000 → Case 2
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(100.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	// fetch account_id, manager_id for SELL orders
	fundMock.ExpectQuery("SELECT account_id, manager_id FROM investment_funds").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "manager_id"}).AddRow(int64(42), int64(5)))
	// portfolio positions
	fundMock.ExpectQuery("SELECT listing_id, quantity, average_cost FROM fund_portfolio_positions").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"listing_id", "quantity", "average_cost"}).
			AddRow(int64(10), 10.0, 100.0)) // 10*100=1000, covers deficit
	// INSERT PENDING transaction
	fundMock.ExpectExec("INSERT INTO client_fund_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT",
		DestinationAccountId: 99, Amount: 1000.0,
	})
	require.NoError(t, err)
	assert.True(t, resp.Pending)
	assert.Equal(t, "Payment will arrive once orders are executed", resp.Message)
	assert.True(t, createOrderCalled)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestWithdrawFund_Case2_BankNoAutoLiquidate(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	// liquid_assets=100 < amount=1000, clientType=BANK → FailedPrecondition
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(100.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))

	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 0, ClientType: "BANK",
		DestinationAccountId: 99, Amount: 1000.0,
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// ── CheckPendingWithdrawals ───────────────────────────────────────────────────

func TestCheckPendingWithdrawals_Completed(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})

	// liquid_assets sufficient to cover the PENDING tx
	fundMock.ExpectQuery("SELECT liquid_assets FROM investment_funds").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets"}).AddRow(5000.0))
	// One PENDING outflow transaction
	fundMock.ExpectQuery("SELECT id, client_id, client_type, amount, destination_account_id").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "client_id", "client_type", "amount", "destination_account_id"}).
			AddRow(int64(1), int64(7), "CLIENT", 1000.0, int64(99)))
	// Credit destination account
	accountDBMock.ExpectExec("UPDATE accounts SET balance").
		WithArgs(1000.0, int64(99)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// DB tx
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE client_fund_positions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE client_fund_transactions SET status = 'COMPLETED'").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectCommit()

	resp, err := s.CheckPendingWithdrawals(context.Background(), &pb.CheckPendingWithdrawalsRequest{FundId: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Completed)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestCheckPendingWithdrawals_StillInsufficient(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	// liquid_assets still too low
	fundMock.ExpectQuery("SELECT liquid_assets FROM investment_funds").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets"}).AddRow(50.0))
	fundMock.ExpectQuery("SELECT id, client_id, client_type, amount, destination_account_id").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "client_id", "client_type", "amount", "destination_account_id"}).
			AddRow(int64(1), int64(7), "CLIENT", 1000.0, int64(99)))

	resp, err := s.CheckPendingWithdrawals(context.Background(), &pb.CheckPendingWithdrawalsRequest{FundId: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Completed)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

// ── GetFundPerformanceHistory ─────────────────────────────────────────────────

func TestGetFundPerformanceHistory_Happy(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery(`SELECT TO_CHAR\(date, 'YYYY-MM-DD'\), fund_value, profit`).
		WithArgs(int64(1), "2025-01-01", "2025-01-31").
		WillReturnRows(sqlmock.NewRows([]string{"date", "fund_value", "profit"}).
			AddRow("2025-01-01", 100000.0, 5000.0).
			AddRow("2025-01-15", 102000.0, 7000.0))

	resp, err := s.GetFundPerformanceHistory(context.Background(), &pb.GetFundPerformanceRequest{
		FundId: 1, From: "2025-01-01", To: "2025-01-31",
	})
	require.NoError(t, err)
	require.Len(t, resp.Records, 2)
	assert.Equal(t, "2025-01-01", resp.Records[0].Date)
	assert.InDelta(t, 100000.0, resp.Records[0].FundValue, 0.01)
	assert.InDelta(t, 5000.0, resp.Records[0].Profit, 0.01)
	assert.Equal(t, "2025-01-15", resp.Records[1].Date)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestGetFundPerformanceHistory_Empty(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery(`SELECT TO_CHAR\(date, 'YYYY-MM-DD'\), fund_value, profit`).
		WithArgs(int64(1), "2025-01-01", "2025-01-31").
		WillReturnRows(sqlmock.NewRows([]string{"date", "fund_value", "profit"}))

	resp, err := s.GetFundPerformanceHistory(context.Background(), &pb.GetFundPerformanceRequest{
		FundId: 1, From: "2025-01-01", To: "2025-01-31",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Records)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

func TestGetFundPerformanceHistory_DBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})

	fundMock.ExpectQuery(`SELECT TO_CHAR\(date, 'YYYY-MM-DD'\), fund_value, profit`).
		WithArgs(int64(1), "2025-01-01", "2025-01-31").
		WillReturnError(fmt.Errorf("db down"))

	_, err := s.GetFundPerformanceHistory(context.Background(), &pb.GetFundPerformanceRequest{
		FundId: 1, From: "2025-01-01", To: "2025-01-31",
	})
	require.Error(t, err)
	require.NoError(t, fundMock.ExpectationsWereMet())
}

// ── InvestFund ────────────────────────────────────────────────────────────────

func TestInvestFund_ZeroAmount(t *testing.T) {
	s, _, _, _ := newFundServer(t, &mockAccountClient{})
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{FundId: 1, Amount: 0})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestInvestFund_NegativeAmount(t *testing.T) {
	s, _, _, _ := newFundServer(t, &mockAccountClient{})
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{FundId: 1, Amount: -500})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestInvestFund_FundNotFound(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnError(sql.ErrNoRows)
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{FundId: 99, Amount: 1000})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestInvestFund_FundDBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnError(fmt.Errorf("connection reset"))
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{FundId: 1, Amount: 1000})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestInvestFund_FundInactive(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(500.0, false, int64(0)))
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{FundId: 1, Amount: 1000})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestInvestFund_BelowMinimum(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(5000.0, true, int64(0)))
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{FundId: 1, Amount: 100})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestInvestFund_AccountNotFound(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(500.0, true, int64(0)))
	accountDBMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnError(sql.ErrNoRows)
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{
		FundId: 1, SourceAccountId: 99, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestInvestFund_AccountDBError(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(500.0, true, int64(0)))
	accountDBMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{
		FundId: 1, SourceAccountId: 10, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestInvestFund_InsufficientBalance(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(500.0, true, int64(0)))
	accountDBMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(200.0))
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{
		FundId: 1, SourceAccountId: 10, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestInvestFund_DebitFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(500.0, true, int64(0)))
	accountDBMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(5000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnError(fmt.Errorf("update failed"))
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{
		FundId: 1, SourceAccountId: 10, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestInvestFund_BeginTxFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(500.0, true, int64(0)))
	accountDBMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(5000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin().WillReturnError(fmt.Errorf("begin failed"))
	// compensation credit
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{
		FundId: 1, SourceAccountId: 10, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestInvestFund_UpdateLiquidAssetsFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(500.0, true, int64(0)))
	accountDBMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(5000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets \\+").
		WillReturnError(fmt.Errorf("update failed"))
	fundMock.ExpectRollback()
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{
		FundId: 1, SourceAccountId: 10, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestInvestFund_UpsertPositionFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(500.0, true, int64(0)))
	accountDBMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(5000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_positions").
		WillReturnError(fmt.Errorf("upsert failed"))
	fundMock.ExpectRollback()
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", SourceAccountId: 10, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestInvestFund_InsertTransactionFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(500.0, true, int64(0)))
	accountDBMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(5000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_positions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_transactions").
		WillReturnError(fmt.Errorf("insert failed"))
	fundMock.ExpectRollback()
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", SourceAccountId: 10, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestInvestFund_CommitFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(500.0, true, int64(0)))
	accountDBMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(5000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_positions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectCommit().WillReturnError(fmt.Errorf("commit failed"))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", SourceAccountId: 10, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── WithdrawFund error paths ──────────────────────────────────────────────────

func TestWithdrawFund_FundNotFound(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnError(sql.ErrNoRows)
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{FundId: 99, ClientId: 1, ClientType: "CLIENT", Amount: 100})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestWithdrawFund_FundDBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{FundId: 1, ClientId: 1, ClientType: "CLIENT", Amount: 100})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestWithdrawFund_FundInactive(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, false, int64(0)))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{FundId: 1, ClientId: 1, ClientType: "CLIENT", Amount: 100})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestWithdrawFund_PositionNotFound(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnError(sql.ErrNoRows)
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{FundId: 1, ClientId: 7, ClientType: "CLIENT", Amount: 100})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestWithdrawFund_PositionDBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{FundId: 1, ClientId: 7, ClientType: "CLIENT", Amount: 100})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestWithdrawFund_AmountExceedsPosition(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(500.0))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestWithdrawFund_CreditAccountFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnError(fmt.Errorf("credit failed"))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 500,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestWithdrawFund_BeginTxFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin().WillReturnError(fmt.Errorf("begin failed"))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 500,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestWithdrawFund_UpdateLiquidAssetsFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnError(fmt.Errorf("update failed"))
	fundMock.ExpectRollback()
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 500,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestWithdrawFund_UpdatePositionFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE client_fund_positions SET total_invested_amount").
		WillReturnError(fmt.Errorf("update failed"))
	fundMock.ExpectRollback()
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 500,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestWithdrawFund_InsertTransactionFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE client_fund_positions SET total_invested_amount").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_transactions").
		WillReturnError(fmt.Errorf("insert failed"))
	fundMock.ExpectRollback()
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 500,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestWithdrawFund_CommitFails(t *testing.T) {
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE client_fund_positions SET total_invested_amount").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectCommit().WillReturnError(fmt.Errorf("commit failed"))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 500,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── autoLiquidate error paths ─────────────────────────────────────────────────

func TestAutoLiquidate_FetchFundDetailFails(t *testing.T) {
	s, fundMock, _, _ := newFundServerFull(t, &mockAccountClient{}, &mockOrderClient{})
	// fund select (liquid < amount → Case 2)
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(50.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	// autoLiquidate: SELECT account_id, manager_id
	fundMock.ExpectQuery("SELECT account_id, manager_id FROM investment_funds").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestAutoLiquidate_PortfolioQueryFails(t *testing.T) {
	s, fundMock, _, _ := newFundServerFull(t, &mockAccountClient{}, &mockOrderClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(50.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	fundMock.ExpectQuery("SELECT account_id, manager_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "manager_id"}).AddRow(int64(42), int64(5)))
	fundMock.ExpectQuery("SELECT listing_id, quantity, average_cost FROM fund_portfolio_positions").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestAutoLiquidate_InsertPendingFails(t *testing.T) {
	s, fundMock, _, _ := newFundServerFull(t, &mockAccountClient{}, &mockOrderClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(50.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	fundMock.ExpectQuery("SELECT account_id, manager_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "manager_id"}).AddRow(int64(42), int64(5)))
	fundMock.ExpectQuery("SELECT listing_id, quantity, average_cost FROM fund_portfolio_positions").
		WillReturnRows(sqlmock.NewRows([]string{"listing_id", "quantity", "average_cost"}).
			AddRow(int64(10), 20.0, 100.0))
	fundMock.ExpectExec("INSERT INTO client_fund_transactions").
		WillReturnError(fmt.Errorf("insert failed"))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── CheckPendingWithdrawals error paths ────────────────────────────────────────

func TestCheckPendingWithdrawals_DBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets FROM investment_funds").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.CheckPendingWithdrawals(context.Background(), &pb.CheckPendingWithdrawalsRequest{FundId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCheckPendingWithdrawals_PendingQueryFails(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets"}).AddRow(5000.0))
	fundMock.ExpectQuery("SELECT id, client_id, client_type, amount, destination_account_id").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.CheckPendingWithdrawals(context.Background(), &pb.CheckPendingWithdrawalsRequest{FundId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCheckPendingWithdrawals_AccountCreditFails(t *testing.T) {
	// Credit fails → continues loop, returns 0 completed
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets"}).AddRow(5000.0))
	fundMock.ExpectQuery("SELECT id, client_id, client_type, amount, destination_account_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "client_id", "client_type", "amount", "destination_account_id"}).
			AddRow(int64(1), int64(7), "CLIENT", 1000.0, int64(99)))
	accountDBMock.ExpectExec("UPDATE accounts SET balance").
		WillReturnError(fmt.Errorf("credit failed"))
	resp, err := s.CheckPendingWithdrawals(context.Background(), &pb.CheckPendingWithdrawalsRequest{FundId: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Completed)
}

func TestCheckPendingWithdrawals_BeginTxFails(t *testing.T) {
	// BeginTx fails → undoes credit, continues loop, returns 0 completed
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets"}).AddRow(5000.0))
	fundMock.ExpectQuery("SELECT id, client_id, client_type, amount, destination_account_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "client_id", "client_type", "amount", "destination_account_id"}).
			AddRow(int64(1), int64(7), "CLIENT", 1000.0, int64(99)))
	accountDBMock.ExpectExec("UPDATE accounts SET balance").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin().WillReturnError(fmt.Errorf("begin failed"))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	resp, err := s.CheckPendingWithdrawals(context.Background(), &pb.CheckPendingWithdrawalsRequest{FundId: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Completed)
}

func TestCheckPendingWithdrawals_CommitFails(t *testing.T) {
	// Commit fails → rollback + undo credit, continues, returns 0 completed
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets"}).AddRow(5000.0))
	fundMock.ExpectQuery("SELECT id, client_id, client_type, amount, destination_account_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "client_id", "client_type", "amount", "destination_account_id"}).
			AddRow(int64(1), int64(7), "CLIENT", 1000.0, int64(99)))
	accountDBMock.ExpectExec("UPDATE accounts SET balance").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE client_fund_positions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE client_fund_transactions SET status = 'COMPLETED'").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectCommit().WillReturnError(fmt.Errorf("commit failed"))
	fundMock.ExpectRollback()
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	resp, err := s.CheckPendingWithdrawals(context.Background(), &pb.CheckPendingWithdrawalsRequest{FundId: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Completed)
}

// ── UpdateFundHolding error paths ─────────────────────────────────────────────

func TestUpdateFundHolding_BeginTxFails(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectBegin().WillReturnError(fmt.Errorf("begin failed"))
	_, err := s.UpdateFundHolding(context.Background(), &pb.UpdateFundHoldingRequest{
		FundId: 1, ListingId: 10, Quantity: 5, Price: 100.0, Direction: "BUY",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestUpdateFundHolding_BuyUpdateLiquidFails(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnError(fmt.Errorf("update failed"))
	_, err := s.UpdateFundHolding(context.Background(), &pb.UpdateFundHoldingRequest{
		FundId: 1, ListingId: 10, Quantity: 5, Price: 100.0, Direction: "BUY",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestUpdateFundHolding_BuyUpsertPositionFails(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO fund_portfolio_positions").
		WillReturnError(fmt.Errorf("upsert failed"))
	_, err := s.UpdateFundHolding(context.Background(), &pb.UpdateFundHoldingRequest{
		FundId: 1, ListingId: 10, Quantity: 5, Price: 100.0, Direction: "BUY",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestUpdateFundHolding_SellUpdateLiquidFails(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets \\+").
		WillReturnError(fmt.Errorf("update failed"))
	_, err := s.UpdateFundHolding(context.Background(), &pb.UpdateFundHoldingRequest{
		FundId: 1, ListingId: 10, Quantity: 3, Price: 100.0, Direction: "SELL",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestUpdateFundHolding_SellUpdatePositionFails(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE fund_portfolio_positions SET quantity = quantity -").
		WillReturnError(fmt.Errorf("update failed"))
	_, err := s.UpdateFundHolding(context.Background(), &pb.UpdateFundHoldingRequest{
		FundId: 1, ListingId: 10, Quantity: 3, Price: 100.0, Direction: "SELL",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestUpdateFundHolding_SellDeletePositionFails(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE fund_portfolio_positions SET quantity = quantity -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("DELETE FROM fund_portfolio_positions").
		WillReturnError(fmt.Errorf("delete failed"))
	_, err := s.UpdateFundHolding(context.Background(), &pb.UpdateFundHoldingRequest{
		FundId: 1, ListingId: 10, Quantity: 3, Price: 100.0, Direction: "SELL",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestUpdateFundHolding_CommitFails(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO fund_portfolio_positions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectCommit().WillReturnError(fmt.Errorf("commit failed"))
	_, err := s.UpdateFundHolding(context.Background(), &pb.UpdateFundHoldingRequest{
		FundId: 1, ListingId: 10, Quantity: 5, Price: 100.0, Direction: "BUY",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── TransferFundsByManager error path ─────────────────────────────────────────

func TestTransferFundsByManager_DBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectExec("UPDATE investment_funds SET manager_id").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.TransferFundsByManager(context.Background(), &pb.TransferFundsByManagerRequest{
		OldManagerId: 7, NewManagerId: 99,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── DeleteFund update error ───────────────────────────────────────────────────

func TestDeleteFund_UpdateFails(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	fundMock.ExpectExec("UPDATE investment_funds SET active = false").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.DeleteFund(context.Background(), &pb.DeleteFundRequest{Id: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── ListFunds fetch error ──────────────────────────────────────────────────────

func TestListFunds_FetchFundError(t *testing.T) {
	// SELECT id returns 1 row, but fetchFundByID fails on the SELECT
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	fundMock.ExpectQuery("SELECT id, name, description").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.ListFunds(context.Background(), &pb.ListFundsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── ValidateFundAccount DB error ──────────────────────────────────────────────

func TestValidateFundAccount_DBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT account_id, manager_id, liquid_assets").
		WillReturnError(fmt.Errorf("connection error"))
	_, err := s.ValidateFundAccount(context.Background(), &pb.ValidateFundAccountRequest{
		FundId: 1, ManagerId: 5, RequiredAmount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetMyPositions DB error ───────────────────────────────────────────────────

func TestGetMyPositions_DBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT cfp.fund_id").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.GetMyPositions(context.Background(), &pb.GetMyPositionsRequest{ClientId: 1, ClientType: "CLIENT"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetFundPortfolio DB error ─────────────────────────────────────────────────

func TestGetFundPortfolio_DBError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT listing_id, quantity, average_cost, acquisition_date").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.GetFundPortfolio(context.Background(), &pb.GetFundPortfolioRequest{FundId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── Scan error tests ──────────────────────────────────────────────────────────

func TestListFunds_ScanError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	// "bad" is a string that cannot be scanned into int64
	fundMock.ExpectQuery("SELECT id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("bad"))
	_, err := s.ListFunds(context.Background(), &pb.ListFundsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetFundPortfolio_ScanError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	// Pass "not-a-date" for acquisition_date time.Time scan — will fail conversion
	fundMock.ExpectQuery("SELECT listing_id, quantity, average_cost, acquisition_date").
		WillReturnRows(sqlmock.NewRows([]string{"listing_id", "quantity", "average_cost", "acquisition_date"}).
			AddRow(int64(10), 50.0, 125.0, "not-a-date"))
	_, err := s.GetFundPortfolio(context.Background(), &pb.GetFundPortfolioRequest{FundId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetFundPerformanceHistory_ScanError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	// "bad" can't be scanned into float64
	fundMock.ExpectQuery(`SELECT TO_CHAR\(date, 'YYYY-MM-DD'\), fund_value, profit`).
		WithArgs(int64(1), "2025-01-01", "2025-01-31").
		WillReturnRows(sqlmock.NewRows([]string{"date", "fund_value", "profit"}).
			AddRow("2025-01-01", "bad", 0.0))
	_, err := s.GetFundPerformanceHistory(context.Background(), &pb.GetFundPerformanceRequest{
		FundId: 1, From: "2025-01-01", To: "2025-01-31",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestAutoLiquidate_ScanError(t *testing.T) {
	s, fundMock, _, _ := newFundServerFull(t, &mockAccountClient{}, &mockOrderClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(50.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	fundMock.ExpectQuery("SELECT account_id, manager_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "manager_id"}).AddRow(int64(42), int64(5)))
	// "bad" can't be scanned into int64 for listing_id
	fundMock.ExpectQuery("SELECT listing_id, quantity, average_cost FROM fund_portfolio_positions").
		WillReturnRows(sqlmock.NewRows([]string{"listing_id", "quantity", "average_cost"}).
			AddRow("bad", 10.0, 100.0))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 1000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetMyPositions_ScanError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	// pass "bad" for float64 totalInvested — will fail scan
	fundMock.ExpectQuery("SELECT cfp.fund_id").
		WillReturnRows(sqlmock.NewRows([]string{
			"fund_id", "total_invested_amount", "name", "description",
			"fund_value", "minimum_contribution", "total_all_invested",
		}).AddRow(int64(1), "bad", "Fund", "", 1000.0, 500.0, 1000.0))
	_, err := s.GetMyPositions(context.Background(), &pb.GetMyPositionsRequest{ClientId: 1, ClientType: "CLIENT"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetBankPositions_ScanError(t *testing.T) {
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	// "bad" for bankInvested float64 — will fail scan
	fundMock.ExpectQuery(`SELECT cfp\.fund_id`).
		WillReturnRows(sqlmock.NewRows([]string{"fund_id", "bank_invested", "name", "fund_value", "manager_id", "total_all_invested"}).
			AddRow(int64(1), "bad", "Fund", 1000.0, int64(5), 1000.0))
	_, err := s.GetBankPositions(context.Background(), &pb.GetBankPositionsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCheckPendingWithdrawals_ScanErrorContinues(t *testing.T) {
	// scan error inside loop → continues → returns 0 completed (no error returned)
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets"}).AddRow(5000.0))
	// "bad" for txID int64 — scan error
	fundMock.ExpectQuery("SELECT id, client_id, client_type, amount, destination_account_id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "client_id", "client_type", "amount", "destination_account_id"}).
			AddRow("bad", int64(7), "CLIENT", 1000.0, int64(99)))
	resp, err := s.CheckPendingWithdrawals(context.Background(), &pb.CheckPendingWithdrawalsRequest{FundId: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Completed)
}

func TestAutoLiquidate_BreakWhenDeficitCovered(t *testing.T) {
	// Two positions; first covers deficit, so loop breaks on second row
	var orderCount int
	order := &mockOrderClient{
		createOrderFn: func(_ context.Context, _ *pb_order.CreateOrderRequest, _ ...grpc.CallOption) (*pb_order.CreateOrderResponse, error) {
			orderCount++
			return &pb_order.CreateOrderResponse{}, nil
		},
	}
	s, fundMock, _, _ := newFundServerFull(t, &mockAccountClient{}, order)

	// liquid=100, position=1000, deficit=900
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(100.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	fundMock.ExpectQuery("SELECT account_id, manager_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "manager_id"}).AddRow(int64(42), int64(5)))
	// Two rows; first: 10 * 100 = 1000 >= deficit 900 → second row triggers break
	fundMock.ExpectQuery("SELECT listing_id, quantity, average_cost FROM fund_portfolio_positions").
		WillReturnRows(sqlmock.NewRows([]string{"listing_id", "quantity", "average_cost"}).
			AddRow(int64(10), 10.0, 100.0).
			AddRow(int64(11), 5.0, 50.0))
	fundMock.ExpectExec("INSERT INTO client_fund_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 1000,
	})
	require.NoError(t, err)
	assert.True(t, resp.Pending)
	// Only first order should have been placed (second row breaks out)
	assert.Equal(t, 1, orderCount)
}

func TestWithdrawFund_FetchFundError(t *testing.T) {
	// All steps succeed but fetchFundByID fails after commit
	s, fundMock, accountDBMock, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(1000.0))
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("UPDATE client_fund_positions SET total_invested_amount").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectCommit()
	// fetchFundByID fails
	fundMock.ExpectQuery("SELECT id, name, description").
		WillReturnError(fmt.Errorf("db error"))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", DestinationAccountId: 99, Amount: 500,
	})
	require.Error(t, err)
}

// ── WithdrawFund: zero position zero amount ────────────────────────────────────

func TestWithdrawFund_ZeroPositionZeroAmount(t *testing.T) {
	// positionAmount=0, amount=0 → amount becomes positionAmount=0 → InvalidArgument "nothing to withdraw"
	s, fundMock, _, _ := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT liquid_assets, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"liquid_assets", "active", "account_id"}).AddRow(10000.0, true, int64(0)))
	fundMock.ExpectQuery("SELECT total_invested_amount FROM client_fund_positions").
		WillReturnRows(sqlmock.NewRows([]string{"total_invested_amount"}).AddRow(0.0))
	_, err := s.WithdrawFund(context.Background(), &pb.WithdrawFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", Amount: 0,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestInvestFund_Happy(t *testing.T) {
	s, fundMock, accountDBMock, empMock := newFundServer(t, &mockAccountClient{})
	fundMock.ExpectQuery("SELECT minimum_contribution, active, account_id FROM investment_funds").
		WillReturnRows(sqlmock.NewRows([]string{"minimum_contribution", "active", "account_id"}).AddRow(500.0, true, int64(42)))
	accountDBMock.ExpectQuery("SELECT available_balance FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"available_balance"}).AddRow(5000.0))
	// Debit source account
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance -").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Credit fund account
	accountDBMock.ExpectExec("UPDATE accounts SET balance = balance \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectBegin()
	fundMock.ExpectExec("UPDATE investment_funds SET liquid_assets = liquid_assets \\+").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_positions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectExec("INSERT INTO client_fund_transactions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	fundMock.ExpectCommit()
	addFetchFundRows(fundMock, accountDBMock, empMock, 1, "Growth Fund", 100, 5)

	resp, err := s.InvestFund(context.Background(), &pb.InvestFundRequest{
		FundId: 1, ClientId: 7, ClientType: "CLIENT", SourceAccountId: 10, Amount: 1000,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Id)
	assert.Equal(t, "Growth Fund", resp.Name)
	require.NoError(t, fundMock.ExpectationsWereMet())
	require.NoError(t, accountDBMock.ExpectationsWereMet())
}
