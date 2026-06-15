package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_email "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
)

// ── gRPC client stubs ────────────────────────────────────────────────────────

type mockClientClient struct {
	resp *pb_client.GetClientByIdResponse
	err  error
}

func (m *mockClientClient) GetAllClients(ctx context.Context, in *pb_client.GetAllClientsRequest, opts ...grpc.CallOption) (*pb_client.GetAllClientsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockClientClient) GetClientById(ctx context.Context, in *pb_client.GetClientByIdRequest, opts ...grpc.CallOption) (*pb_client.GetClientByIdResponse, error) {
	return m.resp, m.err
}
func (m *mockClientClient) CreateClient(ctx context.Context, in *pb_client.CreateClientRequest, opts ...grpc.CallOption) (*pb_client.CreateClientResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockClientClient) UpdateClient(ctx context.Context, in *pb_client.UpdateClientRequest, opts ...grpc.CallOption) (*pb_client.UpdateClientResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockClientClient) GetClientCredentials(ctx context.Context, in *pb_client.GetClientCredentialsRequest, opts ...grpc.CallOption) (*pb_client.GetClientCredentialsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockClientClient) ActivateClient(ctx context.Context, in *pb_client.ActivateClientRequest, opts ...grpc.CallOption) (*pb_client.ActivateClientResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockClientClient) UpdateLoginAttempts(ctx context.Context, in *pb_client.UpdateLoginAttemptsRequest, opts ...grpc.CallOption) (*pb_client.UpdateLoginAttemptsResponse, error) {
	return &pb_client.UpdateLoginAttemptsResponse{}, nil
}

type mockEmailClient struct{}

func (m *mockEmailClient) SendActivationEmail(ctx context.Context, in *pb_email.SendActivationEmailRequest, opts ...grpc.CallOption) (*pb_email.SendActivationEmailResponse, error) {
	return &pb_email.SendActivationEmailResponse{}, nil
}
func (m *mockEmailClient) SendPasswordResetEmail(ctx context.Context, in *pb_email.SendPasswordResetEmailRequest, opts ...grpc.CallOption) (*pb_email.SendPasswordResetEmailResponse, error) {
	return &pb_email.SendPasswordResetEmailResponse{}, nil
}
func (m *mockEmailClient) SendPasswordConfirmationEmail(ctx context.Context, in *pb_email.SendActivationEmailRequest, opts ...grpc.CallOption) (*pb_email.SendActivationEmailResponse, error) {
	return &pb_email.SendActivationEmailResponse{}, nil
}
func (m *mockEmailClient) SendAccountCreatedEmail(ctx context.Context, in *pb_email.SendAccountCreatedEmailRequest, opts ...grpc.CallOption) (*pb_email.SendAccountCreatedEmailResponse, error) {
	return &pb_email.SendAccountCreatedEmailResponse{}, nil
}
func (m *mockEmailClient) SendCardConfirmationEmail(ctx context.Context, in *pb_email.SendCardConfirmationEmailRequest, opts ...grpc.CallOption) (*pb_email.SendCardConfirmationEmailResponse, error) {
	return &pb_email.SendCardConfirmationEmailResponse{}, nil
}
func (m *mockEmailClient) SendLoanLatePaymentEmail(ctx context.Context, in *pb_email.SendLoanLatePaymentEmailRequest, opts ...grpc.CallOption) (*pb_email.SendLoanLatePaymentEmailResponse, error) {
	return &pb_email.SendLoanLatePaymentEmailResponse{}, nil
}
func (m *mockEmailClient) SendAccountLockedEmail(ctx context.Context, in *pb_email.SendAccountLockedEmailRequest, opts ...grpc.CallOption) (*pb_email.SendAccountLockedEmailResponse, error) {
	return &pb_email.SendAccountLockedEmailResponse{}, nil
}
func (m *mockEmailClient) SendPaymentNotificationEmail(ctx context.Context, in *pb_email.SendPaymentNotificationEmailRequest, opts ...grpc.CallOption) (*pb_email.SendPaymentNotificationEmailResponse, error) {
	return &pb_email.SendPaymentNotificationEmailResponse{}, nil
}
func (m *mockEmailClient) SendCardBlockedEmail(ctx context.Context, in *pb_email.SendCardBlockedEmailRequest, opts ...grpc.CallOption) (*pb_email.SendCardBlockedEmailResponse, error) {
	return &pb_email.SendCardBlockedEmailResponse{}, nil
}
func (m *mockEmailClient) SendLoanApprovedEmail(ctx context.Context, in *pb_email.SendLoanApprovedEmailRequest, opts ...grpc.CallOption) (*pb_email.SendLoanApprovedEmailResponse, error) {
	return &pb_email.SendLoanApprovedEmailResponse{}, nil
}
func (m *mockEmailClient) SendLimitChangeEmail(ctx context.Context, in *pb_email.SendLimitChangeEmailRequest, opts ...grpc.CallOption) (*pb_email.SendLimitChangeEmailResponse, error) {
	return &pb_email.SendLimitChangeEmailResponse{}, nil
}

// newLoanServer creates a LoanServer backed by sqlmock DBs for loan_db and account_db.
func newLoanServer(t *testing.T) (*LoanServer, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	db, loanMock, err := sqlmock.New()
	require.NoError(t, err)
	accountDB, accountMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close(); _ = accountDB.Close() })
	return &LoanServer{DB: db, AccountDB: accountDB}, loanMock, accountMock
}

// newLoanServerWithExchange creates a LoanServer with loan_db, account_db, and exchange_db mocked.
func newLoanServerWithExchange(t *testing.T) (*LoanServer, sqlmock.Sqlmock, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	db, loanMock, err := sqlmock.New()
	require.NoError(t, err)
	accountDB, accountMock, err := sqlmock.New()
	require.NoError(t, err)
	exchangeDB, exchangeMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close(); _ = accountDB.Close(); _ = exchangeDB.Close() })
	return &LoanServer{DB: db, AccountDB: accountDB, ExchangeDB: exchangeDB}, loanMock, accountMock, exchangeMock
}

// loanSummaryColumns returns the columns for GetClientLoans scan.
func loanSummaryColumns() []string {
	return []string{"id", "loan_number", "account_number", "loan_type", "amount", "currency", "status", "repayment_period"}
}

// loanDetailColumns returns the columns for GetLoanDetails scan (16 columns).
func loanDetailColumns() []string {
	return []string{
		"id", "loan_number", "account_number", "loan_type", "interest_rate_type",
		"amount", "currency", "repayment_period", "nominal_rate", "effective_rate",
		"agreed_date", "maturity_date", "next_installment_amount", "next_installment_date",
		"remaining_debt", "status",
	}
}

// loanFullColumns returns the 21 columns for GetAllLoans/GetAllApplications scan.
func loanFullColumns() []string {
	return []string{
		"id", "loan_number", "account_number", "loan_type", "interest_rate_type",
		"amount", "currency", "repayment_period", "nominal_rate", "effective_rate",
		"agreed_date", "maturity_date", "next_installment_amount", "next_installment_date",
		"remaining_debt", "status", "purpose", "monthly_salary", "employment_status",
		"employment_period", "contact_phone",
	}
}

// installmentColumns returns the columns for queryInstallments scan.
func installmentColumns() []string {
	return []string{"id", "loan_id", "installment_amount", "interest_rate", "currency", "expected_due_date", "actual_due_date", "status"}
}

// sampleLoanFullRow returns a representative full row for loanFullColumns.
func sampleLoanFullRow() []driver.Value {
	return []driver.Value{
		int64(1), int64(1234567890123), "265000191399797801", "CASH", "FIXED",
		float64(100000), "RSD", int32(24), float64(6.25), float64(6.50),
		time.Now(), time.Now().AddDate(2, 0, 0),
		sql.NullFloat64{Float64: 4500, Valid: true},
		sql.NullTime{Time: time.Now().AddDate(0, 1, 0), Valid: true},
		sql.NullFloat64{Float64: 95500, Valid: true},
		"PENDING",
		sql.NullString{String: "Adaptacija stana", Valid: true},
		sql.NullFloat64{Float64: 80000, Valid: true},
		sql.NullString{String: "PERMANENT", Valid: true},
		sql.NullInt32{Int32: 36, Valid: true},
		sql.NullString{String: "0601234567", Valid: true},
	}
}

// ---- GetClientLoans ----

func TestGetClientLoans_Empty(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").
		WillReturnRows(sqlmock.NewRows(loanSummaryColumns()))

	resp, err := s.GetClientLoans(context.Background(), &pb.GetClientLoansRequest{ClientId: 1})
	require.NoError(t, err)
	assert.Empty(t, resp.Loans)
}

func TestGetClientLoans_HappyPath(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").
		WillReturnRows(sqlmock.NewRows(loanSummaryColumns()).
			AddRow(int64(1), int64(1234567890123), "ACC001", "CASH", float64(100000), "RSD", "APPROVED", int32(24)).
			AddRow(int64(2), int64(9876543210987), "ACC001", "HOUSING", float64(500000), "RSD", "PENDING", int32(120)))

	resp, err := s.GetClientLoans(context.Background(), &pb.GetClientLoansRequest{ClientId: 1})
	require.NoError(t, err)
	assert.Len(t, resp.Loans, 2)
	assert.Equal(t, "CASH", resp.Loans[0].LoanType)
	assert.Equal(t, "PENDING", resp.Loans[1].Status)
}

func TestGetClientLoans_DBError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").WillReturnError(sql.ErrConnDone)

	_, err := s.GetClientLoans(context.Background(), &pb.GetClientLoansRequest{ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetClientLoans_ScanError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").
		WillReturnRows(sqlmock.NewRows(loanSummaryColumns()).
			AddRow("not-an-int", 0, "ACC", "CASH", 0.0, "RSD", "PENDING", 0))

	_, err := s.GetClientLoans(context.Background(), &pb.GetClientLoansRequest{ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- GetLoanDetails ----

func TestGetLoanDetails_NotFound(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number, account_number").
		WillReturnRows(sqlmock.NewRows(loanDetailColumns()))

	_, err := s.GetLoanDetails(context.Background(), &pb.GetLoanDetailsRequest{LoanId: 999})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetLoanDetails_DBError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number, account_number").WillReturnError(sql.ErrConnDone)

	_, err := s.GetLoanDetails(context.Background(), &pb.GetLoanDetailsRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetLoanDetails_HappyPath(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	now := time.Now()
	loanMock.ExpectQuery("SELECT id, loan_number, account_number").
		WillReturnRows(sqlmock.NewRows(loanDetailColumns()).AddRow(
			int64(1), int64(1234567890123), "ACC001", "CASH", "FIXED",
			float64(100000), "RSD", int32(24), float64(6.25), float64(6.50),
			now, now.AddDate(2, 0, 0),
			sql.NullFloat64{Float64: 4500, Valid: true},
			sql.NullTime{Time: now.AddDate(0, 1, 0), Valid: true},
			sql.NullFloat64{Float64: 95500, Valid: true},
			"APPROVED",
		))
	// queryInstallments call
	loanMock.ExpectQuery("SELECT id, loan_id").
		WillReturnRows(sqlmock.NewRows(installmentColumns()))

	resp, err := s.GetLoanDetails(context.Background(), &pb.GetLoanDetailsRequest{LoanId: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Loan.Id)
	assert.Equal(t, "CASH", resp.Loan.LoanType)
	assert.Equal(t, "APPROVED", resp.Loan.Status)
	assert.Equal(t, float64(4500), resp.Loan.NextInstallmentAmount)
}

// ---- GetLoanInstallments ----

func TestGetLoanInstallments_Empty(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_id").
		WillReturnRows(sqlmock.NewRows(installmentColumns()))

	resp, err := s.GetLoanInstallments(context.Background(), &pb.GetLoanInstallmentsRequest{LoanId: 1})
	require.NoError(t, err)
	assert.Empty(t, resp.Installments)
}

func TestGetLoanInstallments_HappyPath(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	now := time.Now()
	loanMock.ExpectQuery("SELECT id, loan_id").
		WillReturnRows(sqlmock.NewRows(installmentColumns()).
			AddRow(int64(1), int64(5), float64(4500), float64(6.5), "RSD", now, sql.NullTime{}, "PENDING").
			AddRow(int64(2), int64(5), float64(4500), float64(6.5), "RSD", now.AddDate(0, 1, 0), sql.NullTime{}, "PENDING"))

	resp, err := s.GetLoanInstallments(context.Background(), &pb.GetLoanInstallmentsRequest{LoanId: 5})
	require.NoError(t, err)
	assert.Len(t, resp.Installments, 2)
	assert.Equal(t, float64(4500), resp.Installments[0].InstallmentAmount)
}

func TestGetLoanInstallments_DBError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_id").WillReturnError(sql.ErrConnDone)

	_, err := s.GetLoanInstallments(context.Background(), &pb.GetLoanInstallmentsRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- RejectLoan ----

func TestRejectLoan_NotFound(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectExec("UPDATE loans SET status = 'REJECTED'").
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := s.RejectLoan(context.Background(), &pb.RejectLoanRequest{LoanId: 99})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestRejectLoan_HappyPath(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectExec("UPDATE loans SET status = 'REJECTED'").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.RejectLoan(context.Background(), &pb.RejectLoanRequest{LoanId: 1})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestRejectLoan_DBError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectExec("UPDATE loans SET status = 'REJECTED'").WillReturnError(sql.ErrConnDone)

	_, err := s.RejectLoan(context.Background(), &pb.RejectLoanRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- GetAllLoanApplications ----

func TestGetAllLoanApplications_Empty(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").
		WillReturnRows(sqlmock.NewRows(loanFullColumns()))

	resp, err := s.GetAllLoanApplications(context.Background(), &pb.GetAllLoanApplicationsRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Applications)
}

func TestGetAllLoanApplications_HappyPath(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").
		WillReturnRows(sqlmock.NewRows(loanFullColumns()).AddRow(sampleLoanFullRow()...))

	resp, err := s.GetAllLoanApplications(context.Background(), &pb.GetAllLoanApplicationsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Applications, 1)
	assert.Equal(t, "CASH", resp.Applications[0].LoanType)
}

func TestGetAllLoanApplications_DBError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").WillReturnError(sql.ErrConnDone)

	_, err := s.GetAllLoanApplications(context.Background(), &pb.GetAllLoanApplicationsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- GetAllLoans ----

func TestGetAllLoans_HappyPath(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").
		WillReturnRows(sqlmock.NewRows(loanFullColumns()).AddRow(sampleLoanFullRow()...))

	resp, err := s.GetAllLoans(context.Background(), &pb.GetAllLoansRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Loans, 1)
}

func TestGetAllLoans_WithFilters(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").
		WillReturnRows(sqlmock.NewRows(loanFullColumns()))

	resp, err := s.GetAllLoans(context.Background(), &pb.GetAllLoansRequest{
		LoanType: "CASH", Status: "APPROVED",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Loans)
}

func TestGetAllLoans_DBError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").WillReturnError(sql.ErrConnDone)

	_, err := s.GetAllLoans(context.Background(), &pb.GetAllLoansRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- SubmitLoanApplication ----

func TestSubmitLoanApplication_InvalidLoanType(t *testing.T) {
	s, _, _ := newLoanServer(t)
	_, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "INVALID", InterestRateType: "FIXED", Amount: 100000, RepaymentPeriod: 12,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSubmitLoanApplication_InvalidRateType(t *testing.T) {
	s, _, _ := newLoanServer(t)
	_, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "MIXED", Amount: 100000, RepaymentPeriod: 12,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSubmitLoanApplication_InvalidAmount(t *testing.T) {
	s, _, _ := newLoanServer(t)
	_, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "FIXED", Amount: -100, RepaymentPeriod: 12,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSubmitLoanApplication_InvalidRepaymentPeriod(t *testing.T) {
	s, _, _ := newLoanServer(t)
	// CASH loans don't allow 360 months
	_, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "FIXED", Amount: 100000, RepaymentPeriod: 360,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSubmitLoanApplication_AccountNotFound(t *testing.T) {
	s, _, accountMock, _ := newLoanServerWithExchange(t)
	accountMock.ExpectQuery("SELECT currency_id FROM accounts").WillReturnError(sql.ErrNoRows)

	_, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "FIXED", Amount: 100000, RepaymentPeriod: 12,
		AccountNumber: "NONEXISTENT", Currency: "RSD",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestSubmitLoanApplication_CurrencyMismatch(t *testing.T) {
	s, _, accountMock, exchangeMock := newLoanServerWithExchange(t)
	accountMock.ExpectQuery("SELECT currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("EUR"))

	_, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "FIXED", Amount: 100000, RepaymentPeriod: 12,
		AccountNumber: "ACC001", Currency: "RSD", // account is EUR, loan is RSD
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSubmitLoanApplication_HappyPath(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)
	// account is RSD (currency_id=1)
	accountMock.ExpectQuery("SELECT currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(1)))
	// currency code = RSD
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	// toRSD: currency is already RSD, no exchange query needed
	// INSERT loan
	loanMock.ExpectQuery("INSERT INTO loans").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	resp, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "FIXED", Amount: 100000, RepaymentPeriod: 12,
		AccountNumber: "ACC001", ClientId: 1, Currency: "RSD",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.LoanId)
	assert.Greater(t, resp.MonthlyInstallment, float64(0))
}

// ---- ApproveLoan ----

func TestApproveLoan_NotFound(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT status, currency").
		WillReturnRows(sqlmock.NewRows([]string{"status", "currency", "loan_type", "interest_rate_type", "account_number", "amount", "effective_rate", "repayment_period", "agreed_date"}))

	_, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 999})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestApproveLoan_NotPending(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT status, currency").
		WillReturnRows(sqlmock.NewRows([]string{"status", "currency", "loan_type", "interest_rate_type", "account_number", "amount", "effective_rate", "repayment_period", "agreed_date"}).
			AddRow("APPROVED", "RSD", "CASH", "FIXED", "ACC001", float64(100000), float64(6.5), int(12), time.Now()))

	_, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestApproveLoan_DisburseError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)
	loanMock.ExpectQuery("SELECT status, currency").
		WillReturnRows(sqlmock.NewRows([]string{"status", "currency", "loan_type", "interest_rate_type", "account_number", "amount", "effective_rate", "repayment_period", "agreed_date"}).
			AddRow("PENDING", "RSD", "CASH", "FIXED", "ACC001", float64(100000), float64(6.5), int(12), time.Now()))
	exchangeMock.ExpectQuery("SELECT id FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	accountMock.ExpectExec("UPDATE accounts SET balance").WillReturnError(sql.ErrConnDone)

	_, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestApproveLoan_HappyPath(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)
	agreedDate := time.Now()
	// 1 month repayment period for minimal mock setup
	loanMock.ExpectQuery("SELECT status, currency").
		WillReturnRows(sqlmock.NewRows([]string{"status", "currency", "loan_type", "interest_rate_type", "account_number", "amount", "effective_rate", "repayment_period", "agreed_date"}).
			AddRow("PENDING", "RSD", "CASH", "FIXED", "ACC001", float64(100000), float64(6.5), int(1), agreedDate))
	exchangeMock.ExpectQuery("SELECT id FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	accountMock.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(1, 1)) // debit bank
	accountMock.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(1, 1)) // credit client
	loanMock.ExpectBegin()
	loanMock.ExpectExec("INSERT INTO loan_installments").WillReturnResult(sqlmock.NewResult(1, 1))
	loanMock.ExpectExec("UPDATE loans SET status = 'APPROVED'").WillReturnResult(sqlmock.NewResult(1, 1))
	loanMock.ExpectCommit()

	resp, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 1})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

// ---- GetLoanDetails: internal DB error ----

func TestGetLoanDetails_InternalError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number, account_number").WillReturnError(sql.ErrConnDone)

	_, err := s.GetLoanDetails(context.Background(), &pb.GetLoanDetailsRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- ApproveLoan: internal DB error ----

func TestApproveLoan_DBError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT status, currency").WillReturnError(sql.ErrConnDone)

	_, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- ApproveLoan: installment INSERT error ----

func TestApproveLoan_InstallmentInsertError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)
	agreedDate := time.Now()
	loanMock.ExpectQuery("SELECT status, currency").
		WillReturnRows(sqlmock.NewRows([]string{"status", "currency", "loan_type", "interest_rate_type", "account_number", "amount", "effective_rate", "repayment_period", "agreed_date"}).
			AddRow("PENDING", "RSD", "CASH", "FIXED", "ACC001", float64(100000), float64(6.5), int(2), agreedDate))
	exchangeMock.ExpectQuery("SELECT id FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	accountMock.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(1, 1)) // debit bank
	accountMock.ExpectExec("UPDATE accounts SET balance").WillReturnResult(sqlmock.NewResult(1, 1)) // credit client
	loanMock.ExpectBegin()
	loanMock.ExpectExec("INSERT INTO loan_installments").WillReturnError(sql.ErrConnDone)

	_, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- processInstallment: installment lookup DB error ----

func TestTriggerInstallments_InstallmentLookupError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	loanMock.ExpectQuery(`SELECT id, loan_number, client_id`).
		WithArgs(int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_number", "client_id", "account_number",
			"next_installment_amount", "currency", "remaining_debt",
		}).AddRow(int64(11), int64(1234567890123), int64(1), "ACC001", float64(4500), "RSD", float64(90000)))

	// DB error (not ErrNoRows) in installment lookup
	loanMock.ExpectQuery(`SELECT id, retry_count FROM loan_installments`).
		WithArgs(int64(11)).
		WillReturnError(sql.ErrConnDone)

	// processInstallment logs and returns — collectInstallments still counts it
	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 11})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Processed)
}

// ---- updateVariableRates: query error ----

func TestUpdateVariableRates_QueryError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	loanMock.ExpectQuery(`SELECT id, loan_type`).WillReturnError(sql.ErrConnDone)

	s.updateVariableRates() // logs and returns, no panic
}

// ---- SubmitLoanApplication: exchange currency code DB error ----

func TestSubmitLoanApplication_CurrencyCodeDBError(t *testing.T) {
	s, _, accountMock, exchangeMock := newLoanServerWithExchange(t)

	accountMock.ExpectQuery("SELECT currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchangeMock.ExpectQuery("SELECT code FROM currencies").WillReturnError(sql.ErrConnDone)

	_, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "FIXED", Amount: 100000, RepaymentPeriod: 12,
		AccountNumber: "ACC001", Currency: "EUR",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ---- SubmitLoanApplication: foreign currency covers toRSD non-RSD branch ----

func TestSubmitLoanApplication_EURCurrency(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	accountMock.ExpectQuery("SELECT currency_id FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchangeMock.ExpectQuery("SELECT code FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("EUR"))
	// toRSD: look up EUR middle rate
	exchangeMock.ExpectQuery("SELECT middle_rate FROM daily_exchange_rates").
		WillReturnRows(sqlmock.NewRows([]string{"middle_rate"}).AddRow(float64(117.5)))
	loanMock.ExpectQuery("INSERT INTO loans").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(55)))

	resp, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "FIXED", Amount: 1000, RepaymentPeriod: 12,
		AccountNumber: "ACC001", ClientId: 1, Currency: "EUR",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(55), resp.LoanId)
}

// ---- GetAllLoanApplications: filter branches ----

func TestGetAllLoanApplications_WithFilters(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").
		WillReturnRows(sqlmock.NewRows(loanFullColumns()))

	resp, err := s.GetAllLoanApplications(context.Background(), &pb.GetAllLoanApplicationsRequest{
		LoanType: "CASH", AccountNumber: "ACC001",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Applications)
}

// ---- GetAllLoans: AccountNumber filter branch ----

func TestGetAllLoans_WithAccountNumber(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").
		WillReturnRows(sqlmock.NewRows(loanFullColumns()).AddRow(sampleLoanFullRow()...))

	resp, err := s.GetAllLoans(context.Background(), &pb.GetAllLoansRequest{
		AccountNumber: "265000191399797801",
	})
	require.NoError(t, err)
	assert.Len(t, resp.Loans, 1)
}

// ---- TriggerInstallments covers collectInstallments ----

func TestTriggerInstallments_ForceLoan_Empty(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	loanMock.ExpectQuery(`SELECT id, loan_number, client_id`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_number", "client_id", "account_number",
			"next_installment_amount", "currency", "remaining_debt",
		}))

	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 1})
	require.NoError(t, err)
	assert.Equal(t, int32(0), resp.Processed)
}

func TestTriggerInstallments_Normal_Empty(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	loanMock.ExpectQuery(`SELECT id, loan_number, client_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_number", "client_id", "account_number",
			"next_installment_amount", "currency", "remaining_debt",
		}))

	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 0})
	require.NoError(t, err)
	assert.Equal(t, int32(0), resp.Processed)
}

func TestTriggerInstallments_ForceLoan_PaymentSuccess(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	// collectInstallments: 1 loan due (remainingDebt=4500 = amount, so newRemaining=0 → PAID_OFF)
	loanMock.ExpectQuery(`SELECT id, loan_number, client_id`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_number", "client_id", "account_number",
			"next_installment_amount", "currency", "remaining_debt",
		}).AddRow(int64(7), int64(1234567890123), int64(1), "ACC001", float64(4500), "RSD", float64(4500)))

	// processInstallment: find UNPAID installment
	loanMock.ExpectQuery(`SELECT id, retry_count FROM loan_installments`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(10), int(0)))

	// resolve currency and bank account
	exchangeMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery(`SELECT account_number FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))

	// debit succeeds (affected=1)
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// credit bank account
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// mark PAID
	loanMock.ExpectExec(`UPDATE loan_installments`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// advance schedule (newRemaining=0 → PAID_OFF)
	loanMock.ExpectExec(`UPDATE loans SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 7})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Processed)
}

func TestTriggerInstallments_ForceLoan_InsufficientFunds(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	// collectInstallments: 1 loan
	loanMock.ExpectQuery(`SELECT id, loan_number, client_id`).
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_number", "client_id", "account_number",
			"next_installment_amount", "currency", "remaining_debt",
		}).AddRow(int64(8), int64(9876543210987), int64(2), "ACC002", float64(4500), "RSD", float64(90000)))

	// processInstallment: find UNPAID installment (retry_count=0)
	loanMock.ExpectQuery(`SELECT id, retry_count FROM loan_installments`).
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(20), int(0)))

	// resolve currency and bank account
	exchangeMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery(`SELECT account_number FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))

	// debit fails due to insufficient funds (affected=0)
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// mark LATE
	loanMock.ExpectExec(`UPDATE loan_installments`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// set IN_DELAY
	loanMock.ExpectExec(`UPDATE loans SET status = 'IN_DELAY'`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 8})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Processed)
}

func TestTriggerInstallments_ForceLoan_MaxRetries(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	// collectInstallments: 1 loan
	loanMock.ExpectQuery(`SELECT id, loan_number, client_id`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_number", "client_id", "account_number",
			"next_installment_amount", "currency", "remaining_debt",
		}).AddRow(int64(9), int64(1111111111111), int64(3), "ACC003", float64(3000), "RSD", float64(50000)))

	// processInstallment: retry_count=4 → newRetry=5 >= maxRetries(5) → penalty applies
	loanMock.ExpectQuery(`SELECT id, retry_count FROM loan_installments`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(30), int(4)))

	// resolve currency and bank account
	exchangeMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery(`SELECT account_number FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))

	// debit fails (affected=0)
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// mark LATE
	loanMock.ExpectExec(`UPDATE loan_installments`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// set IN_DELAY with penalty rate (penaltyRate=0.05)
	loanMock.ExpectExec(`UPDATE loans SET status = 'IN_DELAY'`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 9})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Processed)
}

// ---- updateVariableRates covers monthly cron logic and paidInstallmentCount ----

func TestUpdateVariableRates_Empty(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	loanMock.ExpectQuery(`SELECT id, loan_type`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_type", "amount", "nominal_rate", "effective_rate",
			"remaining_debt", "repayment_period", "agreed_date",
		}))

	s.updateVariableRates() // no panic
}

func TestUpdateVariableRates_WithLoan(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	loanMock.ExpectQuery(`SELECT id, loan_type`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_type", "amount", "nominal_rate", "effective_rate",
			"remaining_debt", "repayment_period", "agreed_date",
		}).AddRow(int64(1), "CASH", float64(100000), float64(6.25), float64(8.0), float64(95500), int(24), time.Now()))

	// paidInstallmentCount: 1 paid so far (remaining = 24-1 = 23)
	loanMock.ExpectQuery(`SELECT COUNT\(\*\) FROM loan_installments`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// UPDATE loans with new rate
	loanMock.ExpectExec(`UPDATE loans SET effective_rate`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s.updateVariableRates()
}

func TestUpdateVariableRates_AllPaid(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	// repaymentPeriod=12, paidCount=12 → remaining=0 → skip
	loanMock.ExpectQuery(`SELECT id, loan_type`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_type", "amount", "nominal_rate", "effective_rate",
			"remaining_debt", "repayment_period", "agreed_date",
		}).AddRow(int64(2), "CASH", float64(50000), float64(6.25), float64(8.0), float64(0), int(12), time.Now()))

	// paidInstallmentCount: all 12 paid
	loanMock.ExpectQuery(`SELECT COUNT\(\*\) FROM loan_installments`).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))

	s.updateVariableRates() // remaining=0 → continue (no UPDATE)
}

// ── SubmitLoanApplication: account DB error ───────────────────────────────────

func TestSubmitLoanApplication_AccountDBError(t *testing.T) {
	s, _, accountMock, _ := newLoanServerWithExchange(t)

	accountMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WithArgs("265-0001-9139979-78").
		WillReturnError(sql.ErrConnDone)

	_, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "FIXED", Amount: 300000,
		RepaymentPeriod: 36, AccountNumber: "265-0001-9139979-78",
		Currency: "RSD",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── SubmitLoanApplication: INSERT error ───────────────────────────────────────

func TestSubmitLoanApplication_InsertError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	// account OK, currency RSD
	accountMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WithArgs("265-0001-9139979-78").
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(1)))
	exchangeMock.ExpectQuery(`SELECT code FROM currencies WHERE id`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("RSD"))
	// toRSD: currency is RSD → returns immediately, no exchange query
	// INSERT fails
	loanMock.ExpectQuery(`INSERT INTO loans`).
		WillReturnError(sql.ErrConnDone)

	_, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "FIXED", Amount: 300000,
		RepaymentPeriod: 36, AccountNumber: "265-0001-9139979-78",
		Currency: "RSD",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── toRSD: rate not found ─────────────────────────────────────────────────────

func TestToRSD_RateNotFound(t *testing.T) {
	s, _, _, exchangeMock := newLoanServerWithExchange(t)

	exchangeMock.ExpectQuery(`SELECT middle_rate FROM daily_exchange_rates`).
		WithArgs("EUR").
		WillReturnRows(sqlmock.NewRows([]string{"middle_rate"})) // empty → ErrNoRows

	result, err := s.toRSD(context.Background(), 100.0, "EUR")
	require.Error(t, err)
	assert.Equal(t, float64(0), result)
}

// ── processInstallment: debit error ──────────────────────────────────────────

func TestProcessInstallment_DebitError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	loanMock.ExpectQuery(`SELECT id, retry_count FROM loan_installments`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(10), 0))
	exchangeMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery(`SELECT account_number FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnError(sql.ErrConnDone)

	// No panic expected — errors are only logged
	s.processInstallment(context.Background(), 1, 1234567890123, 1, "ACC001", 500.0, "RSD", 5000.0)
}

// ── processInstallment: mark PAID error (debit succeeded) ────────────────────

func TestProcessInstallment_MarkPaidError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	loanMock.ExpectQuery(`SELECT id, retry_count FROM loan_installments`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(10), 0))
	exchangeMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery(`SELECT account_number FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(1, 1)) // affected=1 → success path
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(1, 1)) // credit bank account
	loanMock.ExpectExec(`UPDATE loan_installments SET status = 'PAID'`).
		WillReturnError(sql.ErrConnDone)
	loanMock.ExpectExec(`UPDATE loans SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s.processInstallment(context.Background(), 1, 1234567890123, 1, "ACC001", 500.0, "RSD", 5000.0)
}

// ── processInstallment: mark LATE error (debit failed, no funds) ─────────────

func TestProcessInstallment_MarkLateError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	loanMock.ExpectQuery(`SELECT id, retry_count FROM loan_installments`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(10), 0))
	exchangeMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery(`SELECT account_number FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(1, 0)) // affected=0 → insufficient funds
	loanMock.ExpectExec(`UPDATE loan_installments SET status = 'LATE'`).
		WillReturnError(sql.ErrConnDone)
	loanMock.ExpectExec(`UPDATE loans SET status = 'IN_DELAY'`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s.processInstallment(context.Background(), 1, 1234567890123, 1, "ACC001", 500.0, "RSD", 5000.0)
}

// ── processInstallment: set IN_DELAY error ───────────────────────────────────

func TestProcessInstallment_SetDelayError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	loanMock.ExpectQuery(`SELECT id, retry_count FROM loan_installments`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(10), 0))
	exchangeMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery(`SELECT account_number FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(1, 0)) // affected=0
	loanMock.ExpectExec(`UPDATE loan_installments SET status = 'LATE'`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	loanMock.ExpectExec(`UPDATE loans SET status = 'IN_DELAY'`).
		WillReturnError(sql.ErrConnDone)

	s.processInstallment(context.Background(), 1, 1234567890123, 1, "ACC001", 500.0, "RSD", 5000.0)
}

// ── collectInstallments: scan error → continue ───────────────────────────────

func TestCollectInstallments_ScanError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	// Return 1 column instead of 7 → Scan fails → continue → 0 loans
	loanMock.ExpectQuery(`SELECT id, loan_number, client_id`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))

	count := s.collectInstallments(1)
	assert.Equal(t, 0, count)
}

// ── SubmitLoanApplication: toRSD failure → log, use raw amount ───────────────

func TestSubmitLoanApplication_ToRSDFailure(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	// account has EUR (currency_id=2)
	accountMock.ExpectQuery(`SELECT currency_id FROM accounts`).
		WithArgs("265-0001-9139979-78").
		WillReturnRows(sqlmock.NewRows([]string{"currency_id"}).AddRow(int64(2)))
	exchangeMock.ExpectQuery(`SELECT code FROM currencies WHERE id`).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("EUR"))
	// toRSD: EUR rate not found → logs, uses raw amount
	exchangeMock.ExpectQuery(`SELECT middle_rate FROM daily_exchange_rates`).
		WithArgs("EUR").
		WillReturnRows(sqlmock.NewRows([]string{"middle_rate"})) // empty → ErrNoRows
	// INSERT succeeds
	loanMock.ExpectQuery(`INSERT INTO loans`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))

	resp, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "FIXED", Amount: 300000,
		RepaymentPeriod: 36, AccountNumber: "265-0001-9139979-78",
		Currency: "EUR",
	})
	require.NoError(t, err) // error is only logged, raw amount is used
	assert.Equal(t, int64(99), resp.LoanId)
}

// ── updateVariableRates: scan error → continue ───────────────────────────────

func TestUpdateVariableRates_ScanError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	// Return 1 column instead of 8 → Scan fails → log and continue
	loanMock.ExpectQuery(`SELECT id, loan_type`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	s.updateVariableRates() // scan error → logged → no panic
}

// ── queryInstallments: scan error ────────────────────────────────────────────

func TestGetLoanInstallments_ScanError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	// GetLoanInstallments queries via queryInstallments → return 1 col instead of 8
	loanMock.ExpectQuery(`SELECT id, loan_id, installment_amount`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))

	_, err := s.GetLoanInstallments(context.Background(), &pb.GetLoanInstallmentsRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── updateVariableRates: UPDATE exec error ────────────────────────────────────

func TestUpdateVariableRates_UpdateError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	loanMock.ExpectQuery(`SELECT id, loan_type`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_type", "amount", "nominal_rate", "effective_rate",
			"remaining_debt", "repayment_period", "agreed_date",
		}).AddRow(int64(3), "CASH", float64(200000), float64(6.25), float64(8.0), float64(190000), int(36), time.Now()))

	loanMock.ExpectQuery(`SELECT COUNT\(\*\) FROM loan_installments`).
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2)) // remaining = 36-2 = 34

	loanMock.ExpectExec(`UPDATE loans SET effective_rate`).
		WillReturnError(sql.ErrConnDone) // logged, continues

	s.updateVariableRates() // no panic
}

// ── ApproveLoan: BeginTx error ────────────────────────────────────────────────

func TestApproveLoan_BeginTxError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	loanMock.ExpectQuery(`SELECT status, currency, loan_type`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"status", "currency", "loan_type", "interest_rate_type", "account_number",
			"amount", "effective_rate", "repayment_period", "agreed_date",
		}).AddRow("PENDING", "RSD", "CASH", "FIXED", "ACC001", float64(100000), float64(8.0), int(12), time.Now()))

	exchangeMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery(`SELECT account_number FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(1, 1)) // debit bank account
	accountMock.ExpectExec(`UPDATE accounts SET balance = balance \+`).
		WillReturnResult(sqlmock.NewResult(1, 1)) // credit client account

	loanMock.ExpectBegin().WillReturnError(sql.ErrConnDone)

	_, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── ApproveLoan: UPDATE loans error ──────────────────────────────────────────

func TestApproveLoan_UpdateLoansError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	loanMock.ExpectQuery(`SELECT status, currency, loan_type`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"status", "currency", "loan_type", "interest_rate_type", "account_number",
			"amount", "effective_rate", "repayment_period", "agreed_date",
		}).AddRow("PENDING", "RSD", "CASH", "FIXED", "ACC001", float64(100000), float64(8.0), int(1), time.Now()))

	exchangeMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery(`SELECT account_number FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(1, 1)) // debit bank account
	accountMock.ExpectExec(`UPDATE accounts SET balance = balance \+`).
		WillReturnResult(sqlmock.NewResult(1, 1)) // credit client account

	loanMock.ExpectBegin()
	// 1 installment inserted OK
	loanMock.ExpectExec(`INSERT INTO loan_installments`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// UPDATE loans fails
	loanMock.ExpectExec(`UPDATE loans SET status = 'APPROVED'`).
		WillReturnError(sql.ErrConnDone)

	_, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── ApproveLoan: commit error ─────────────────────────────────────────────────

func TestApproveLoan_CommitError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	loanMock.ExpectQuery(`SELECT status, currency, loan_type`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"status", "currency", "loan_type", "interest_rate_type", "account_number",
			"amount", "effective_rate", "repayment_period", "agreed_date",
		}).AddRow("PENDING", "RSD", "CASH", "FIXED", "ACC001", float64(100000), float64(8.0), int(1), time.Now()))

	exchangeMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery(`SELECT account_number FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(1, 1)) // debit bank account
	accountMock.ExpectExec(`UPDATE accounts SET balance = balance \+`).
		WillReturnResult(sqlmock.NewResult(1, 1)) // credit client account

	loanMock.ExpectBegin()
	loanMock.ExpectExec(`INSERT INTO loan_installments`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	loanMock.ExpectExec(`UPDATE loans SET status = 'APPROVED'`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	loanMock.ExpectCommit().WillReturnError(sql.ErrConnDone)

	_, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── processInstallment: email notification paths ──────────────────────────────

func setupInsufficientFundsInstallment(t *testing.T, loanMock, accountMock, exchangeMock sqlmock.Sqlmock) {
	t.Helper()
	// installment lookup returns UNPAID row
	loanMock.ExpectQuery(`SELECT id, retry_count FROM loan_installments`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(10), 0))
	// resolve currency and bank account before debit
	exchangeMock.ExpectQuery(`SELECT id FROM currencies`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery(`SELECT account_number FROM accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	// debit returns 0 rows → insufficient funds
	accountMock.ExpectExec(`UPDATE accounts`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	// mark LATE
	loanMock.ExpectExec(`UPDATE loan_installments`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// update loans to IN_DELAY
	loanMock.ExpectExec(`UPDATE loans SET status = 'IN_DELAY'`).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func TestProcessInstallment_EmailSent(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)
	s.EmailClient = &mockEmailClient{}
	s.ClientClient = &mockClientClient{
		resp: &pb_client.GetClientByIdResponse{
			Client: &pb_client.Client{
				Email: "test@example.com", FirstName: "Test",
			},
		},
	}
	setupInsufficientFundsInstallment(t, loanMock, accountMock, exchangeMock)
	// processInstallment doesn't return → just verify it doesn't panic
	s.processInstallment(context.Background(), 1, 1234567890123, 1, "ACC001", 500.0, "RSD", 5000.0)
}

func TestProcessInstallment_EmailClientLookupFails(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)
	s.EmailClient = &mockEmailClient{}
	s.ClientClient = &mockClientClient{err: fmt.Errorf("client not found")}
	setupInsufficientFundsInstallment(t, loanMock, accountMock, exchangeMock)
	s.processInstallment(context.Background(), 1, 1234567890123, 1, "ACC001", 500.0, "RSD", 5000.0)
}

func TestTriggerInstallments_ForceLoan_NoInstallments(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)

	// collectInstallments returns 1 loan
	loanMock.ExpectQuery(`SELECT id, loan_number, client_id`).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_number", "client_id", "account_number",
			"next_installment_amount", "currency", "remaining_debt",
		}).AddRow(int64(5), int64(1234567890123), int64(1), "ACC001", float64(4500), "RSD", float64(95500)))

	// processInstallment: no UNPAID/LATE installment → early return
	loanMock.ExpectQuery(`SELECT id, retry_count FROM loan_installments`).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}))

	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 5})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Processed)
}
