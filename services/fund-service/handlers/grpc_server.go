package handlers

import (
	"context"
	"database/sql"
	"math"
	"strings"
	"time"

	pb_account "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	pb_exchange "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/exchange"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	pb_order "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_securities "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FundServer struct {
	pb.UnimplementedFundServiceServer
	DB               *sql.DB // fund_db
	AccountDB        *sql.DB // account_db
	EmployeeDB       *sql.DB // employee_db
	ExchangeDB       *sql.DB // exchange_db (currency code lookup)
	AccountClient    pb_account.AccountServiceClient
	OrderClient      pb_order.OrderServiceClient
	ExchangeClient   pb_exchange.ExchangeServiceClient
	SecuritiesClient pb_securities.SecuritiesServiceClient
	PortfolioClient  pb_portfolio.PortfolioServiceClient
}

func (s *FundServer) Ping(_ context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "fund-service ok"}, nil
}

func (s *FundServer) CreateFund(ctx context.Context, req *pb.CreateFundRequest) (*pb.FundResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.ManagerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "manager_id is required")
	}

	// Validate that the manager is an active supervisor
	var permStr string
	err := s.EmployeeDB.QueryRowContext(ctx,
		`SELECT array_to_string(permissions, ',') FROM employees WHERE id = $1 AND active = true`,
		req.ManagerId,
	).Scan(&permStr)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.InvalidArgument, "manager not found or inactive")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify manager: %v", err)
	}
	upper := strings.ToUpper(permStr)
	if !strings.Contains(upper, "SUPERVISOR") && !strings.Contains(upper, "ADMIN") {
		return nil, status.Error(codes.PermissionDenied, "fund manager must be a supervisor or admin")
	}

	// Create a bank account for this fund
	accountResp, err := s.AccountClient.CreateAccount(ctx, &pb_account.CreateAccountRequest{
		ClientId:       0,
		AccountType:    "BANK",
		AccountSubtype: "FUND",
		CurrencyCode:   "RSD",
		InitialBalance: 0,
		AccountName:    "Fund: " + req.Name,
		CreateCard:     false,
		EmployeeId:     req.CreatedById,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create fund account: %v", err)
	}

	accountID := accountResp.GetAccount().GetId()

	var id int64
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO investment_funds (name, description, minimum_contribution, manager_id, account_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		req.Name, req.Description, req.MinimumContribution, req.ManagerId, accountID,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, status.Errorf(codes.AlreadyExists, "fund with name %q already exists", req.Name)
		}
		return nil, status.Errorf(codes.Internal, "failed to create fund: %v", err)
	}

	return s.fetchFundByID(ctx, id, true)
}

func (s *FundServer) ListFunds(ctx context.Context, req *pb.ListFundsRequest) (*pb.ListFundsResponse, error) {
	query := `SELECT id FROM investment_funds WHERE active = true`
	args := []interface{}{}

	if req.ManagerIdFilter != 0 {
		args = append(args, req.ManagerIdFilter)
		query += ` AND manager_id = $1`
	}
	query += ` ORDER BY name ASC`

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list funds: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan fund id: %v", err)
		}
		ids = append(ids, id)
	}

	funds := make([]*pb.FundResponse, 0, len(ids))
	for _, id := range ids {
		f, err := s.fetchFundByID(ctx, id, false)
		if err != nil {
			return nil, err
		}
		funds = append(funds, f)
	}
	return &pb.ListFundsResponse{Funds: funds}, nil
}

func (s *FundServer) GetFund(ctx context.Context, req *pb.GetFundRequest) (*pb.FundResponse, error) {
	return s.fetchFundByID(ctx, req.Id, true)
}

func (s *FundServer) UpdateFund(ctx context.Context, req *pb.UpdateFundRequest) (*pb.FundResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	_, err := s.DB.ExecContext(ctx, `
		UPDATE investment_funds
		SET name = $1, description = $2, minimum_contribution = $3, manager_id = $4
		WHERE id = $5 AND active = true`,
		req.Name, req.Description, req.MinimumContribution, req.ManagerId, req.Id,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, status.Errorf(codes.AlreadyExists, "fund with name %q already exists", req.Name)
		}
		return nil, status.Errorf(codes.Internal, "failed to update fund: %v", err)
	}

	return s.fetchFundByID(ctx, req.Id, true)
}

func (s *FundServer) DeleteFund(ctx context.Context, req *pb.DeleteFundRequest) (*pb.DeleteFundResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Check for active positions
	var count int64
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM client_fund_positions
		WHERE fund_id = $1 AND total_invested_amount > 0`, req.Id,
	).Scan(&count)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check fund positions: %v", err)
	}
	if count > 0 {
		return nil, status.Error(codes.PermissionDenied, "cannot delete fund with active client positions")
	}

	res, err := s.DB.ExecContext(ctx, `UPDATE investment_funds SET active = false WHERE id = $1 AND active = true`, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete fund: %v", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, status.Error(codes.NotFound, "fund not found")
	}

	return &pb.DeleteFundResponse{}, nil
}

func (s *FundServer) fetchFundByID(ctx context.Context, id int64, includeAccountNumber bool) (*pb.FundResponse, error) {
	var f pb.FundResponse
	var description sql.NullString
	var accountID sql.NullInt64
	var createdAt time.Time

	err := s.DB.QueryRowContext(ctx, `
		SELECT id, name, description, minimum_contribution, manager_id,
		       liquid_assets, account_id, created_at, active
		FROM investment_funds WHERE id = $1`, id,
	).Scan(
		&f.Id, &f.Name, &description, &f.MinimumContribution, &f.ManagerId,
		&f.LiquidAssets, &accountID, &createdAt, &f.Active,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "fund not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch fund: %v", err)
	}

	if description.Valid {
		f.Description = description.String
	}
	if accountID.Valid {
		f.AccountId = accountID.Int64
	}
	f.CreatedAt = createdAt.Format(time.RFC3339)

	portfolioRows, pErr := s.DB.QueryContext(ctx,
		`SELECT listing_id, quantity FROM fund_portfolio_positions WHERE fund_id = $1 AND quantity > 0`, id)
	var portfolioValue float64
	if pErr == nil {
		defer func() { _ = portfolioRows.Close() }()
		for portfolioRows.Next() {
			var listingID int64
			var qty float64
			if err := portfolioRows.Scan(&listingID, &qty); err != nil {
				continue
			}
			resp, err := s.SecuritiesClient.GetListingById(ctx, &pb_securities.GetListingByIdRequest{Id: listingID})
			if err != nil || resp.Summary == nil {
				continue
			}
			priceRSD := s.convertViaExchange(ctx, resp.Summary.Currency, "RSD", resp.Summary.Price)
			portfolioValue += qty * priceRSD
		}
	}
	f.FundValue = f.LiquidAssets + portfolioValue

	// profit = fund_value - total invested
	var totalInvested float64
	_ = s.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_invested_amount), 0)
		FROM client_fund_positions WHERE fund_id = $1`, id,
	).Scan(&totalInvested)
	f.Profit = f.FundValue - totalInvested

	// manager name
	if f.ManagerId != 0 {
		var managerName string
		_ = s.EmployeeDB.QueryRowContext(ctx,
			`SELECT first_name || ' ' || last_name FROM employees WHERE id = $1`, f.ManagerId,
		).Scan(&managerName)
		f.ManagerName = managerName
	}

	// account number (only needed for GetFund, not list)
	if includeAccountNumber && f.AccountId != 0 {
		var accountNumber string
		_ = s.AccountDB.QueryRowContext(ctx,
			`SELECT account_number FROM accounts WHERE id = $1`, f.AccountId,
		).Scan(&accountNumber)
		f.AccountNumber = accountNumber
	}

	f.Stats = s.calculateFundStats(ctx, id)
	return &f, nil
}

func (s *FundServer) InvestFund(ctx context.Context, req *pb.InvestFundRequest) (*pb.FundResponse, error) {
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than 0")
	}

	var minimumContribution float64
	var active bool
	var fundAccountID int64
	err := s.DB.QueryRowContext(ctx,
		`SELECT minimum_contribution, active, account_id FROM investment_funds WHERE id = $1`, req.FundId,
	).Scan(&minimumContribution, &active, &fundAccountID)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "fund not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch fund: %v", err)
	}
	if !active {
		return nil, status.Error(codes.NotFound, "fund not found")
	}
	if req.Amount < minimumContribution {
		return nil, status.Errorf(codes.InvalidArgument, "amount %.2f is below minimum contribution %.2f", req.Amount, minimumContribution)
	}

	var availableBalance float64
	err = s.AccountDB.QueryRowContext(ctx,
		`SELECT available_balance FROM accounts WHERE id = $1`, req.SourceAccountId,
	).Scan(&availableBalance)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "source account not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch account: %v", err)
	}
	srcCurrency := s.getAccountCurrencyCode(ctx, req.SourceAccountId)
	debitAmount := s.convertViaExchange(ctx, "RSD", srcCurrency, req.Amount)

	if availableBalance < debitAmount {
		return nil, status.Error(codes.FailedPrecondition, "insufficient account balance")
	}

	_, err = s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
		debitAmount, req.SourceAccountId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to debit account: %v", err)
	}

	_, err = s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
		req.Amount, fundAccountID,
	)
	if err != nil {
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			debitAmount, req.SourceAccountId,
		)
		return nil, status.Errorf(codes.Internal, "failed to credit fund account: %v", err)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		// compensate
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			debitAmount, req.SourceAccountId,
		)
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE investment_funds SET liquid_assets = liquid_assets + $1 WHERE id = $2`,
		req.Amount, req.FundId,
	)
	if err != nil {
		_ = tx.Rollback()
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			debitAmount, req.SourceAccountId,
		)
		return nil, status.Errorf(codes.Internal, "failed to update fund liquid assets: %v", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO client_fund_positions (client_id, client_type, fund_id, total_invested_amount)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (client_id, client_type, fund_id) DO UPDATE
		SET total_invested_amount = client_fund_positions.total_invested_amount + EXCLUDED.total_invested_amount,
		    last_modified_at = NOW()`,
		req.ClientId, req.ClientType, req.FundId, req.Amount,
	)
	if err != nil {
		_ = tx.Rollback()
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			debitAmount, req.SourceAccountId,
		)
		return nil, status.Errorf(codes.Internal, "failed to upsert position: %v", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO client_fund_transactions (client_id, client_type, fund_id, amount, is_inflow, status) VALUES ($1, $2, $3, $4, true, 'COMPLETED')`,
		req.ClientId, req.ClientType, req.FundId, req.Amount,
	)
	if err != nil {
		_ = tx.Rollback()
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			debitAmount, req.SourceAccountId,
		)
		return nil, status.Errorf(codes.Internal, "failed to insert transaction: %v", err)
	}

	if err = tx.Commit(); err != nil {
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			debitAmount, req.SourceAccountId,
		)
		return nil, status.Errorf(codes.Internal, "failed to commit: %v", err)
	}

	return s.fetchFundByID(ctx, req.FundId, true)
}

func (s *FundServer) WithdrawFund(ctx context.Context, req *pb.WithdrawFundRequest) (*pb.WithdrawFundResponse, error) {
	var liquidAssets float64
	var active bool
	var fundAccountID int64
	err := s.DB.QueryRowContext(ctx,
		`SELECT liquid_assets, active, account_id FROM investment_funds WHERE id = $1`, req.FundId,
	).Scan(&liquidAssets, &active, &fundAccountID)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "fund not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch fund: %v", err)
	}
	if !active {
		return nil, status.Error(codes.NotFound, "fund not found")
	}

	var positionAmount float64
	err = s.DB.QueryRowContext(ctx,
		`SELECT total_invested_amount FROM client_fund_positions WHERE fund_id = $1 AND client_id = $2 AND client_type = $3`,
		req.FundId, req.ClientId, req.ClientType,
	).Scan(&positionAmount)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "no position in this fund")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch position: %v", err)
	}

	amount := req.Amount
	if amount == 0 {
		amount = positionAmount
	}
	if amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "nothing to withdraw")
	}
	if amount > positionAmount {
		return nil, status.Errorf(codes.InvalidArgument, "withdrawal amount %.2f exceeds position %.2f", amount, positionAmount)
	}

	// Case 2: insufficient liquidity
	if amount > liquidAssets {
		if req.ClientType != "CLIENT" {
			return nil, status.Error(codes.FailedPrecondition, "insufficient fund liquidity")
		}
		// Auto-liquidate: sell portfolio positions until deficit is covered
		return s.autoLiquidate(ctx, req, amount, liquidAssets)
	}

	// Case 1: sufficient liquidity — immediate settlement
	destCurrency := s.getAccountCurrencyCode(ctx, req.DestinationAccountId)
	creditAmount := s.convertViaExchange(ctx, "RSD", destCurrency, amount)

	_, err = s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
		amount, fundAccountID,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to debit fund account: %v", err)
	}

	_, err = s.AccountDB.ExecContext(ctx,
		`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
		creditAmount, req.DestinationAccountId,
	)
	if err != nil {
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			amount, fundAccountID,
		)
		return nil, status.Errorf(codes.Internal, "failed to credit account: %v", err)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
			creditAmount, req.DestinationAccountId,
		)
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			amount, fundAccountID,
		)
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE investment_funds SET liquid_assets = liquid_assets - $1 WHERE id = $2`,
		amount, req.FundId,
	)
	if err != nil {
		_ = tx.Rollback()
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
			creditAmount, req.DestinationAccountId,
		)
		return nil, status.Errorf(codes.Internal, "failed to update fund liquid assets: %v", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE client_fund_positions SET total_invested_amount = total_invested_amount - $1, last_modified_at = NOW() WHERE fund_id = $2 AND client_id = $3 AND client_type = $4`,
		amount, req.FundId, req.ClientId, req.ClientType,
	)
	if err != nil {
		_ = tx.Rollback()
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
			creditAmount, req.DestinationAccountId,
		)
		return nil, status.Errorf(codes.Internal, "failed to update position: %v", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO client_fund_transactions (client_id, client_type, fund_id, amount, is_inflow, status) VALUES ($1, $2, $3, $4, false, 'COMPLETED')`,
		req.ClientId, req.ClientType, req.FundId, amount,
	)
	if err != nil {
		_ = tx.Rollback()
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
			creditAmount, req.DestinationAccountId,
		)
		return nil, status.Errorf(codes.Internal, "failed to insert transaction: %v", err)
	}

	if err = tx.Commit(); err != nil {
		_, _ = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
			creditAmount, req.DestinationAccountId,
		)
		return nil, status.Errorf(codes.Internal, "failed to commit: %v", err)
	}

	fund, err := s.fetchFundByID(ctx, req.FundId, true)
	if err != nil {
		return nil, err
	}
	return &pb.WithdrawFundResponse{Pending: false, Fund: fund}, nil
}

// autoLiquidate handles Case 2: sells portfolio positions to cover a withdrawal deficit,
// stores a PENDING transaction, and returns a 202-style response.
func (s *FundServer) autoLiquidate(ctx context.Context, req *pb.WithdrawFundRequest, amount, liquidAssets float64) (*pb.WithdrawFundResponse, error) {
	var accountID, managerID int64
	err := s.DB.QueryRowContext(ctx,
		`SELECT account_id, manager_id FROM investment_funds WHERE id = $1`, req.FundId,
	).Scan(&accountID, &managerID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch fund details: %v", err)
	}

	rows, err := s.DB.QueryContext(ctx,
		`SELECT listing_id, quantity, average_cost FROM fund_portfolio_positions
		 WHERE fund_id = $1 AND quantity > 0 ORDER BY quantity ASC`, req.FundId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch portfolio: %v", err)
	}
	defer func() { _ = rows.Close() }()

	deficit := amount - liquidAssets
	var covered float64
	for rows.Next() {
		if covered >= deficit {
			break
		}
		var listingID int64
		var qty, avgCost float64
		if err := rows.Scan(&listingID, &qty, &avgCost); err != nil {
			return nil, status.Errorf(codes.Internal, "scan portfolio position: %v", err)
		}
		covered += qty * avgCost
		if s.OrderClient != nil {
			_, _ = s.OrderClient.CreateOrder(ctx, &pb_order.CreateOrderRequest{
				UserId:    managerID,
				UserType:  "EMPLOYEE",
				AssetId:   listingID,
				Quantity:  int32(qty),
				AccountId: accountID,
				Direction: "SELL",
				FundId:    req.FundId,
			})
		}
	}

	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO client_fund_transactions
		 (client_id, client_type, fund_id, amount, is_inflow, status, destination_account_id)
		 VALUES ($1, $2, $3, $4, false, 'PENDING', $5)`,
		req.ClientId, req.ClientType, req.FundId, amount, req.DestinationAccountId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert pending transaction: %v", err)
	}

	return &pb.WithdrawFundResponse{
		Pending: true,
		Message: "Payment will arrive once orders are executed",
	}, nil
}

func (s *FundServer) CheckPendingWithdrawals(ctx context.Context, req *pb.CheckPendingWithdrawalsRequest) (*pb.CheckPendingWithdrawalsResponse, error) {
	var liquidAssets float64
	if err := s.DB.QueryRowContext(ctx,
		`SELECT liquid_assets FROM investment_funds WHERE id = $1`, req.FundId,
	).Scan(&liquidAssets); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch fund: %v", err)
	}

	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, client_id, client_type, amount, destination_account_id
		 FROM client_fund_transactions
		 WHERE fund_id = $1 AND is_inflow = false AND status = 'PENDING'
		 ORDER BY timestamp ASC`, req.FundId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch pending transactions: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var completed int64
	for rows.Next() {
		var txID, clientID, destAccID int64
		var clientType string
		var txAmount float64
		if err := rows.Scan(&txID, &clientID, &clientType, &txAmount, &destAccID); err != nil {
			continue
		}
		if liquidAssets < txAmount {
			continue
		}

		destCurrency := s.getAccountCurrencyCode(ctx, destAccID)
		creditAmount := s.convertViaExchange(ctx, "RSD", destCurrency, txAmount)

		_, err = s.AccountDB.ExecContext(ctx,
			`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
			creditAmount, destAccID)
		if err != nil {
			continue
		}

		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			// Undo account credit
			_, _ = s.AccountDB.ExecContext(ctx,
				`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
				creditAmount, destAccID)
			continue
		}
		_, _ = tx.ExecContext(ctx,
			`UPDATE investment_funds SET liquid_assets = liquid_assets - $1 WHERE id = $2`,
			txAmount, req.FundId)
		_, _ = tx.ExecContext(ctx,
			`UPDATE client_fund_positions
			 SET total_invested_amount = total_invested_amount - $1, last_modified_at = NOW()
			 WHERE fund_id = $2 AND client_id = $3 AND client_type = $4`,
			txAmount, req.FundId, clientID, clientType)
		_, _ = tx.ExecContext(ctx,
			`UPDATE client_fund_transactions SET status = 'COMPLETED' WHERE id = $1`, txID)
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			_, _ = s.AccountDB.ExecContext(ctx,
				`UPDATE accounts SET balance = balance - $1, available_balance = available_balance - $1 WHERE id = $2`,
				txAmount, destAccID)
			continue
		}

		liquidAssets -= txAmount
		completed++
	}
	return &pb.CheckPendingWithdrawalsResponse{Completed: completed}, nil
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation (error code 23505).
func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "unique constraint")
}

// getAccountCurrencyCode returns the ISO currency code for the given account (e.g. "RSD", "USD").
// Falls back to "RSD" on any error so callers never need to handle a missing currency.
func (s *FundServer) getAccountCurrencyCode(ctx context.Context, accountID int64) string {
	var currencyID int64
	if err := s.AccountDB.QueryRowContext(ctx,
		`SELECT currency_id FROM accounts WHERE id = $1`, accountID,
	).Scan(&currencyID); err != nil {
		return "RSD"
	}
	if s.ExchangeDB == nil {
		return "RSD"
	}
	var code string
	if err := s.ExchangeDB.QueryRowContext(ctx,
		`SELECT code FROM currencies WHERE id = $1`, currencyID,
	).Scan(&code); err != nil {
		return "RSD"
	}
	return code
}

// convertViaExchange converts amount from fromCurrency to toCurrency using published exchange rates (no commission).
// Returns amount unchanged if currencies are equal or the exchange client is unavailable.
func (s *FundServer) convertViaExchange(ctx context.Context, fromCurrency, toCurrency string, amount float64) float64 {
	if s.ExchangeClient == nil || fromCurrency == toCurrency {
		return amount
	}
	resp, err := s.ExchangeClient.PreviewConversion(ctx, &pb_exchange.PreviewConversionRequest{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Amount:       amount,
	})
	if err != nil {
		return amount
	}
	return resp.ToAmount
}

func (s *FundServer) ValidateFundAccount(ctx context.Context, req *pb.ValidateFundAccountRequest) (*pb.ValidateFundAccountResponse, error) {
	var accountID, managerID int64
	var liquidAssets float64
	err := s.DB.QueryRowContext(ctx,
		`SELECT account_id, manager_id, liquid_assets FROM investment_funds WHERE id = $1 AND active = TRUE`,
		req.FundId,
	).Scan(&accountID, &managerID, &liquidAssets)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "fund not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "validate fund account: %v", err)
	}
	if managerID != req.ManagerId {
		return nil, status.Error(codes.PermissionDenied, "not the fund manager")
	}
	return &pb.ValidateFundAccountResponse{
		AccountId:    accountID,
		IsLiquid:     liquidAssets >= req.RequiredAmount,
		LiquidAssets: liquidAssets,
	}, nil
}

func (s *FundServer) UpdateFundHolding(ctx context.Context, req *pb.UpdateFundHoldingRequest) (*pb.UpdateFundHoldingResponse, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	tradeValue := float64(req.Quantity) * req.Price

	if req.Direction == "BUY" {
		_, err = tx.ExecContext(ctx,
			`UPDATE investment_funds SET liquid_assets = liquid_assets - $1 WHERE id = $2`,
			tradeValue, req.FundId)
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE investment_funds SET liquid_assets = liquid_assets + $1 WHERE id = $2`,
			tradeValue, req.FundId)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update liquid_assets: %v", err)
	}

	if req.Direction == "BUY" {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO fund_portfolio_positions (fund_id, listing_id, quantity, average_cost, acquisition_date)
			VALUES ($1, $2, $3, $4, CURRENT_DATE)
			ON CONFLICT (fund_id, listing_id) DO UPDATE SET
			  average_cost = (fund_portfolio_positions.quantity * fund_portfolio_positions.average_cost + $3 * $4)
			               / (fund_portfolio_positions.quantity + $3),
			  quantity     = fund_portfolio_positions.quantity + $3`,
			req.FundId, req.ListingId, req.Quantity, req.Price)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "upsert fund position: %v", err)
		}
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE fund_portfolio_positions SET quantity = quantity - $1 WHERE fund_id = $2 AND listing_id = $3`,
			req.Quantity, req.FundId, req.ListingId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "update fund position: %v", err)
		}
		_, err = tx.ExecContext(ctx,
			`DELETE FROM fund_portfolio_positions WHERE fund_id = $1 AND listing_id = $2 AND quantity <= 0`,
			req.FundId, req.ListingId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cleanup fund position: %v", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}
	return &pb.UpdateFundHoldingResponse{}, nil
}

func (s *FundServer) TransferFundsByManager(ctx context.Context, req *pb.TransferFundsByManagerRequest) (*pb.TransferFundsByManagerResponse, error) {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE investment_funds SET manager_id = $1 WHERE manager_id = $2 AND active = TRUE`,
		req.NewManagerId, req.OldManagerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "transfer funds: %v", err)
	}
	n, _ := res.RowsAffected()
	return &pb.TransferFundsByManagerResponse{FundsTransferred: n}, nil
}

func (s *FundServer) GetMyPositions(ctx context.Context, req *pb.GetMyPositionsRequest) (*pb.GetMyPositionsResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT cfp.fund_id,
		       cfp.total_invested_amount,
		       f.name,
		       COALESCE(f.description, ''),
		       f.liquid_assets            AS fund_value,
		       f.minimum_contribution,
		       COALESCE((SELECT SUM(total_invested_amount)
		                 FROM client_fund_positions
		                 WHERE fund_id = cfp.fund_id), 0) AS total_all_invested
		FROM client_fund_positions cfp
		JOIN investment_funds f ON f.id = cfp.fund_id
		WHERE cfp.client_id   = $1
		  AND cfp.client_type = $2
		  AND f.active        = TRUE`,
		req.ClientId, req.ClientType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get my positions: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var positions []*pb.ClientFundPosition
	for rows.Next() {
		var (
			fundID           int64
			totalInvested    float64
			fundName         string
			description      string
			fundValue        float64
			minContribution  float64
			totalAllInvested float64
		)
		if err := rows.Scan(&fundID, &totalInvested, &fundName, &description,
			&fundValue, &minContribution, &totalAllInvested); err != nil {
			return nil, status.Errorf(codes.Internal, "scan position: %v", err)
		}

		var currentPositionValue float64
		if totalAllInvested > 0 {
			currentPositionValue = (totalInvested / totalAllInvested) * fundValue
		}
		var fundPercentage float64
		if fundValue > 0 {
			fundPercentage = (currentPositionValue / fundValue) * 100
		}

		positions = append(positions, &pb.ClientFundPosition{
			FundId:               fundID,
			FundName:             fundName,
			Description:          description,
			FundValue:            fundValue,
			FundPercentage:       fundPercentage,
			CurrentPositionValue: currentPositionValue,
			TotalInvestedAmount:  totalInvested,
			Profit:               currentPositionValue - totalInvested,
			MinimumContribution:  minContribution,
		})
	}
	if positions == nil {
		positions = []*pb.ClientFundPosition{}
	}
	return &pb.GetMyPositionsResponse{Positions: positions}, nil
}

// GetBankPositions returns all investment fund positions held by the bank (client_type='BANK', client_id=0).
// Used by the bank profit portal (#234).
func (s *FundServer) GetBankPositions(ctx context.Context, _ *pb.GetBankPositionsRequest) (*pb.GetBankPositionsResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT cfp.fund_id,
		       cfp.total_invested_amount                                         AS bank_invested,
		       f.name,
		       f.liquid_assets                                                   AS fund_value,
		       f.manager_id,
		       COALESCE((SELECT SUM(total_invested_amount)
		                 FROM client_fund_positions
		                 WHERE fund_id = cfp.fund_id), 0)                        AS total_all_invested
		FROM client_fund_positions cfp
		JOIN investment_funds f ON f.id = cfp.fund_id
		WHERE cfp.client_type = 'BANK'
		  AND cfp.client_id   = 0
		  AND f.active        = TRUE`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get bank positions: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var positions []*pb.BankFundPosition
	for rows.Next() {
		var (
			fundID           int64
			bankInvested     float64
			fundName         string
			fundValue        float64
			managerID        int64
			totalAllInvested float64
		)
		if err := rows.Scan(&fundID, &bankInvested, &fundName, &fundValue, &managerID, &totalAllInvested); err != nil {
			return nil, status.Errorf(codes.Internal, "scan bank position: %v", err)
		}

		var bankShareRSD float64
		if totalAllInvested > 0 {
			bankShareRSD = (bankInvested / totalAllInvested) * fundValue
		}
		var bankSharePercent float64
		if fundValue > 0 {
			bankSharePercent = (bankShareRSD / fundValue) * 100
		}

		var managerName string
		if managerID != 0 {
			_ = s.EmployeeDB.QueryRowContext(ctx,
				`SELECT first_name || ' ' || last_name FROM employees WHERE id = $1`, managerID,
			).Scan(&managerName)
		}

		positions = append(positions, &pb.BankFundPosition{
			FundId:           fundID,
			FundName:         fundName,
			ManagerName:      managerName,
			BankSharePercent: bankSharePercent,
			BankShareRsd:     bankShareRSD,
			ProfitRsd:        bankShareRSD - bankInvested,
		})
	}
	if positions == nil {
		positions = []*pb.BankFundPosition{}
	}
	return &pb.GetBankPositionsResponse{Positions: positions}, nil
}

func (s *FundServer) GetFundPortfolio(ctx context.Context, req *pb.GetFundPortfolioRequest) (*pb.GetFundPortfolioResponse, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT listing_id, quantity, average_cost, acquisition_date
		 FROM fund_portfolio_positions
		 WHERE fund_id = $1 AND quantity > 0`, req.FundId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get fund portfolio: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var positions []*pb.FundPortfolioPosition
	for rows.Next() {
		var p pb.FundPortfolioPosition
		var acqDate time.Time
		if err := rows.Scan(&p.ListingId, &p.Quantity, &p.AverageCost, &acqDate); err != nil {
			return nil, status.Errorf(codes.Internal, "scan position: %v", err)
		}
		p.AcquisitionDate = acqDate.Format("2006-01-02")
		positions = append(positions, &p)
	}
	if positions == nil {
		positions = []*pb.FundPortfolioPosition{}
	}
	return &pb.GetFundPortfolioResponse{Positions: positions}, nil
}

func (s *FundServer) GetFundPerformanceHistory(ctx context.Context, req *pb.GetFundPerformanceRequest) (*pb.GetFundPerformanceResponse, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT TO_CHAR(date, 'YYYY-MM-DD'), fund_value, profit
		 FROM fund_performance_history
		 WHERE fund_id = $1 AND date >= $2 AND date <= $3
		 ORDER BY date ASC`,
		req.FundId, req.From, req.To,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query performance history: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var records []*pb.PerformanceRecord
	for rows.Next() {
		var r pb.PerformanceRecord
		if err := rows.Scan(&r.Date, &r.FundValue, &r.Profit); err != nil {
			return nil, status.Errorf(codes.Internal, "scan performance record: %v", err)
		}
		records = append(records, &r)
	}
	if records == nil {
		records = []*pb.PerformanceRecord{}
	}
	return &pb.GetFundPerformanceResponse{Records: records}, nil
}

func (s *FundServer) calculateFundStats(ctx context.Context, fundID int64) *pb.FundStats {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT fund_value FROM fund_performance_history WHERE fund_id = $1 ORDER BY date ASC`, fundID)
	if err != nil {
		return &pb.FundStats{HasSufficientData: false}
	}
	defer func() { _ = rows.Close() }()

	var vals []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err == nil {
			vals = append(vals, v)
		}
	}

	if len(vals) < 30 || vals[0] == 0 {
		return &pb.FundStats{HasSufficientData: false}
	}

	n := float64(len(vals) - 1)
	annualReturn := math.Pow(vals[len(vals)-1]/vals[0], 365.0/n) - 1

	var monthlyReturns []float64
	for i := 30; i < len(vals); i += 30 {
		if vals[i-30] != 0 {
			monthlyReturns = append(monthlyReturns, (vals[i]-vals[i-30])/vals[i-30])
		}
	}

	var volatility float64
	if len(monthlyReturns) >= 2 {
		mean := 0.0
		for _, r := range monthlyReturns {
			mean += r
		}
		mean /= float64(len(monthlyReturns))
		variance := 0.0
		for _, r := range monthlyReturns {
			d := r - mean
			variance += d * d
		}
		variance /= float64(len(monthlyReturns) - 1)
		volatility = math.Sqrt(variance) * math.Sqrt(12)
	}

	rewardToVariability := 0.0
	if volatility != 0 {
		rewardToVariability = annualReturn / volatility
	}

	peak := vals[0]
	maxDD := 0.0
	for _, v := range vals {
		if v > peak {
			peak = v
		}
		if peak > 0 {
			if dd := (peak - v) / peak; dd > maxDD {
				maxDD = dd
			}
		}
	}

	return &pb.FundStats{
		AnnualReturn:        annualReturn,
		RewardToVariability: rewardToVariability,
		MaxDrawdown:         maxDD,
		Volatility:          volatility,
		HasSufficientData:   true,
	}
}

func (s *FundServer) GetAveragePerformance(ctx context.Context, req *pb.GetAveragePerformanceRequest) (*pb.GetAveragePerformanceResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT TO_CHAR(date, 'YYYY-MM-DD'), AVG(fund_value)
		FROM fund_performance_history fph
		JOIN investment_funds f ON f.id = fph.fund_id
		WHERE f.active = true AND date >= $1 AND date <= $2
		GROUP BY date ORDER BY date ASC`, req.From, req.To)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query average performance: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var records []*pb.AveragePerformanceRecord
	for rows.Next() {
		var r pb.AveragePerformanceRecord
		if err := rows.Scan(&r.Date, &r.AvgFundValue); err != nil {
			return nil, status.Errorf(codes.Internal, "scan avg performance: %v", err)
		}
		records = append(records, &r)
	}
	if records == nil {
		records = []*pb.AveragePerformanceRecord{}
	}
	return &pb.GetAveragePerformanceResponse{Records: records}, nil
}

func (s *FundServer) ProcessFundDividend(ctx context.Context, req *pb.ProcessFundDividendRequest) (*pb.ProcessFundDividendResponse, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT fund_id, quantity FROM fund_portfolio_positions WHERE listing_id = $1 AND quantity > 0`,
		req.ListingId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query fund positions: %v", err)
	}
	defer func() { _ = rows.Close() }()

	type fundPos struct {
		fundID   int64
		quantity float64
	}
	var positions []fundPos
	for rows.Next() {
		var fp fundPos
		if err := rows.Scan(&fp.fundID, &fp.quantity); err == nil {
			positions = append(positions, fp)
		}
	}
	_ = rows.Close()

	var processed int64
	for _, fp := range positions {
		totalDividend := fp.quantity * req.Price * (req.DividendYield / 4.0)
		if totalDividend <= 0 {
			continue
		}

		newShares := totalDividend / req.Price
		_, _ = s.DB.ExecContext(ctx,
			`UPDATE fund_portfolio_positions SET quantity = quantity + $1 WHERE fund_id = $2 AND listing_id = $3`,
			newShares, fp.fundID, req.ListingId)

		var totalInvested float64
		_ = s.DB.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(total_invested_amount), 0) FROM client_fund_positions WHERE fund_id = $1`,
			fp.fundID,
		).Scan(&totalInvested)
		if totalInvested <= 0 {
			processed++
			continue
		}

		cliRows, err := s.DB.QueryContext(ctx,
			`SELECT client_id, client_type, total_invested_amount FROM client_fund_positions WHERE fund_id = $1 AND total_invested_amount > 0`,
			fp.fundID)
		if err != nil {
			continue
		}

		type cliPos struct {
			clientID   int64
			clientType string
			invested   float64
		}
		var clients []cliPos
		for cliRows.Next() {
			var cp cliPos
			if err := cliRows.Scan(&cp.clientID, &cp.clientType, &cp.invested); err == nil {
				clients = append(clients, cp)
			}
		}
		_ = cliRows.Close()

		today := req.PaymentDate
		for _, cp := range clients {
			gross := totalDividend * (cp.invested / totalInvested)
			var taxRSD, net float64
			if cp.clientType == "CLIENT" {
				taxRSD = gross * 0.15
				net = gross * 0.85
			} else {
				net = gross
			}

			var accountID int64
			_ = s.AccountDB.QueryRowContext(ctx, `
				SELECT id FROM accounts
				WHERE owner_id = $1
				  AND currency_id = (SELECT id FROM currencies WHERE code = $2)
				  AND account_type NOT IN ('BANK','STATE')
				  AND status = 'ACTIVE'
				ORDER BY id LIMIT 1`, cp.clientID, req.Currency,
			).Scan(&accountID)

			if accountID != 0 {
				_, _ = s.AccountDB.ExecContext(ctx,
					`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE id = $2`,
					net, accountID)
			}

			if s.PortfolioClient != nil {
				_, _ = s.PortfolioClient.CreateDividendPayout(ctx, &pb_portfolio.CreateDividendPayoutRequest{
					UserId:         cp.clientID,
					UserType:       cp.clientType,
					StockListingId: req.ListingId,
					Quantity:       fp.quantity,
					GrossAmount:    gross,
					Currency:       req.Currency,
					TaxRsd:         taxRSD,
					NetAmount:      net,
					AccountId:      accountID,
					PaymentDate:    today,
				})
			}
		}
		processed++
	}

	return &pb.ProcessFundDividendResponse{FundsProcessed: processed}, nil
}
