package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/lib/pq"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	DB         *sql.DB // payment_db
	AccountDB  *sql.DB // account_db
	ExchangeDB *sql.DB // exchange_db
}

func (s *PaymentServer) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
	// 1. Load fromAccount and verify ownership
	var fromID int64
	var ownerID int64
	var availableBalance float64
	var dailyLimit, monthlyLimit sql.NullFloat64
	var dailySpent, monthlySpent float64
	var fromCurrencyID int64

	err := s.AccountDB.QueryRowContext(ctx, `
		SELECT id, owner_id, available_balance,
		       daily_limit, monthly_limit, daily_spent, monthly_spent, currency_id
		FROM accounts WHERE account_number = $1`, req.FromAccount,
	).Scan(&fromID, &ownerID, &availableBalance,
		&dailyLimit, &monthlyLimit, &dailySpent, &monthlySpent, &fromCurrencyID)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "source account %s not found", req.FromAccount)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load source account: %v", err)
	}
	if ownerID != req.ClientId {
		return nil, status.Errorf(codes.PermissionDenied, "account does not belong to this client")
	}

	// 2. Validate funds and limits
	if availableBalance < req.Amount {
		return nil, status.Errorf(codes.FailedPrecondition, "insufficient funds")
	}
	if dailyLimit.Valid && dailySpent+req.Amount > dailyLimit.Float64 {
		return nil, status.Errorf(codes.FailedPrecondition, "daily limit exceeded")
	}
	if monthlyLimit.Valid && monthlySpent+req.Amount > monthlyLimit.Float64 {
		return nil, status.Errorf(codes.FailedPrecondition, "monthly limit exceeded")
	}

	// 3. Determine fee (issue #37): same currency → fee=0, different → 0–1%
	var toCurrencyID int64
	var toAccountID int64
	toExists := false
	_ = s.AccountDB.QueryRowContext(ctx,
		`SELECT id, currency_id FROM accounts WHERE account_number = $1`, req.RecipientAccount,
	).Scan(&toAccountID, &toCurrencyID)
	if toAccountID != 0 {
		toExists = true
	}

	fee := 0.0
	finalAmount := req.Amount
	if toExists && toCurrencyID != fromCurrencyID {
		// Different currencies: random fee 0–1%
		feeRate := rand.Float64() * 0.01
		fee = req.Amount * feeRate
		finalAmount = req.Amount - fee
	}

	// 4. Execute transfer in account_db transaction
	tx, err := s.AccountDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Debit fromAccount
	_, err = tx.ExecContext(ctx, `
		UPDATE accounts SET
			balance           = balance - $1,
			available_balance = available_balance - $1,
			daily_spent       = daily_spent + $1,
			monthly_spent     = monthly_spent + $1
		WHERE id = $2`, req.Amount, fromID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to debit source account: %v", err)
	}

	// Credit toAccount if it's in our system
	if toExists {
		_, err = tx.ExecContext(ctx, `
			UPDATE accounts SET
				balance           = balance + $1,
				available_balance = available_balance + $1
			WHERE id = $2`, finalAmount, toAccountID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to credit destination account: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	// 5. Persist payment record
	orderNumber := fmt.Sprintf("ORD-%d-%04d", time.Now().UnixMilli(), rand.Intn(10000))
	now := time.Now()

	var paymentID int64
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO payments
			(order_number, from_account, to_account, initial_amount, final_amount,
			 fee, recipient_id, payment_code, reference_number, purpose, timestamp, status)
		VALUES ($1, $2, $3, $4, $5, $6,
			(SELECT id FROM payment_recipients WHERE client_id = $7 AND account_number = $8 LIMIT 1),
			$9, $10, $11, $12, 'COMPLETED')
		RETURNING id`,
		orderNumber, req.FromAccount, req.RecipientAccount,
		req.Amount, finalAmount, fee,
		req.ClientId, req.RecipientAccount,
		req.PaymentCode, req.ReferenceNumber, req.Purpose, now,
	).Scan(&paymentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to persist payment: %v", err)
	}

	return &pb.CreatePaymentResponse{
		Id:            paymentID,
		OrderNumber:   orderNumber,
		FromAccount:   req.FromAccount,
		ToAccount:     req.RecipientAccount,
		InitialAmount: req.Amount,
		FinalAmount:   finalAmount,
		Fee:           fee,
		Status:        "COMPLETED",
		Timestamp:     now.Format(time.RFC3339),
	}, nil
}

func (s *PaymentServer) CreatePaymentRecipient(ctx context.Context, req *pb.CreatePaymentRecipientRequest) (*pb.CreatePaymentRecipientResponse, error) {
	var id int64
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO payment_recipients (client_id, name, account_number)
		VALUES ($1, $2, $3)
		RETURNING id`,
		req.ClientId, req.Name, req.AccountNumber,
	).Scan(&id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create payment recipient: %v", err)
	}
	return &pb.CreatePaymentRecipientResponse{
		Recipient: &pb.PaymentRecipient{
			Id:            id,
			ClientId:      req.ClientId,
			Name:          req.Name,
			AccountNumber: req.AccountNumber,
		},
	}, nil
}

func (s *PaymentServer) GetPaymentRecipients(ctx context.Context, req *pb.GetPaymentRecipientsRequest) (*pb.GetPaymentRecipientsResponse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, client_id, name, account_number
		FROM payment_recipients
		WHERE client_id = $1`, req.ClientId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query payment recipients: %v", err)
	}
	defer rows.Close()

	var recipients []*pb.PaymentRecipient
	for rows.Next() {
		var r pb.PaymentRecipient
		if err := rows.Scan(&r.Id, &r.ClientId, &r.Name, &r.AccountNumber); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan payment recipient: %v", err)
		}
		recipients = append(recipients, &r)
	}
	return &pb.GetPaymentRecipientsResponse{Recipients: recipients}, nil
}

func (s *PaymentServer) UpdatePaymentRecipient(ctx context.Context, req *pb.UpdatePaymentRecipientRequest) (*pb.UpdatePaymentRecipientResponse, error) {
	var r pb.PaymentRecipient
	err := s.DB.QueryRowContext(ctx, `
		UPDATE payment_recipients
		SET name = $3, account_number = $4
		WHERE id = $1 AND client_id = $2
		RETURNING id, client_id, name, account_number`,
		req.Id, req.ClientId, req.Name, req.AccountNumber,
	).Scan(&r.Id, &r.ClientId, &r.Name, &r.AccountNumber)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "payment recipient not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update payment recipient: %v", err)
	}
	return &pb.UpdatePaymentRecipientResponse{Recipient: &r}, nil
}

func (s *PaymentServer) DeletePaymentRecipient(ctx context.Context, req *pb.DeletePaymentRecipientRequest) (*pb.DeletePaymentRecipientResponse, error) {
	result, err := s.DB.ExecContext(ctx, `
		DELETE FROM payment_recipients WHERE id = $1 AND client_id = $2`,
		req.Id, req.ClientId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete payment recipient: %v", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Error(codes.NotFound, "payment recipient not found")
	}
	return &pb.DeletePaymentRecipientResponse{}, nil
}

func (s *PaymentServer) GetPaymentById(ctx context.Context, req *pb.GetPaymentByIdRequest) (*pb.GetPaymentByIdResponse, error) {
	var p pb.Payment
	var ts time.Time
	var recipientName sql.NullString

	err := s.DB.QueryRowContext(ctx, `
		SELECT p.id, p.order_number, p.from_account, p.to_account,
		       p.initial_amount, p.final_amount, p.fee,
		       p.payment_code, p.reference_number, p.purpose,
		       p.timestamp, p.status, r.name
		FROM payments p
		LEFT JOIN payment_recipients r ON p.recipient_id = r.id
		WHERE p.id = $1`,
		req.PaymentId,
	).Scan(&p.Id, &p.OrderNumber, &p.FromAccount, &p.ToAccount,
		&p.InitialAmount, &p.FinalAmount, &p.Fee,
		&p.PaymentCode, &p.ReferenceNumber, &p.Purpose,
		&ts, &p.Status, &recipientName,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "payment not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query payment: %v", err)
	}

	// Verify ownership: from_account must belong to this client
	var ownerID int64
	err = s.AccountDB.QueryRowContext(ctx,
		`SELECT owner_id FROM accounts WHERE account_number = $1`, p.FromAccount,
	).Scan(&ownerID)
	if err != nil || ownerID != req.ClientId {
		return nil, status.Error(codes.PermissionDenied, "payment does not belong to this client")
	}

	p.Timestamp = ts.Format(time.RFC3339)
	if recipientName.Valid {
		p.RecipientName = recipientName.String
	}
	return &pb.GetPaymentByIdResponse{Payment: &p}, nil
}

func (s *PaymentServer) GetPayments(ctx context.Context, req *pb.GetPaymentsRequest) (*pb.GetPaymentsResponse, error) {
	// 1. Get all account numbers owned by this client
	accRows, err := s.AccountDB.QueryContext(ctx,
		`SELECT account_number FROM accounts WHERE owner_id = $1`, req.ClientId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query accounts: %v", err)
	}
	defer accRows.Close()

	var accountNumbers []string
	for accRows.Next() {
		var an string
		if err := accRows.Scan(&an); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan account: %v", err)
		}
		accountNumbers = append(accountNumbers, an)
	}
	if len(accountNumbers) == 0 {
		return &pb.GetPaymentsResponse{Payments: []*pb.Payment{}}, nil
	}

	// 2. Query payments with optional filters
	pmRows, err := s.DB.QueryContext(ctx, `
		SELECT id, order_number, from_account, to_account,
		       initial_amount, final_amount, fee,
		       payment_code, reference_number, purpose,
		       timestamp, status
		FROM payments
		WHERE from_account = ANY($1)
		  AND ($2 = '' OR timestamp >= $2::timestamptz)
		  AND ($3 = '' OR timestamp <= $3::timestamptz)
		  AND ($4 = 0 OR initial_amount >= $4)
		  AND ($5 = 0 OR initial_amount <= $5)
		  AND ($6 = '' OR status = $6)
		ORDER BY timestamp DESC`,
		pq.Array(accountNumbers),
		req.DateFrom, req.DateTo,
		req.AmountMin, req.AmountMax,
		req.Status,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query payments: %v", err)
	}
	defer pmRows.Close()

	var payments []*pb.Payment
	for pmRows.Next() {
		var p pb.Payment
		var ts time.Time
		if err := pmRows.Scan(
			&p.Id, &p.OrderNumber, &p.FromAccount, &p.ToAccount,
			&p.InitialAmount, &p.FinalAmount, &p.Fee,
			&p.PaymentCode, &p.ReferenceNumber, &p.Purpose,
			&ts, &p.Status,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan payment: %v", err)
		}
		p.Timestamp = ts.Format(time.RFC3339)
		payments = append(payments, &p)
	}
	return &pb.GetPaymentsResponse{Payments: payments}, nil
}

func (s *PaymentServer) CreateTransfer(ctx context.Context, req *pb.CreateTransferRequest) (*pb.CreateTransferResponse, error) {
	if req.FromAccount == req.ToAccount {
		return nil, status.Error(codes.InvalidArgument, "from and to accounts must be different")
	}

	// 1. Load fromAccount – verify ownership and balance
	var fromID int64
	var fromOwnerID int64
	var availableBalance float64
	var fromCurrencyID int64

	err := s.AccountDB.QueryRowContext(ctx,
		`SELECT id, owner_id, available_balance, currency_id
		 FROM accounts WHERE account_number = $1`, req.FromAccount,
	).Scan(&fromID, &fromOwnerID, &availableBalance, &fromCurrencyID)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "source account %s not found", req.FromAccount)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load source account: %v", err)
	}
	if fromOwnerID != req.ClientId {
		return nil, status.Errorf(codes.PermissionDenied, "source account does not belong to this client")
	}

	// 2. Load toAccount – verify ownership
	var toID int64
	var toOwnerID int64
	var toCurrencyID int64

	err = s.AccountDB.QueryRowContext(ctx,
		`SELECT id, owner_id, currency_id
		 FROM accounts WHERE account_number = $1`, req.ToAccount,
	).Scan(&toID, &toOwnerID, &toCurrencyID)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "destination account %s not found", req.ToAccount)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load destination account: %v", err)
	}
	if toOwnerID != req.ClientId {
		return nil, status.Errorf(codes.PermissionDenied, "destination account does not belong to this client")
	}

	// 3. Check balance
	if availableBalance < req.Amount {
		return nil, status.Error(codes.FailedPrecondition, "insufficient funds")
	}

	// 4. Determine exchange rate and final amount
	exchangeRate := 1.0
	finalAmount := req.Amount

	if fromCurrencyID != toCurrencyID {
		// Resolve currency codes
		var fromCode, toCode string
		if err := s.ExchangeDB.QueryRowContext(ctx,
			`SELECT code FROM currencies WHERE id = $1`, fromCurrencyID,
		).Scan(&fromCode); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to resolve source currency: %v", err)
		}
		if err := s.ExchangeDB.QueryRowContext(ctx,
			`SELECT code FROM currencies WHERE id = $1`, toCurrencyID,
		).Scan(&toCode); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to resolve destination currency: %v", err)
		}

		if err := s.ExchangeDB.QueryRowContext(ctx,
			`SELECT rate FROM exchange_rates WHERE from_currency = $1 AND to_currency = $2`,
			fromCode, toCode,
		).Scan(&exchangeRate); err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "no exchange rate for %s → %s", fromCode, toCode)
		} else if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch exchange rate: %v", err)
		}

		finalAmount = req.Amount * exchangeRate
	}

	// 5. Execute balance update in account_db transaction
	tx, err := s.AccountDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE accounts SET
			balance           = balance - $1,
			available_balance = available_balance - $1
		WHERE id = $2`, req.Amount, fromID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to debit source account: %v", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE accounts SET
			balance           = balance + $1,
			available_balance = available_balance + $1
		WHERE id = $2`, finalAmount, toID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to credit destination account: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	// 6. Persist transfer record
	orderNumber := fmt.Sprintf("TRF-%d-%04d", time.Now().UnixMilli(), rand.Intn(10000))
	now := time.Now()

	var transferID int64
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO transfers
			(order_number, from_account, to_account, initial_amount, final_amount, exchange_rate, fee, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, 0, $7)
		RETURNING id`,
		orderNumber, req.FromAccount, req.ToAccount,
		req.Amount, finalAmount, exchangeRate, now,
	).Scan(&transferID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to persist transfer: %v", err)
	}

	return &pb.CreateTransferResponse{
		Id:            transferID,
		OrderNumber:   orderNumber,
		FromAccount:   req.FromAccount,
		ToAccount:     req.ToAccount,
		InitialAmount: req.Amount,
		FinalAmount:   finalAmount,
		ExchangeRate:  exchangeRate,
		Fee:           0,
		Timestamp:     now.Format(time.RFC3339),
	}, nil
}
