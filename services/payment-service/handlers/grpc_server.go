package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	DB        *sql.DB // payment_db
	AccountDB *sql.DB // account_db
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
			 fee, payment_code, reference_number, purpose, timestamp, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'COMPLETED')
		RETURNING id`,
		orderNumber, req.FromAccount, req.RecipientAccount,
		req.Amount, finalAmount, fee,
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
