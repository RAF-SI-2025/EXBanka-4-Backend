package handlers

import (
	"context"
	"database/sql"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	pb_email "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/account-service/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AccountServer struct {
	pb.UnimplementedAccountServiceServer
	DB          *sql.DB
	ClientDB    *sql.DB
	ExchangeDB  *sql.DB
	EmailClient pb_email.EmailServiceClient
}

func (s *AccountServer) GetMyAccounts(ctx context.Context, req *pb.GetMyAccountsRequest) (*pb.GetMyAccountsResponse, error) {
	// 1. Fetch accounts from account_db
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, account_name, account_number, available_balance, currency_id
		 FROM accounts WHERE owner_id = $1
		 ORDER BY available_balance DESC`,
		req.OwnerId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query accounts: %v", err)
	}
	defer rows.Close()

	type row struct {
		id               int64
		accountName      string
		accountNumber    string
		availableBalance float64
		currencyID       int64
	}
	var accs []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.accountName, &r.accountNumber, &r.availableBalance, &r.currencyID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan account: %v", err)
		}
		accs = append(accs, r)
	}

	// 2. Build currency_id → code map from exchange_db
	currencyMap := map[int64]string{}
	for _, a := range accs {
		if _, ok := currencyMap[a.currencyID]; !ok {
			var code string
			if err := s.ExchangeDB.QueryRowContext(ctx,
				`SELECT code FROM currencies WHERE id = $1`, a.currencyID,
			).Scan(&code); err == nil {
				currencyMap[a.currencyID] = code
			}
		}
	}

	// 3. Assemble response
	summaries := make([]*pb.AccountSummary, 0, len(accs))
	for _, a := range accs {
		summaries = append(summaries, &pb.AccountSummary{
			Id:               a.id,
			AccountName:      a.accountName,
			AccountNumber:    a.accountNumber,
			AvailableBalance: a.availableBalance,
			CurrencyCode:     currencyMap[a.currencyID],
		})
	}
	return &pb.GetMyAccountsResponse{Accounts: summaries}, nil
}

func (s *AccountServer) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	var a pb.AccountDetails
	var currencyID int64
	var ownerID int64
	var companyID sql.NullInt64
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, account_name, account_number, owner_id, balance, available_balance,
		       balance - available_balance AS reserved_funds,
		       currency_id, status, account_type, COALESCE(account_subtype, ''),
		       COALESCE(daily_limit, 0), COALESCE(monthly_limit, 0),
		       daily_spent, monthly_spent, company_id
		FROM accounts WHERE id = $1`, req.AccountId,
	).Scan(&a.Id, &a.AccountName, &a.AccountNumber, &ownerID,
		&a.Balance, &a.AvailableBalance, &a.ReservedFunds,
		&currencyID, &a.Status, &a.AccountType, &a.AccountSubtype,
		&a.DailyLimit, &a.MonthlyLimit, &a.DailySpent, &a.MonthlySpent, &companyID)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "account %d not found", req.AccountId)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query account: %v", err)
	}
	if req.OwnerId != 0 && ownerID != req.OwnerId {
		return nil, status.Errorf(codes.PermissionDenied, "account does not belong to this user")
	}

	// Resolve currency code from exchange_db
	_ = s.ExchangeDB.QueryRowContext(ctx, `SELECT code FROM currencies WHERE id = $1`, currencyID).Scan(&a.CurrencyCode)

	// Resolve owner name from client_db
	var firstName, lastName string
	if err := s.ClientDB.QueryRowContext(ctx,
		`SELECT first_name, last_name FROM clients WHERE id = $1`, ownerID,
	).Scan(&firstName, &lastName); err == nil {
		a.Owner = firstName + " " + lastName
	}

	// Resolve company data for business accounts
	if companyID.Valid {
		var c pb.CompanyData
		if err := s.DB.QueryRowContext(ctx,
			`SELECT name, registration_number, pib, activity_code, address FROM companies WHERE id = $1`,
			companyID.Int64,
		).Scan(&c.Name, &c.RegistrationNumber, &c.Pib, &c.ActivityCode, &c.Address); err == nil {
			a.CompanyData = &c
		}
	}

	return &pb.GetAccountResponse{Account: &a}, nil
}

func (s *AccountServer) RenameAccount(ctx context.Context, req *pb.RenameAccountRequest) (*pb.RenameAccountResponse, error) {
	// Verify ownership and get current name
	var currentName string
	var ownerID int64
	err := s.DB.QueryRowContext(ctx,
		`SELECT account_name, owner_id FROM accounts WHERE id = $1`, req.AccountId,
	).Scan(&currentName, &ownerID)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "account %d not found", req.AccountId)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query account: %v", err)
	}
	if ownerID != req.OwnerId {
		return nil, status.Errorf(codes.PermissionDenied, "account does not belong to this user")
	}
	if req.NewName == currentName {
		return nil, status.Errorf(codes.InvalidArgument, "new name must differ from current name")
	}

	// Check no other account of this owner has the same name
	var conflict int
	_ = s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM accounts WHERE owner_id = $1 AND account_name = $2 AND id != $3`,
		req.OwnerId, req.NewName, req.AccountId,
	).Scan(&conflict)
	if conflict > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "another account with this name already exists")
	}

	_, err = s.DB.ExecContext(ctx,
		`UPDATE accounts SET account_name = $1 WHERE id = $2`, req.NewName, req.AccountId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to rename account: %v", err)
	}
	return &pb.RenameAccountResponse{}, nil
}

func (s *AccountServer) GetAllAccounts(ctx context.Context, _ *pb.GetAllAccountsRequest) (*pb.GetAllAccountsResponse, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, account_number, account_name, owner_id, account_type, currency_id, available_balance
		 FROM accounts
		 WHERE account_type != 'BANK'
		 ORDER BY id DESC`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query accounts: %v", err)
	}
	defer rows.Close()

	type row struct {
		id               int64
		accountNumber    string
		accountName      string
		ownerID          int64
		accountType      string
		currencyID       int64
		availableBalance float64
	}
	var accs []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.accountNumber, &r.accountName, &r.ownerID, &r.accountType, &r.currencyID, &r.availableBalance); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan account: %v", err)
		}
		accs = append(accs, r)
	}

	// Build currency_id → code map
	currencyMap := map[int64]string{}
	for _, a := range accs {
		if _, ok := currencyMap[a.currencyID]; !ok {
			var code string
			if err := s.ExchangeDB.QueryRowContext(ctx,
				`SELECT code FROM currencies WHERE id = $1`, a.currencyID,
			).Scan(&code); err == nil {
				currencyMap[a.currencyID] = code
			}
		}
	}

	// Build owner_id → name map
	ownerMap := map[int64][2]string{}
	for _, a := range accs {
		if _, ok := ownerMap[a.ownerID]; !ok {
			var firstName, lastName string
			if err := s.ClientDB.QueryRowContext(ctx,
				`SELECT first_name, last_name FROM clients WHERE id = $1`, a.ownerID,
			).Scan(&firstName, &lastName); err == nil {
				ownerMap[a.ownerID] = [2]string{firstName, lastName}
			}
		}
	}

	items := make([]*pb.AccountListItem, 0, len(accs))
	for _, a := range accs {
		owner := ownerMap[a.ownerID]
		items = append(items, &pb.AccountListItem{
			Id:               a.id,
			AccountNumber:    a.accountNumber,
			AccountName:      a.accountName,
			OwnerId:          a.ownerID,
			OwnerFirstName:   owner[0],
			OwnerLastName:    owner[1],
			AccountType:      a.accountType,
			CurrencyCode:     currencyMap[a.currencyID],
			AvailableBalance: a.availableBalance,
		})
	}
	return &pb.GetAllAccountsResponse{Accounts: items}, nil
}

// accountTypeCode maps account type string to 2-digit code used in account number generation.
func accountTypeCode(accountType string) string {
	switch accountType {
	case "CURRENT":
		return "01"
	case "SAVINGS":
		return "02"
	case "FOREIGN_CURRENCY":
		return "03"
	case "BUSINESS":
		return "04"
	default:
		return "00"
	}
}

func (s *AccountServer) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	// 1. Validate client exists and fetch contact info for email
	var clientID int64
	var clientEmail, clientFirstName string
	err := s.ClientDB.QueryRowContext(ctx,
		`SELECT id, email, first_name FROM clients WHERE id = $1`, req.ClientId).
		Scan(&clientID, &clientEmail, &clientFirstName)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "client with id %d not found", req.ClientId)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify client: %v", err)
	}

	// 2. Validate currency exists and get its id
	var currencyID int64
	var currencyCode string
	err = s.ExchangeDB.QueryRowContext(ctx,
		`SELECT id, code FROM currencies WHERE code = $1`, req.CurrencyCode).Scan(&currencyID, &currencyCode)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "currency with code %q not found", req.CurrencyCode)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify currency: %v", err)
	}

	// 3. Generate account number
	accountNumber := utils.GenerateAccountNumber("265", "0001", accountTypeCode(req.AccountType))

	// 4. Set expiration date 5 years from now
	expirationDate := time.Now().AddDate(5, 0, 0).Format("2006-01-02")

	// 5. Resolve company for BUSINESS accounts
	var companyID *int64
	if req.AccountType == "BUSINESS" && req.CompanyData != nil && req.CompanyData.Name != "" {
		var cid int64
		err = s.DB.QueryRowContext(ctx,
			`SELECT id FROM companies WHERE registration_number = $1`,
			req.CompanyData.RegistrationNumber,
		).Scan(&cid)
		if err == sql.ErrNoRows {
			// Company doesn't exist yet – create it
			err = s.DB.QueryRowContext(ctx,
				`INSERT INTO companies
					(name, registration_number, pib, activity_code, address, owner_client_id)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING id`,
				req.CompanyData.Name,
				req.CompanyData.RegistrationNumber,
				req.CompanyData.Pib,
				req.CompanyData.ActivityCode,
				req.CompanyData.Address,
				req.ClientId,
			).Scan(&cid)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create company: %v", err)
			}
		} else if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to look up company: %v", err)
		}
		companyID = &cid
	}

	// 6. Insert account
	var accountID int64
	var createdDate string
	err = s.DB.QueryRowContext(ctx,
		`INSERT INTO accounts
			(account_number, account_name, owner_id, employee_id, currency_id,
			 account_type, account_subtype, status, balance, available_balance, expiration_date, company_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'ACTIVE', $8, $8, $9, $10)
		RETURNING id, created_date`,
		accountNumber,
		req.AccountName,
		req.ClientId,
		req.EmployeeId,
		currencyID,
		req.AccountType,
		req.AccountSubtype,
		req.InitialBalance,
		expirationDate,
		companyID,
	).Scan(&accountID, &createdDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
	}

	// 7. Send account created email (non-blocking – log on failure)
	if s.EmailClient != nil {
		_, emailErr := s.EmailClient.SendAccountCreatedEmail(ctx, &pb_email.SendAccountCreatedEmailRequest{
			Email:         clientEmail,
			FirstName:     clientFirstName,
			AccountName:   req.AccountName,
			AccountNumber: accountNumber,
			CurrencyCode:  currencyCode,
		})
		if emailErr != nil {
			// log but don't fail the request
			_ = emailErr
		}
	}

	return &pb.CreateAccountResponse{
		Account: &pb.AccountResponse{
			Id:               accountID,
			AccountNumber:    accountNumber,
			AccountName:      req.AccountName,
			OwnerId:          req.ClientId,
			EmployeeId:       req.EmployeeId,
			CurrencyCode:     currencyCode,
			AccountType:      req.AccountType,
			Status:           "ACTIVE",
			Balance:          req.InitialBalance,
			AvailableBalance: req.InitialBalance,
			CreatedDate:      createdDate,
		},
	}, nil
}

func (s *AccountServer) UpdateAccountLimits(ctx context.Context, req *pb.UpdateAccountLimitsRequest) (*pb.UpdateAccountLimitsResponse, error) {
	var res sql.Result
	var err error
	if req.OwnerId == 0 {
		// Employee / admin call — no ownership check
		res, err = s.DB.ExecContext(ctx,
			`UPDATE accounts SET daily_limit = $1, monthly_limit = $2 WHERE id = $3`,
			req.DailyLimit, req.MonthlyLimit, req.AccountId,
		)
	} else {
		res, err = s.DB.ExecContext(ctx,
			`UPDATE accounts SET daily_limit = $1, monthly_limit = $2 WHERE id = $3 AND owner_id = $4`,
			req.DailyLimit, req.MonthlyLimit, req.AccountId, req.OwnerId,
		)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update limits: %v", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "account not found or access denied")
	}
	return &pb.UpdateAccountLimitsResponse{}, nil
}
