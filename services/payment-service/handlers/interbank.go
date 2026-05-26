package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *PaymentServer) PrepareInterbankPayment(ctx context.Context, req *pb.PrepareInterbankPaymentRequest) (*pb.PrepareInterbankPaymentResponse, error) {
	idemRoutingNum := fmt.Sprintf("%d", req.IdempotenceKey.RoutingNumber)
	idemKey := req.IdempotenceKey.LocallyGeneratedKey
	txRoutingNum := fmt.Sprintf("%d", req.TransactionId.RoutingNumber)
	txID := req.TransactionId.Id

	// Check idempotence: return cached vote if we have already seen this key
	var cachedVote string
	err := s.DB.QueryRowContext(ctx,
		`SELECT cached_vote FROM interbank_transactions WHERE idem_routing_number = $1 AND idem_key = $2`,
		idemRoutingNum, idemKey,
	).Scan(&cachedVote)
	if err == nil {
		return &pb.PrepareInterbankPaymentResponse{Vote: cachedVote}, nil
	}
	if err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "failed to check idempotence: %v", err)
	}

	// Validate monetary postings sum to zero per currency (double-entry: debits = credits).
	// OPTION postings are non-monetary (stock instrument) and are excluded from this check.
	totals := make(map[string]float64)
	for _, p := range req.Postings {
		if p.AccountType == "OPTION" {
			continue
		}
		totals[p.Currency] += p.Amount
	}
	for currency, sum := range totals {
		if sum > 0.001 || sum < -0.001 {
			if err := s.insertInterbankTx(ctx, txRoutingNum, txID, idemRoutingNum, idemKey, "", 0, "", "NO"); err != nil {
				return nil, err
			}
			return &pb.PrepareInterbankPaymentResponse{
				Vote:    "NO",
				Reasons: []*pb.InterbankReason{{Reason: fmt.Sprintf("UNBALANCED_TX: postings for %s do not sum to zero", currency)}},
			}, nil
		}
	}

	// Find the credit posting for MONAS (positive amount = money flowing into our bank)
	var creditPosting *pb.InterbankPosting
	for _, p := range req.Postings {
		if p.Amount > 0 && p.AssetType == "MONAS" {
			creditPosting = p
			break
		}
	}
	if creditPosting == nil {
		if err := s.insertInterbankTx(ctx, txRoutingNum, txID, idemRoutingNum, idemKey, "", 0, "", "NO"); err != nil {
			return nil, err
		}
		return &pb.PrepareInterbankPaymentResponse{
			Vote:    "NO",
			Reasons: []*pb.InterbankReason{{Reason: "NO_CREDIT_POSTING: no positive MONAS posting found"}},
		}, nil
	}

	// Verify receiver account exists, is ACTIVE, and currency matches.
	// PERSON postings identify the account by owner_id rather than account_number.
	var resolvedAccountNum string
	var accountStatus, accountCurrencyCode string
	var currencyID int64

	if creditPosting.AccountType == "PERSON" {
		ownerID, parseErr := strconv.ParseInt(creditPosting.AccountNum, 10, 64)
		if parseErr != nil {
			if err2 := s.insertInterbankTx(ctx, txRoutingNum, txID, idemRoutingNum, idemKey, "", 0, "", "NO"); err2 != nil {
				return nil, err2
			}
			return &pb.PrepareInterbankPaymentResponse{
				Vote:    "NO",
				Reasons: []*pb.InterbankReason{{Reason: "NO_SUCH_ACCOUNT: invalid PERSON account num"}},
			}, nil
		}
		err = s.AccountDB.QueryRowContext(ctx,
			`SELECT account_number, status, currency_id FROM accounts
			 WHERE owner_id = $1 AND status = 'ACTIVE' ORDER BY id LIMIT 1`, ownerID,
		).Scan(&resolvedAccountNum, &accountStatus, &currencyID)
	} else {
		resolvedAccountNum = creditPosting.AccountNum
		err = s.AccountDB.QueryRowContext(ctx,
			`SELECT status, currency_id FROM accounts WHERE account_number = $1`,
			creditPosting.AccountNum,
		).Scan(&accountStatus, &currencyID)
	}

	if err == sql.ErrNoRows {
		if err2 := s.insertInterbankTx(ctx, txRoutingNum, txID, idemRoutingNum, idemKey, resolvedAccountNum, creditPosting.Amount, creditPosting.Currency, "NO"); err2 != nil {
			return nil, err2
		}
		return &pb.PrepareInterbankPaymentResponse{
			Vote:    "NO",
			Reasons: []*pb.InterbankReason{{Reason: "NO_SUCH_ACCOUNT: receiver account not found"}},
		}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to look up receiver account: %v", err)
	}
	if accountStatus != "ACTIVE" {
		if err2 := s.insertInterbankTx(ctx, txRoutingNum, txID, idemRoutingNum, idemKey, resolvedAccountNum, creditPosting.Amount, creditPosting.Currency, "NO"); err2 != nil {
			return nil, err2
		}
		return &pb.PrepareInterbankPaymentResponse{
			Vote:    "NO",
			Reasons: []*pb.InterbankReason{{Reason: "NO_SUCH_ACCOUNT: receiver account is not active"}},
		}, nil
	}

	// Check currency match
	if err := s.ExchangeDB.QueryRowContext(ctx,
		`SELECT code FROM currencies WHERE id = $1`, currencyID,
	).Scan(&accountCurrencyCode); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve account currency: %v", err)
	}
	if accountCurrencyCode != creditPosting.Currency {
		if err2 := s.insertInterbankTx(ctx, txRoutingNum, txID, idemRoutingNum, idemKey, resolvedAccountNum, creditPosting.Amount, creditPosting.Currency, "NO"); err2 != nil {
			return nil, err2
		}
		return &pb.PrepareInterbankPaymentResponse{
			Vote:    "NO",
			Reasons: []*pb.InterbankReason{{Reason: "UNACCEPTABLE_ASSET: account currency does not match posting currency"}},
		}, nil
	}

	// All checks passed — store PENDING with cached YES vote
	if err := s.insertInterbankTx(ctx, txRoutingNum, txID, idemRoutingNum, idemKey, resolvedAccountNum, creditPosting.Amount, creditPosting.Currency, "YES"); err != nil {
		return nil, err
	}
	return &pb.PrepareInterbankPaymentResponse{Vote: "YES"}, nil
}

func (s *PaymentServer) CommitInterbankPayment(ctx context.Context, req *pb.CommitRollbackInterbankRequest) (*pb.CommitRollbackInterbankResponse, error) {
	txRoutingNum := fmt.Sprintf("%d", req.TransactionId.RoutingNumber)
	txID := req.TransactionId.Id

	var id int64
	var txStatus, toAccount string
	var amount float64
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, status, to_account, amount FROM interbank_transactions WHERE tx_routing_number = $1 AND tx_id = $2`,
		txRoutingNum, txID,
	).Scan(&id, &txStatus, &toAccount, &amount)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "interbank transaction not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to look up interbank transaction: %v", err)
	}

	if txStatus == "COMMITTED" {
		return &pb.CommitRollbackInterbankResponse{}, nil
	}
	if txStatus == "ROLLED_BACK" {
		return nil, status.Errorf(codes.FailedPrecondition, "transaction was already rolled back")
	}

	// Credit the receiver account and mark committed in a single DB transaction
	dbTx, err := s.AccountDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() { _ = dbTx.Rollback() }()

	if _, err = dbTx.ExecContext(ctx,
		`UPDATE accounts SET balance = balance + $1, available_balance = available_balance + $1 WHERE account_number = $2`,
		amount, toAccount,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to credit receiver account: %v", err)
	}
	if err = dbTx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit account credit: %v", err)
	}

	if _, err = s.DB.ExecContext(ctx,
		`UPDATE interbank_transactions SET status = 'COMMITTED' WHERE id = $1`, id,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update interbank transaction status: %v", err)
	}
	return &pb.CommitRollbackInterbankResponse{}, nil
}

func (s *PaymentServer) RollbackInterbankPayment(ctx context.Context, req *pb.CommitRollbackInterbankRequest) (*pb.CommitRollbackInterbankResponse, error) {
	txRoutingNum := fmt.Sprintf("%d", req.TransactionId.RoutingNumber)
	txID := req.TransactionId.Id

	var id int64
	var txStatus string
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, status FROM interbank_transactions WHERE tx_routing_number = $1 AND tx_id = $2`,
		txRoutingNum, txID,
	).Scan(&id, &txStatus)
	if err == sql.ErrNoRows {
		// Nothing to roll back — idempotent OK
		return &pb.CommitRollbackInterbankResponse{}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to look up interbank transaction: %v", err)
	}

	if txStatus == "ROLLED_BACK" {
		return &pb.CommitRollbackInterbankResponse{}, nil
	}
	if txStatus == "COMMITTED" {
		return nil, status.Errorf(codes.FailedPrecondition, "transaction already committed, cannot roll back")
	}

	if _, err = s.DB.ExecContext(ctx,
		`UPDATE interbank_transactions SET status = 'ROLLED_BACK' WHERE id = $1`, id,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update interbank transaction status: %v", err)
	}
	return &pb.CommitRollbackInterbankResponse{}, nil
}

func (s *PaymentServer) insertInterbankTx(ctx context.Context, txRoutingNum, txID, idemRoutingNum, idemKey, toAccount string, amount float64, currency, vote string) error {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO interbank_transactions
			(tx_routing_number, tx_id, idem_routing_number, idem_key, status, to_account, amount, currency, cached_vote)
		VALUES ($1, $2, $3, $4, 'PENDING', $5, $6, $7, $8)
		ON CONFLICT (idem_routing_number, idem_key) DO NOTHING`,
		txRoutingNum, txID, idemRoutingNum, idemKey, toAccount, amount, currency, vote,
	)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to store interbank transaction: %v", err)
	}
	return nil
}
