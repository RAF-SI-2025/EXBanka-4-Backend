package handlers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	pb_email "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- mock email client ----

type mockEmailClient struct{ err error }

func (m *mockEmailClient) SendActivationEmail(_ context.Context, _ *pb_email.SendActivationEmailRequest, _ ...grpc.CallOption) (*pb_email.SendActivationEmailResponse, error) {
	return &pb_email.SendActivationEmailResponse{}, m.err
}
func (m *mockEmailClient) SendPasswordResetEmail(_ context.Context, _ *pb_email.SendPasswordResetEmailRequest, _ ...grpc.CallOption) (*pb_email.SendPasswordResetEmailResponse, error) {
	return &pb_email.SendPasswordResetEmailResponse{}, m.err
}
func (m *mockEmailClient) SendPasswordConfirmationEmail(_ context.Context, _ *pb_email.SendActivationEmailRequest, _ ...grpc.CallOption) (*pb_email.SendActivationEmailResponse, error) {
	return &pb_email.SendActivationEmailResponse{}, m.err
}
func (m *mockEmailClient) SendAccountCreatedEmail(_ context.Context, _ *pb_email.SendAccountCreatedEmailRequest, _ ...grpc.CallOption) (*pb_email.SendAccountCreatedEmailResponse, error) {
	return &pb_email.SendAccountCreatedEmailResponse{}, m.err
}
func (m *mockEmailClient) SendCardConfirmationEmail(_ context.Context, _ *pb_email.SendCardConfirmationEmailRequest, _ ...grpc.CallOption) (*pb_email.SendCardConfirmationEmailResponse, error) {
	return &pb_email.SendCardConfirmationEmailResponse{}, m.err
}
func (m *mockEmailClient) SendLoanLatePaymentEmail(_ context.Context, _ *pb_email.SendLoanLatePaymentEmailRequest, _ ...grpc.CallOption) (*pb_email.SendLoanLatePaymentEmailResponse, error) {
	return &pb_email.SendLoanLatePaymentEmailResponse{}, m.err
}

func newServer(t *testing.T) (*AccountServer, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	clientDB, clientMock, err := sqlmock.New()
	require.NoError(t, err)
	exchangeDB, exchangeMock, err := sqlmock.New()
	require.NoError(t, err)
	s := &AccountServer{DB: db, ClientDB: clientDB, ExchangeDB: exchangeDB, EmailClient: &mockEmailClient{}}
	t.Cleanup(func() { _ = db.Close(); _ = clientDB.Close(); _ = exchangeDB.Close() })
	return s, dbMock, clientMock, exchangeMock
}

// ---- accountTypeCode ----

func TestAccountTypeCode(t *testing.T) {
	cases := map[string]string{
		"personal": "01",
		"business": "04",
		"OTHER":    "01",
		"":         "01",
	}
	for input, want := range cases {
		assert.Equal(t, want, accountTypeCode(input), "accountTypeCode(%q)", input)
	}
}

// ---- GetMyAccounts ----

func TestGetMyAccounts_DBError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT").WillReturnError(sql.ErrConnDone)

	_, err := s.GetMyAccounts(context.Background(), &pb.GetMyAccountsRequest{OwnerId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetMyAccounts_Empty(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "account_name", "account_number", "available_balance", "currency_id"}))

	resp, err := s.GetMyAccounts(context.Background(), &pb.GetMyAccountsRequest{OwnerId: 1})
	require.NoError(t, err)
	assert.Empty(t, resp.Accounts)
}

func TestGetMyAccounts_HappyPath(t *testing.T) {
	s, dbMock, _, exchangeMock := newServer(t)

	dbMock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "account_name", "account_number", "available_balance", "currency_id"}).
			AddRow(int64(1), "Moj racun", "888000100000000101", float64(1000), int64(1)),
	)
	exchangeMock.ExpectQuery("SELECT code FROM currencies").WillReturnRows(
		sqlmock.NewRows([]string{"code"}).AddRow("RSD"),
	)

	resp, err := s.GetMyAccounts(context.Background(), &pb.GetMyAccountsRequest{OwnerId: 5})
	require.NoError(t, err)
	require.Len(t, resp.Accounts, 1)
	assert.Equal(t, "Moj racun", resp.Accounts[0].AccountName)
	assert.Equal(t, "RSD", resp.Accounts[0].CurrencyCode)
}

// ---- GetAccount ----

func TestGetAccount_NotFound(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	_, err := s.GetAccount(context.Background(), &pb.GetAccountRequest{AccountId: 99, OwnerId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetAccount_WrongOwner(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "account_name", "account_number", "owner_id", "balance", "available_balance", "reserved_funds", "currency_id", "status", "account_type", "account_subtype", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "company_id"}).
			AddRow(int64(1), "Racun", "888000100000000101", int64(99), float64(500), float64(500), float64(0), int64(1), "ACTIVE", "personal", "", float64(0), float64(0), float64(0), float64(0), nil),
	)

	_, err := s.GetAccount(context.Background(), &pb.GetAccountRequest{AccountId: 1, OwnerId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGetAccount_HappyPath(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	dbMock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "account_name", "account_number", "owner_id", "balance", "available_balance", "reserved_funds", "currency_id", "status", "account_type", "account_subtype", "daily_limit", "monthly_limit", "daily_spent", "monthly_spent", "company_id"}).
			AddRow(int64(1), "Racun", "888000100000000101", int64(5), float64(1000), float64(900), float64(100), int64(1), "ACTIVE", "personal", "", float64(1000), float64(5000), float64(100), float64(200), nil),
	)
	exchangeMock.ExpectQuery("SELECT code").WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	clientMock.ExpectQuery("SELECT first_name").WillReturnRows(sqlmock.NewRows([]string{"first_name", "last_name"}).AddRow("Ana", "Anic"))

	resp, err := s.GetAccount(context.Background(), &pb.GetAccountRequest{AccountId: 1, OwnerId: 5})
	require.NoError(t, err)
	assert.Equal(t, "Racun", resp.Account.AccountName)
	assert.Equal(t, "RSD", resp.Account.CurrencyCode)
	assert.Equal(t, "Ana Anic", resp.Account.Owner)
}

// ---- RenameAccount ----

func TestRenameAccount_NotFound(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT account_name").WillReturnError(sql.ErrNoRows)

	_, err := s.RenameAccount(context.Background(), &pb.RenameAccountRequest{AccountId: 99, OwnerId: 1, NewName: "Novi"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestRenameAccount_WrongOwner(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT account_name").WillReturnRows(
		sqlmock.NewRows([]string{"account_name", "owner_id"}).AddRow("Stari", int64(99)),
	)

	_, err := s.RenameAccount(context.Background(), &pb.RenameAccountRequest{AccountId: 1, OwnerId: 1, NewName: "Novi"})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestRenameAccount_SameName(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT account_name").WillReturnRows(
		sqlmock.NewRows([]string{"account_name", "owner_id"}).AddRow("Isti", int64(1)),
	)

	_, err := s.RenameAccount(context.Background(), &pb.RenameAccountRequest{AccountId: 1, OwnerId: 1, NewName: "Isti"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRenameAccount_NameConflict(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT account_name").WillReturnRows(
		sqlmock.NewRows([]string{"account_name", "owner_id"}).AddRow("Stari", int64(1)),
	)
	dbMock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	_, err := s.RenameAccount(context.Background(), &pb.RenameAccountRequest{AccountId: 1, OwnerId: 1, NewName: "Novi"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRenameAccount_HappyPath(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT account_name").WillReturnRows(
		sqlmock.NewRows([]string{"account_name", "owner_id"}).AddRow("Stari", int64(1)),
	)
	dbMock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	dbMock.ExpectExec("UPDATE accounts").WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := s.RenameAccount(context.Background(), &pb.RenameAccountRequest{AccountId: 1, OwnerId: 1, NewName: "Novi"})
	require.NoError(t, err)
}

// ---- GetAllAccounts ----

func TestGetAllAccounts_DBError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT").WillReturnError(sql.ErrConnDone)

	_, err := s.GetAllAccounts(context.Background(), &pb.GetAllAccountsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetAllAccounts_HappyPath(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	dbMock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "account_number", "account_name", "owner_id", "account_type", "currency_id", "available_balance", "account_subtype"}).
			AddRow(int64(1), "888000100000000101", "Racun 1", int64(5), "personal", int64(1), float64(1000), ""),
	)
	exchangeMock.ExpectQuery("SELECT code").WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	clientMock.ExpectQuery("SELECT first_name").WillReturnRows(sqlmock.NewRows([]string{"first_name", "last_name"}).AddRow("Ana", "Anic"))

	resp, err := s.GetAllAccounts(context.Background(), &pb.GetAllAccountsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Accounts, 1)
	assert.Equal(t, "Ana", resp.Accounts[0].OwnerFirstName)
	assert.Equal(t, "RSD", resp.Accounts[0].CurrencyCode)
}

// ---- CreateAccount ----

func TestCreateAccount_ClientNotFound(t *testing.T) {
	s, _, clientMock, _ := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnError(sql.ErrNoRows)

	_, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 99, EmployeeId: 1, CurrencyCode: "RSD", AccountType: "CURRENT", AccountName: "Test",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCreateAccount_CurrencyNotFound(t *testing.T) {
	s, _, clientMock, exchangeMock := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email", "first_name"}).AddRow(int64(1), "ana@test.com", "Ana"),
	)
	exchangeMock.ExpectQuery("SELECT id, code").WillReturnError(sql.ErrNoRows)

	_, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, EmployeeId: 1, CurrencyCode: "XXX", AccountType: "CURRENT", AccountName: "Test",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCreateAccount_HappyPath_Personal(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email", "first_name"}).AddRow(int64(1), "ana@test.com", "Ana"),
	)
	exchangeMock.ExpectQuery("SELECT id, code").WillReturnRows(
		sqlmock.NewRows([]string{"id", "code"}).AddRow(int64(1), "RSD"),
	)
	dbMock.ExpectQuery("INSERT INTO accounts").WillReturnRows(
		sqlmock.NewRows([]string{"id", "created_date"}).AddRow(int64(1), "2026-03-18"),
	)

	resp, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, EmployeeId: 2, CurrencyCode: "RSD", AccountType: "CURRENT",
		AccountName: "Tekuci racun", InitialBalance: 500,
	})
	require.NoError(t, err)
	assert.Equal(t, "CURRENT", resp.Account.AccountType)
	assert.Equal(t, "ACTIVE", resp.Account.Status)
	assert.Equal(t, float64(500), resp.Account.Balance)
}

func TestCreateAccount_HappyPath_Business_NewCompany(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email", "first_name"}).AddRow(int64(1), "firma@test.com", "Marko"),
	)
	exchangeMock.ExpectQuery("SELECT id, code").WillReturnRows(
		sqlmock.NewRows([]string{"id", "code"}).AddRow(int64(1), "RSD"),
	)
	// Company lookup – not found
	dbMock.ExpectQuery("SELECT id FROM companies").WillReturnError(sql.ErrNoRows)
	// Company insert
	dbMock.ExpectQuery("INSERT INTO companies").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(10)),
	)
	// Account insert
	dbMock.ExpectQuery("INSERT INTO accounts").WillReturnRows(
		sqlmock.NewRows([]string{"id", "created_date"}).AddRow(int64(2), "2026-03-18"),
	)

	resp, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, EmployeeId: 2, CurrencyCode: "RSD", AccountType: "business",
		AccountName: "Poslovni racun",
		CompanyData: &pb.CompanyData{
			Name: "Firma DOO", RegistrationNumber: "12345678", Pib: "987654321",
			ActivityCode: "62.01", Address: "Beograd",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "business", resp.Account.AccountType)
}

func TestCreateAccount_HappyPath_Business_ExistingCompany(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email", "first_name"}).AddRow(int64(1), "firma@test.com", "Marko"),
	)
	exchangeMock.ExpectQuery("SELECT id, code").WillReturnRows(
		sqlmock.NewRows([]string{"id", "code"}).AddRow(int64(1), "RSD"),
	)
	// Company already exists
	dbMock.ExpectQuery("SELECT id FROM companies").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(10)),
	)
	// Account insert
	dbMock.ExpectQuery("INSERT INTO accounts").WillReturnRows(
		sqlmock.NewRows([]string{"id", "created_date"}).AddRow(int64(3), "2026-03-18"),
	)

	resp, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, EmployeeId: 2, CurrencyCode: "RSD", AccountType: "business",
		AccountName: "Drugi poslovni",
		CompanyData: &pb.CompanyData{
			Name: "Firma DOO", RegistrationNumber: "12345678",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "business", resp.Account.AccountType)
}

// ---- Additional DB error coverage ----

func TestGetMyAccounts_ScanError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "account_name", "account_number", "available_balance", "currency_id"}).
			AddRow("not-an-int", "name", "number", 100.0, 1),
	)
	_, err := s.GetMyAccounts(context.Background(), &pb.GetMyAccountsRequest{OwnerId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetAccount_InternalError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT").WillReturnError(sql.ErrConnDone)
	_, err := s.GetAccount(context.Background(), &pb.GetAccountRequest{AccountId: 1, OwnerId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestRenameAccount_QueryInternalError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT account_name").WillReturnError(sql.ErrConnDone)
	_, err := s.RenameAccount(context.Background(), &pb.RenameAccountRequest{AccountId: 1, OwnerId: 1, NewName: "X"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestRenameAccount_ExecError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT account_name").WillReturnRows(
		sqlmock.NewRows([]string{"account_name", "owner_id"}).AddRow("Stari", int64(1)),
	)
	dbMock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	dbMock.ExpectExec("UPDATE accounts").WillReturnError(sql.ErrConnDone)
	_, err := s.RenameAccount(context.Background(), &pb.RenameAccountRequest{AccountId: 1, OwnerId: 1, NewName: "Novi"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetAllAccounts_ScanError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "account_number", "account_name", "owner_id", "account_type", "currency_id", "available_balance"}).
			AddRow("not-an-int", "ACC1", "Racun", 1, "CURRENT", 1, 1000.0),
	)
	_, err := s.GetAllAccounts(context.Background(), &pb.GetAllAccountsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateAccount_ClientInternalError(t *testing.T) {
	s, _, clientMock, _ := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnError(sql.ErrConnDone)
	_, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, CurrencyCode: "RSD", AccountType: "CURRENT", AccountName: "Test",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateAccount_CurrencyInternalError(t *testing.T) {
	s, _, clientMock, exchangeMock := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email", "first_name"}).AddRow(int64(1), "a@b.com", "Ana"),
	)
	exchangeMock.ExpectQuery("SELECT id, code").WillReturnError(sql.ErrConnDone)
	_, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, CurrencyCode: "RSD", AccountType: "CURRENT", AccountName: "Test",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateAccount_CompanyInsertError(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email", "first_name"}).AddRow(int64(1), "a@b.com", "Ana"),
	)
	exchangeMock.ExpectQuery("SELECT id, code").WillReturnRows(
		sqlmock.NewRows([]string{"id", "code"}).AddRow(int64(1), "RSD"),
	)
	dbMock.ExpectQuery("SELECT id FROM companies").WillReturnError(sql.ErrNoRows)
	dbMock.ExpectQuery("INSERT INTO companies").WillReturnError(sql.ErrConnDone)
	_, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, CurrencyCode: "RSD", AccountType: "BUSINESS", AccountName: "Test",
		CompanyData: &pb.CompanyData{Name: "Firma", RegistrationNumber: "123"},
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateAccount_CompanyLookupError(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email", "first_name"}).AddRow(int64(1), "a@b.com", "Ana"),
	)
	exchangeMock.ExpectQuery("SELECT id, code").WillReturnRows(
		sqlmock.NewRows([]string{"id", "code"}).AddRow(int64(1), "RSD"),
	)
	dbMock.ExpectQuery("SELECT id FROM companies").WillReturnError(sql.ErrConnDone)
	_, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, CurrencyCode: "RSD", AccountType: "BUSINESS", AccountName: "Test",
		CompanyData: &pb.CompanyData{Name: "Firma", RegistrationNumber: "123"},
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateAccount_InsertError(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email", "first_name"}).AddRow(int64(1), "a@b.com", "Ana"),
	)
	exchangeMock.ExpectQuery("SELECT id, code").WillReturnRows(
		sqlmock.NewRows([]string{"id", "code"}).AddRow(int64(1), "RSD"),
	)
	dbMock.ExpectQuery("INSERT INTO accounts").WillReturnError(sql.ErrConnDone)
	_, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, CurrencyCode: "RSD", AccountType: "CURRENT", AccountName: "Test",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateAccount_EmailError_DoesNotFail(t *testing.T) {
	s, dbMock, clientMock, exchangeMock := newServer(t)
	s.EmailClient = &mockEmailClient{err: sql.ErrConnDone}
	clientMock.ExpectQuery("SELECT id, email, first_name").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email", "first_name"}).AddRow(int64(1), "a@b.com", "Ana"),
	)
	exchangeMock.ExpectQuery("SELECT id, code").WillReturnRows(
		sqlmock.NewRows([]string{"id", "code"}).AddRow(int64(1), "RSD"),
	)
	dbMock.ExpectQuery("INSERT INTO accounts").WillReturnRows(
		sqlmock.NewRows([]string{"id", "created_date"}).AddRow(int64(1), "2026-03-18"),
	)
	resp, err := s.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		ClientId: 1, CurrencyCode: "RSD", AccountType: "CURRENT", AccountName: "Test",
	})
	require.NoError(t, err, "email error must not fail the request")
	assert.NotNil(t, resp)
}

// ---- UpdateAccountLimits ----

func TestUpdateAccountLimits_AdminCall_DBError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectExec(`UPDATE accounts SET daily_limit`).WillReturnError(sql.ErrConnDone)
	_, err := s.UpdateAccountLimits(context.Background(), &pb.UpdateAccountLimitsRequest{
		AccountId: 1, OwnerId: 0, DailyLimit: 1000, MonthlyLimit: 5000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestUpdateAccountLimits_AdminCall_NotFound(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectExec(`UPDATE accounts SET daily_limit`).WillReturnResult(sqlmock.NewResult(0, 0))
	_, err := s.UpdateAccountLimits(context.Background(), &pb.UpdateAccountLimitsRequest{
		AccountId: 99, OwnerId: 0, DailyLimit: 1000, MonthlyLimit: 5000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateAccountLimits_AdminCall_Happy(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectExec(`UPDATE accounts SET daily_limit`).WillReturnResult(sqlmock.NewResult(1, 1))
	resp, err := s.UpdateAccountLimits(context.Background(), &pb.UpdateAccountLimitsRequest{
		AccountId: 1, OwnerId: 0, DailyLimit: 1000, MonthlyLimit: 5000,
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestUpdateAccountLimits_ClientCall_DBError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectExec(`UPDATE accounts SET daily_limit`).WillReturnError(sql.ErrConnDone)
	_, err := s.UpdateAccountLimits(context.Background(), &pb.UpdateAccountLimitsRequest{
		AccountId: 1, OwnerId: 42, DailyLimit: 500, MonthlyLimit: 2000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestUpdateAccountLimits_ClientCall_NotFound(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectExec(`UPDATE accounts SET daily_limit`).WillReturnResult(sqlmock.NewResult(0, 0))
	_, err := s.UpdateAccountLimits(context.Background(), &pb.UpdateAccountLimitsRequest{
		AccountId: 99, OwnerId: 42, DailyLimit: 500, MonthlyLimit: 2000,
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateAccountLimits_ClientCall_Happy(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectExec(`UPDATE accounts SET daily_limit`).WillReturnResult(sqlmock.NewResult(1, 1))
	resp, err := s.UpdateAccountLimits(context.Background(), &pb.UpdateAccountLimitsRequest{
		AccountId: 1, OwnerId: 42, DailyLimit: 500, MonthlyLimit: 2000,
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ---- GetBankAccounts ----

func TestGetBankAccounts_DBError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery(`SELECT account_number`).WillReturnError(sql.ErrConnDone)
	_, err := s.GetBankAccounts(context.Background(), &pb.GetBankAccountsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetBankAccounts_ScanError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	// Return only 1 column, but Scan expects 6 — causes scan error
	dbMock.ExpectQuery(`SELECT id, account_number`).WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(1)),
	)
	_, err := s.GetBankAccounts(context.Background(), &pb.GetBankAccountsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetBankAccounts_Empty(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectQuery(`SELECT id, account_number`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "account_number", "account_name", "balance", "available_balance", "currency_id"}),
	)
	resp, err := s.GetBankAccounts(context.Background(), &pb.GetBankAccountsRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Accounts)
}

func TestGetBankAccounts_Happy(t *testing.T) {
	s, dbMock, _, exchangeMock := newServer(t)
	dbMock.ExpectQuery(`SELECT id, account_number`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "account_number", "account_name", "balance", "available_balance", "currency_id"}).
			AddRow(int64(1), "888000000000000001", "EUR Bank", float64(1000000), float64(1000000), int64(2)),
	)
	exchangeMock.ExpectQuery(`SELECT code FROM currencies`).WillReturnRows(
		sqlmock.NewRows([]string{"code"}).AddRow("EUR"),
	)
	resp, err := s.GetBankAccounts(context.Background(), &pb.GetBankAccountsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Accounts, 1)
	assert.Equal(t, "EUR", resp.Accounts[0].CurrencyCode)
}

// ---- DeleteAccount ----

func TestDeleteAccount_DBError(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectExec(`DELETE FROM accounts`).WillReturnError(sql.ErrConnDone)
	_, err := s.DeleteAccount(context.Background(), &pb.DeleteAccountRequest{AccountId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestDeleteAccount_NotFound(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectExec(`DELETE FROM accounts`).WillReturnResult(sqlmock.NewResult(0, 0))
	_, err := s.DeleteAccount(context.Background(), &pb.DeleteAccountRequest{AccountId: 99})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestDeleteAccount_Happy(t *testing.T) {
	s, dbMock, _, _ := newServer(t)
	dbMock.ExpectExec(`DELETE FROM accounts`).WillReturnResult(sqlmock.NewResult(1, 1))
	resp, err := s.DeleteAccount(context.Background(), &pb.DeleteAccountRequest{AccountId: 1})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
