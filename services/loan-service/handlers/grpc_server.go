package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LoanServer struct {
	pb.UnimplementedLoanServiceServer
	DB         *sql.DB // loan_db
	AccountDB  *sql.DB // account_db
	ExchangeDB *sql.DB // exchange_db (for currency conversion to determine rate tier)
}

// --- GetClientLoans ---

func (s *LoanServer) GetClientLoans(ctx context.Context, req *pb.GetClientLoansRequest) (*pb.GetClientLoansResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, loan_number, account_number, loan_type, amount, currency, status, repayment_period
		FROM loans
		WHERE client_id = $1
		ORDER BY amount DESC`, req.ClientId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query loans: %v", err)
	}
	defer rows.Close()

	var loans []*pb.LoanSummary
	for rows.Next() {
		var l pb.LoanSummary
		if err := rows.Scan(&l.Id, &l.LoanNumber, &l.AccountNumber, &l.LoanType,
			&l.Amount, &l.Currency, &l.Status, &l.RepaymentPeriod); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan loan: %v", err)
		}
		loans = append(loans, &l)
	}
	return &pb.GetClientLoansResponse{Loans: loans}, nil
}

// --- GetLoanDetails ---

func (s *LoanServer) GetLoanDetails(ctx context.Context, req *pb.GetLoanDetailsRequest) (*pb.GetLoanDetailsResponse, error) {
	var l pb.LoanDetail
	var agreedDate, maturityDate time.Time
	var nextDate sql.NullTime
	var nextAmt, remainingDebt sql.NullFloat64

	err := s.DB.QueryRowContext(ctx, `
		SELECT id, loan_number, account_number, loan_type, interest_rate_type,
		       amount, currency, repayment_period, nominal_rate, effective_rate,
		       agreed_date, maturity_date, next_installment_amount,
		       next_installment_date, remaining_debt, status
		FROM loans WHERE id = $1`, req.LoanId,
	).Scan(&l.Id, &l.LoanNumber, &l.AccountNumber, &l.LoanType, &l.InterestRateType,
		&l.Amount, &l.Currency, &l.RepaymentPeriod, &l.NominalRate, &l.EffectiveRate,
		&agreedDate, &maturityDate, &nextAmt, &nextDate, &remainingDebt, &l.Status)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "loan not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query loan: %v", err)
	}

	l.AgreedDate = agreedDate.Format("2006-01-02")
	l.MaturityDate = maturityDate.Format("2006-01-02")
	if nextAmt.Valid {
		l.NextInstallmentAmount = nextAmt.Float64
	}
	if nextDate.Valid {
		l.NextInstallmentDate = nextDate.Time.Format("2006-01-02")
	}
	if remainingDebt.Valid {
		l.RemainingDebt = remainingDebt.Float64
	}

	installments, err := s.queryInstallments(ctx, req.LoanId)
	if err != nil {
		return nil, err
	}
	return &pb.GetLoanDetailsResponse{Loan: &l, Installments: installments}, nil
}

// --- GetLoanInstallments ---

func (s *LoanServer) GetLoanInstallments(ctx context.Context, req *pb.GetLoanInstallmentsRequest) (*pb.GetLoanInstallmentsResponse, error) {
	installments, err := s.queryInstallments(ctx, req.LoanId)
	if err != nil {
		return nil, err
	}
	return &pb.GetLoanInstallmentsResponse{Installments: installments}, nil
}

func (s *LoanServer) queryInstallments(ctx context.Context, loanID int64) ([]*pb.Installment, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, loan_id, installment_amount, interest_rate, currency,
		       expected_due_date, actual_due_date, status
		FROM loan_installments
		WHERE loan_id = $1
		ORDER BY expected_due_date ASC`, loanID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query installments: %v", err)
	}
	defer rows.Close()

	var installments []*pb.Installment
	for rows.Next() {
		var i pb.Installment
		var expected time.Time
		var actual sql.NullTime
		if err := rows.Scan(&i.Id, &i.LoanId, &i.InstallmentAmount, &i.InterestRate,
			&i.Currency, &expected, &actual, &i.Status); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan installment: %v", err)
		}
		i.ExpectedDueDate = expected.Format("2006-01-02")
		if actual.Valid {
			i.ActualDueDate = actual.Time.Format("2006-01-02")
		}
		installments = append(installments, &i)
	}
	return installments, nil
}

// --- SubmitLoanApplication (#100) ---

func (s *LoanServer) SubmitLoanApplication(ctx context.Context, req *pb.SubmitLoanApplicationRequest) (*pb.SubmitLoanApplicationResponse, error) {
	// 1. Validate loan type
	validTypes := map[string]bool{"CASH": true, "HOUSING": true, "AUTO": true, "REFINANCING": true, "STUDENT": true}
	if !validTypes[req.LoanType] {
		return nil, status.Errorf(codes.InvalidArgument, "invalid loan type: %s", req.LoanType)
	}
	if req.InterestRateType != "FIXED" && req.InterestRateType != "VARIABLE" {
		return nil, status.Error(codes.InvalidArgument, "interest_rate_type must be 'FIXED' or 'VARIABLE'")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	if req.RepaymentPeriod <= 0 {
		return nil, status.Error(codes.InvalidArgument, "repayment_period must be positive")
	}

	// 2. Validate repayment period for loan type
	if !validRepaymentPeriods(req.LoanType)[int(req.RepaymentPeriod)] {
		return nil, status.Errorf(codes.InvalidArgument, "invalid repayment_period %d for loan type %s", req.RepaymentPeriod, req.LoanType)
	}

	// 3. Validate account currency matches loan currency
	var accountCurrencyID int64
	if err := s.AccountDB.QueryRowContext(ctx,
		`SELECT currency_id FROM accounts WHERE account_number = $1`, req.AccountNumber,
	).Scan(&accountCurrencyID); err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "account not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load account: %v", err)
	}
	var accountCurrencyCode string
	if err := s.ExchangeDB.QueryRowContext(ctx,
		`SELECT code FROM currencies WHERE id = $1`, accountCurrencyID,
	).Scan(&accountCurrencyCode); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve account currency: %v", err)
	}
	if accountCurrencyCode != req.Currency {
		return nil, status.Errorf(codes.InvalidArgument,
			"account currency (%s) does not match loan currency (%s)", accountCurrencyCode, req.Currency)
	}

	// 4. Convert amount to RSD for rate tier lookup
	amountRSD, err := s.toRSD(ctx, req.Amount, req.Currency)
	if err != nil {
		log.Printf("loan-service: currency conversion failed: %v — using raw amount", err)
		amountRSD = req.Amount
	}

	// 4. Calculate rates
	fixed := req.InterestRateType == "FIXED"
	nominalRate := lookupRateTier(amountRSD, fixed) // base rate
	effectiveRate := effectiveAnnualRate(req.LoanType, amountRSD, fixed, 0)

	// 5. Calculate monthly installment
	installmentAmt := monthlyInstallment(req.Amount, effectiveRate, int(req.RepaymentPeriod))

	// 6. Generate unique loan number
	loanNumber, err := generateLoanNumber()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate loan number: %v", err)
	}

	// 7. Compute dates
	today := time.Now()
	maturityDate := today.AddDate(0, int(req.RepaymentPeriod), 0)
	nextInstallmentDate := today.AddDate(0, 1, 0)

	// 8. Insert loan
	var loanID int64
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO loans (
			loan_number, account_number, client_id, loan_type, interest_rate_type,
			amount, currency, repayment_period, nominal_rate, effective_rate,
			agreed_date, maturity_date, next_installment_amount, next_installment_date,
			remaining_debt, status, purpose, monthly_salary,
			employment_status, employment_period, contact_phone
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, 'PENDING', $16, $17,
			$18, $19, $20
		) RETURNING id`,
		loanNumber, req.AccountNumber, req.ClientId, req.LoanType, req.InterestRateType,
		req.Amount, req.Currency, req.RepaymentPeriod, nominalRate, effectiveRate,
		today.Format("2006-01-02"), maturityDate.Format("2006-01-02"),
		installmentAmt, nextInstallmentDate.Format("2006-01-02"),
		req.Amount, req.Purpose, req.MonthlySalary,
		req.EmploymentStatus, req.EmploymentPeriod, req.ContactPhone,
	).Scan(&loanID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert loan: %v", err)
	}

	return &pb.SubmitLoanApplicationResponse{
		LoanId:             loanID,
		LoanNumber:         loanNumber,
		MonthlyInstallment: installmentAmt,
	}, nil
}

// toRSD converts an amount in the given currency to RSD using today's middle rate from exchange_db.
func (s *LoanServer) toRSD(ctx context.Context, amount float64, currency string) (float64, error) {
	if currency == "RSD" {
		return amount, nil
	}
	var middleRate float64
	err := s.ExchangeDB.QueryRowContext(ctx,
		`SELECT middle_rate FROM daily_exchange_rates WHERE currency_code = $1 AND date = CURRENT_DATE`,
		currency,
	).Scan(&middleRate)
	if err != nil {
		return 0, fmt.Errorf("no exchange rate for %s: %w", currency, err)
	}
	return amount * middleRate, nil
}

// generateLoanNumber generates a random 13-digit loan number.
func generateLoanNumber() (int64, error) {
	// Range: 1_000_000_000_000 to 9_999_999_999_999
	min := int64(1_000_000_000_000)
	max := int64(9_999_999_999_999)
	diff := big.NewInt(max - min)
	n, err := rand.Int(rand.Reader, diff)
	if err != nil {
		return 0, err
	}
	return min + n.Int64(), nil
}

// --- ApproveLoan (#102) ---

func (s *LoanServer) ApproveLoan(ctx context.Context, req *pb.ApproveLoanRequest) (*pb.ApproveLoanResponse, error) {
	// 1. Load loan
	var loanStatus, currency, loanType, interestRateType, accountNumber string
	var amount, effectiveRate float64
	var repaymentPeriod int
	var agreedDate time.Time

	err := s.DB.QueryRowContext(ctx, `
		SELECT status, currency, loan_type, interest_rate_type, account_number,
		       amount, effective_rate, repayment_period, agreed_date
		FROM loans WHERE id = $1`, req.LoanId,
	).Scan(&loanStatus, &currency, &loanType, &interestRateType, &accountNumber,
		&amount, &effectiveRate, &repaymentPeriod, &agreedDate)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "loan not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load loan: %v", err)
	}
	if loanStatus != "PENDING" {
		return nil, status.Errorf(codes.FailedPrecondition, "loan is not in PENDING state")
	}

	// 2. Disburse amount to account
	_, err = s.AccountDB.ExecContext(ctx, `
		UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1
		WHERE account_number = $2`, amount, accountNumber)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to disburse loan: %v", err)
	}

	// 3. Generate installment schedule
	installmentAmt := monthlyInstallment(amount, effectiveRate, repaymentPeriod)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	for i := 1; i <= repaymentPeriod; i++ {
		dueDate := agreedDate.AddDate(0, i, 0)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO loan_installments (loan_id, installment_amount, interest_rate, currency, expected_due_date)
			VALUES ($1, $2, $3, $4, $5)`,
			req.LoanId, installmentAmt, effectiveRate, currency, dueDate.Format("2006-01-02"))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create installment: %v", err)
		}
	}

	nextDate := agreedDate.AddDate(0, 1, 0)
	_, err = tx.ExecContext(ctx, `
		UPDATE loans SET status = 'APPROVED', next_installment_date = $1,
		next_installment_amount = $2, remaining_debt = amount
		WHERE id = $3`,
		nextDate.Format("2006-01-02"), installmentAmt, req.LoanId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to approve loan: %v", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit: %v", err)
	}

	return &pb.ApproveLoanResponse{Success: true}, nil
}

// --- RejectLoan (#102) ---

func (s *LoanServer) RejectLoan(ctx context.Context, req *pb.RejectLoanRequest) (*pb.RejectLoanResponse, error) {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE loans SET status = 'REJECTED' WHERE id = $1 AND status = 'PENDING'`, req.LoanId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reject loan: %v", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, status.Error(codes.NotFound, "loan not found or not in PENDING state")
	}
	return &pb.RejectLoanResponse{Success: true}, nil
}

// --- GetAllLoanApplications (#103) ---

func (s *LoanServer) GetAllLoanApplications(ctx context.Context, req *pb.GetAllLoanApplicationsRequest) (*pb.GetAllLoanApplicationsResponse, error) {
	query := `SELECT id, loan_number, account_number, loan_type, interest_rate_type,
		         amount, currency, repayment_period, nominal_rate, effective_rate,
		         agreed_date, maturity_date, next_installment_amount, next_installment_date,
		         remaining_debt, status, purpose, monthly_salary, employment_status,
		         employment_period, contact_phone
		  FROM loans WHERE status = 'PENDING'`
	args := []any{}
	if req.LoanType != "" {
		args = append(args, req.LoanType)
		query += fmt.Sprintf(" AND loan_type = $%d", len(args))
	}
	if req.AccountNumber != "" {
		args = append(args, req.AccountNumber)
		query += fmt.Sprintf(" AND account_number = $%d", len(args))
	}
	query += " ORDER BY agreed_date ASC"

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query applications: %v", err)
	}
	defer rows.Close()

	loans, err := scanLoanDetails(rows)
	if err != nil {
		return nil, err
	}
	return &pb.GetAllLoanApplicationsResponse{Applications: loans}, nil
}

// --- GetAllLoans (#103) ---

func (s *LoanServer) GetAllLoans(ctx context.Context, req *pb.GetAllLoansRequest) (*pb.GetAllLoansResponse, error) {
	query := `SELECT id, loan_number, account_number, loan_type, interest_rate_type,
		         amount, currency, repayment_period, nominal_rate, effective_rate,
		         agreed_date, maturity_date, next_installment_amount, next_installment_date,
		         remaining_debt, status, purpose, monthly_salary, employment_status,
		         employment_period, contact_phone
		  FROM loans WHERE 1=1`
	args := []any{}
	if req.LoanType != "" {
		args = append(args, req.LoanType)
		query += fmt.Sprintf(" AND loan_type = $%d", len(args))
	}
	if req.AccountNumber != "" {
		args = append(args, req.AccountNumber)
		query += fmt.Sprintf(" AND account_number = $%d", len(args))
	}
	if req.Status != "" {
		args = append(args, req.Status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	query += " ORDER BY account_number ASC"

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query loans: %v", err)
	}
	defer rows.Close()

	loans, err := scanLoanDetails(rows)
	if err != nil {
		return nil, err
	}
	return &pb.GetAllLoansResponse{Loans: loans}, nil
}

func scanLoanDetails(rows *sql.Rows) ([]*pb.LoanDetail, error) {
	var loans []*pb.LoanDetail
	for rows.Next() {
		var l pb.LoanDetail
		var agreedDate, maturityDate time.Time
		var nextDate sql.NullTime
		var nextAmt, remainingDebt, monthlySalary sql.NullFloat64
		var purpose, employmentStatus, contactPhone sql.NullString
		var employmentPeriod sql.NullInt32
		if err := rows.Scan(&l.Id, &l.LoanNumber, &l.AccountNumber, &l.LoanType, &l.InterestRateType,
			&l.Amount, &l.Currency, &l.RepaymentPeriod, &l.NominalRate, &l.EffectiveRate,
			&agreedDate, &maturityDate, &nextAmt, &nextDate, &remainingDebt, &l.Status,
			&purpose, &monthlySalary, &employmentStatus, &employmentPeriod, &contactPhone); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan loan: %v", err)
		}
		l.AgreedDate = agreedDate.Format("2006-01-02")
		l.MaturityDate = maturityDate.Format("2006-01-02")
		if nextAmt.Valid {
			l.NextInstallmentAmount = nextAmt.Float64
		}
		if nextDate.Valid {
			l.NextInstallmentDate = nextDate.Time.Format("2006-01-02")
		}
		if remainingDebt.Valid {
			l.RemainingDebt = remainingDebt.Float64
		}
		if purpose.Valid {
			l.Purpose = purpose.String
		}
		if monthlySalary.Valid {
			l.MonthlySalary = monthlySalary.Float64
		}
		if employmentStatus.Valid {
			l.EmploymentStatus = employmentStatus.String
		}
		if employmentPeriod.Valid {
			l.EmploymentPeriod = employmentPeriod.Int32
		}
		if contactPhone.Valid {
			l.ContactPhone = contactPhone.String
		}
		loans = append(loans, &l)
	}
	return loans, nil
}
