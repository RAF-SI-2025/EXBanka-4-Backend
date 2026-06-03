package handlers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ── SubmitLoanApplication: repayment_period <= 0 (line 149-151) ─────────────

func TestSubmitLoanApplication_ZeroRepaymentPeriod(t *testing.T) {
	s, _, _ := newLoanServer(t)
	_, err := s.SubmitLoanApplication(context.Background(), &pb.SubmitLoanApplicationRequest{
		LoanType: "CASH", InterestRateType: "FIXED", Amount: 100000, RepaymentPeriod: 0,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// ── GetLoanDetails: queryInstallments error (line 89-91) ─────────────────────

func TestGetLoanDetails_InstallmentsError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number, account_number").
		WillReturnRows(sqlmock.NewRows(loanDetailColumns()).
			AddRow(
				int64(1), int64(1234567890123), "ACC001", "CASH", "FIXED",
				float64(100000), "RSD", int32(24), float64(6.25), float64(7.00),
				time.Now(), time.Now().AddDate(2, 0, 0),
				sql.NullFloat64{Float64: 4500, Valid: true},
				sql.NullTime{Time: time.Now().AddDate(0, 1, 0), Valid: true},
				sql.NullFloat64{Float64: 95500, Valid: true},
				"APPROVED",
			))
	// installments query fails
	loanMock.ExpectQuery("SELECT id, loan_id, installment_amount").
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetLoanDetails(context.Background(), &pb.GetLoanDetailsRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── queryInstallments: actual_due_date non-null (line 127-129) ───────────────

func TestGetLoanInstallments_WithActualDate(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	actualDate := time.Now()
	loanMock.ExpectQuery("SELECT id, loan_id, installment_amount").
		WillReturnRows(sqlmock.NewRows(installmentColumns()).
			AddRow(
				int64(1), int64(1), float64(4500), float64(7.0), "RSD",
				time.Now().AddDate(0, -1, 0),                // expected
				sql.NullTime{Time: actualDate, Valid: true}, // actual non-null
				"PAID",
			))

	resp, err := s.GetLoanInstallments(context.Background(), &pb.GetLoanInstallmentsRequest{LoanId: 1})
	require.NoError(t, err)
	require.Len(t, resp.Installments, 1)
	assert.NotEmpty(t, resp.Installments[0].ActualDueDate)
}

// ── ApproveLoan: exchange currency resolution error (line 295-297) ───────────

func TestApproveLoan_CurrencyResolutionError(t *testing.T) {
	s, loanMock, _, exchangeMock := newLoanServerWithExchange(t)
	loanMock.ExpectQuery("SELECT status, currency").
		WillReturnRows(sqlmock.NewRows([]string{
			"status", "currency", "loan_type", "interest_rate_type",
			"account_number", "amount", "effective_rate", "repayment_period", "agreed_date",
		}).AddRow("PENDING", "EUR", "CASH", "FIXED", "ACC001", float64(10000), float64(6.5), int(12), time.Now()))
	exchangeMock.ExpectQuery("SELECT id FROM currencies").WillReturnError(sql.ErrConnDone)

	_, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── ApproveLoan: bank account not found (line 301-303) ───────────────────────

func TestApproveLoan_BankAccountNotFound(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)
	loanMock.ExpectQuery("SELECT status, currency").
		WillReturnRows(sqlmock.NewRows([]string{
			"status", "currency", "loan_type", "interest_rate_type",
			"account_number", "amount", "effective_rate", "repayment_period", "agreed_date",
		}).AddRow("PENDING", "RSD", "CASH", "FIXED", "ACC001", float64(100000), float64(6.5), int(12), time.Now()))
	exchangeMock.ExpectQuery("SELECT id FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery("SELECT account_number FROM accounts").WillReturnError(sql.ErrNoRows)

	_, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── ApproveLoan: credit client account error (line 315-317) ─────────────────

func TestApproveLoan_CreditClientError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)
	loanMock.ExpectQuery("SELECT status, currency").
		WillReturnRows(sqlmock.NewRows([]string{
			"status", "currency", "loan_type", "interest_rate_type",
			"account_number", "amount", "effective_rate", "repayment_period", "agreed_date",
		}).AddRow("PENDING", "RSD", "CASH", "FIXED", "ACC001", float64(100000), float64(6.5), int(12), time.Now()))
	exchangeMock.ExpectQuery("SELECT id FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))
	accountMock.ExpectExec("UPDATE accounts SET balance").
		WillReturnResult(sqlmock.NewResult(1, 1)) // debit bank succeeds
	accountMock.ExpectExec("UPDATE accounts SET balance").
		WillReturnError(sql.ErrConnDone) // credit client fails

	_, err := s.ApproveLoan(context.Background(), &pb.ApproveLoanRequest{LoanId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetAllLoanApplications: scan error (line 396-398 via 451-453) ────────────

func TestGetAllLoanApplications_ScanError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").
		WillReturnRows(sqlmock.NewRows(loanFullColumns()).
			AddRow(
				"bad-id", int64(1234567890123), "ACC001", "CASH", "FIXED",
				float64(100000), "RSD", int32(24), float64(6.25), float64(7.0),
				time.Now(), time.Now().AddDate(2, 0, 0),
				sql.NullFloat64{Valid: false}, sql.NullTime{Valid: false},
				sql.NullFloat64{Valid: false}, "PENDING",
				sql.NullString{Valid: false}, sql.NullFloat64{Valid: false},
				sql.NullString{Valid: false}, sql.NullInt32{Valid: false},
				sql.NullString{Valid: false},
			))

	_, err := s.GetAllLoanApplications(context.Background(), &pb.GetAllLoanApplicationsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetAllLoans: scan error (line 433-435) ───────────────────────────────────

func TestGetAllLoans_ScanError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number").
		WillReturnRows(sqlmock.NewRows(loanFullColumns()).
			AddRow(
				"bad-id", int64(1234567890123), "ACC001", "CASH", "FIXED",
				float64(100000), "RSD", int32(24), float64(6.25), float64(7.0),
				time.Now(), time.Now().AddDate(2, 0, 0),
				sql.NullFloat64{Valid: false}, sql.NullTime{Valid: false},
				sql.NullFloat64{Valid: false}, "APPROVED",
				sql.NullString{Valid: false}, sql.NullFloat64{Valid: false},
				sql.NullString{Valid: false}, sql.NullInt32{Valid: false},
				sql.NullString{Valid: false},
			))

	_, err := s.GetAllLoans(context.Background(), &pb.GetAllLoansRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── collectInstallments: query error (line 65-68) ────────────────────────────

func TestCollectInstallments_QueryError(t *testing.T) {
	s, loanMock, _ := newLoanServer(t)
	loanMock.ExpectQuery("SELECT id, loan_number, client_id").
		WithArgs(int64(42)).
		WillReturnError(sql.ErrConnDone)

	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 42})
	require.NoError(t, err)
	assert.Equal(t, int32(0), resp.Processed)
}

// ── processInstallment: currency resolution error (line 123-126) ─────────────

func TestProcessInstallment_CurrencyResolutionError(t *testing.T) {
	s, loanMock, _, exchangeMock := newLoanServerWithExchange(t)

	loanMock.ExpectQuery("SELECT id, loan_number, client_id").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_number", "client_id", "account_number",
			"next_installment_amount", "currency", "remaining_debt",
		}).AddRow(int64(1), int64(1234567890123), int64(5), "ACC001", float64(4500), "EUR", float64(90000)))

	loanMock.ExpectQuery("SELECT id, retry_count FROM loan_installments").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(10), int(0)))

	exchangeMock.ExpectQuery("SELECT id FROM currencies").WillReturnError(sql.ErrConnDone)

	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 1})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Processed)
}

// ── processInstallment: bank account not found (line 130-133) ────────────────

func TestProcessInstallment_BankAccountNotFound(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	loanMock.ExpectQuery("SELECT id, loan_number, client_id").
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_number", "client_id", "account_number",
			"next_installment_amount", "currency", "remaining_debt",
		}).AddRow(int64(2), int64(1234567890123), int64(5), "ACC001", float64(4500), "EUR", float64(90000)))

	loanMock.ExpectQuery("SELECT id, retry_count FROM loan_installments").
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(20), int(0)))

	exchangeMock.ExpectQuery("SELECT id FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(3)))

	accountMock.ExpectQuery("SELECT account_number FROM accounts").WillReturnError(sql.ErrNoRows)

	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 2})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Processed)
}

// ── processInstallment: credit bank account error (non-fatal, line 151-153) ──
// Also covers the PAID_OFF branch (line 156-158) by making remaining debt = 0.

func TestProcessInstallment_CreditBankError_PaidOff(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	// remaining_debt == amount → newRemaining = 0 → PAID_OFF
	loanMock.ExpectQuery("SELECT id, loan_number, client_id").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_number", "client_id", "account_number",
			"next_installment_amount", "currency", "remaining_debt",
		}).AddRow(int64(3), int64(1234567890123), int64(5), "ACC001", float64(4500), "RSD", float64(4500)))

	loanMock.ExpectQuery("SELECT id, retry_count FROM loan_installments").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(30), int(0)))

	exchangeMock.ExpectQuery("SELECT id FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))

	// Debit client account succeeds (affected = 1)
	accountMock.ExpectExec("UPDATE accounts").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Credit bank account FAILS (non-fatal)
	accountMock.ExpectExec("UPDATE accounts").
		WillReturnError(sql.ErrConnDone)

	// Mark installment PAID
	loanMock.ExpectExec("UPDATE loan_installments").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Advance schedule (status = 'PAID_OFF' since newRemaining = 0)
	loanMock.ExpectExec("UPDATE loans SET").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 3})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Processed)
}

// ── processInstallment: advance schedule error (non-fatal, line 177-179) ─────

func TestProcessInstallment_AdvanceScheduleError(t *testing.T) {
	s, loanMock, accountMock, exchangeMock := newLoanServerWithExchange(t)

	loanMock.ExpectQuery("SELECT id, loan_number, client_id").
		WithArgs(int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "loan_number", "client_id", "account_number",
			"next_installment_amount", "currency", "remaining_debt",
		}).AddRow(int64(4), int64(1234567890123), int64(5), "ACC001", float64(4500), "RSD", float64(90000)))

	loanMock.ExpectQuery("SELECT id, retry_count FROM loan_installments").
		WithArgs(int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "retry_count"}).AddRow(int64(40), int(0)))

	exchangeMock.ExpectQuery("SELECT id FROM currencies").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	accountMock.ExpectQuery("SELECT account_number FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"account_number"}).AddRow("BANK001"))

	accountMock.ExpectExec("UPDATE accounts").WillReturnResult(sqlmock.NewResult(1, 1)) // debit
	accountMock.ExpectExec("UPDATE accounts").WillReturnResult(sqlmock.NewResult(1, 1)) // credit bank

	loanMock.ExpectExec("UPDATE loan_installments").WillReturnResult(sqlmock.NewResult(1, 1)) // mark PAID

	// Advance schedule fails (non-fatal)
	loanMock.ExpectExec("UPDATE loans SET").WillReturnError(sql.ErrConnDone)

	resp, err := s.TriggerInstallments(context.Background(), &pb.TriggerInstallmentsRequest{ForceLoanId: 4})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Processed)
}
