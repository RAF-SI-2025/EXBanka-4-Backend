package handlers

import (
	"context"
	"net/mail"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/email-service/queue"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
)

type Publisher interface {
	Publish(msg queue.ActivationMessage) error
	PublishPasswordReset(msg queue.PasswordResetMessage) error
	PublishPasswordConfirmation(msg queue.PasswordConfirmationMessage) error
	PublishAccountCreated(msg queue.AccountCreatedMessage) error
	PublishCardConfirmation(msg queue.CardConfirmationMessage) error
	PublishLoanLatePayment(msg queue.LoanLatePaymentMessage) error
	PublishAccountLocked(msg queue.AccountLockedMessage) error
	PublishPaymentNotification(msg queue.PaymentNotificationMessage) error
	PublishCardBlocked(msg queue.CardBlockedMessage) error
	PublishLoanApproved(msg queue.LoanApprovedMessage) error
	PublishLimitChange(msg queue.LimitChangeMessage) error
}

type EmailServer struct {
	pb.UnimplementedEmailServiceServer
	Producer Publisher
}

func (s *EmailServer) SendActivationEmail(_ context.Context, req *pb.SendActivationEmailRequest) (*pb.SendActivationEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.Publish(queue.ActivationMessage{
		Email:          req.Email,
		FirstName:      req.FirstName,
		ActivationLink: req.ActivationLink,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendActivationEmailResponse{}, nil
}

func (s *EmailServer) SendPasswordResetEmail(_ context.Context, req *pb.SendPasswordResetEmailRequest) (*pb.SendPasswordResetEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.PublishPasswordReset(queue.PasswordResetMessage{
		Email:     req.Email,
		FirstName: req.FirstName,
		ResetLink: req.ResetLink,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendPasswordResetEmailResponse{}, nil
}

func (s *EmailServer) SendAccountCreatedEmail(_ context.Context, req *pb.SendAccountCreatedEmailRequest) (*pb.SendAccountCreatedEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.PublishAccountCreated(queue.AccountCreatedMessage{
		Email:         req.Email,
		FirstName:     req.FirstName,
		AccountName:   req.AccountName,
		AccountNumber: req.AccountNumber,
		CurrencyCode:  req.CurrencyCode,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendAccountCreatedEmailResponse{}, nil
}

func (s *EmailServer) SendPasswordConfirmationEmail(_ context.Context, req *pb.SendActivationEmailRequest) (*pb.SendActivationEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.PublishPasswordConfirmation(queue.PasswordConfirmationMessage{
		Email:     req.Email,
		FirstName: req.FirstName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendActivationEmailResponse{}, nil
}

func (s *EmailServer) SendCardConfirmationEmail(_ context.Context, req *pb.SendCardConfirmationEmailRequest) (*pb.SendCardConfirmationEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.PublishCardConfirmation(queue.CardConfirmationMessage{
		Email:            req.Email,
		FirstName:        req.FirstName,
		ConfirmationCode: req.ConfirmationCode,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendCardConfirmationEmailResponse{}, nil
}

func (s *EmailServer) SendLoanLatePaymentEmail(_ context.Context, req *pb.SendLoanLatePaymentEmailRequest) (*pb.SendLoanLatePaymentEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.PublishLoanLatePayment(queue.LoanLatePaymentMessage{
		Email:      req.Email,
		FirstName:  req.FirstName,
		LoanNumber: req.LoanNumber,
		AmountDue:  req.AmountDue,
		Currency:   req.Currency,
		RetryCount: req.RetryCount,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendLoanLatePaymentEmailResponse{}, nil
}

func (s *EmailServer) SendAccountLockedEmail(_ context.Context, req *pb.SendAccountLockedEmailRequest) (*pb.SendAccountLockedEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.PublishAccountLocked(queue.AccountLockedMessage{
		Email:             req.Email,
		FirstName:         req.FirstName,
		PasswordResetLink: req.PasswordResetLink,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendAccountLockedEmailResponse{}, nil
}

func (s *EmailServer) SendPaymentNotificationEmail(_ context.Context, req *pb.SendPaymentNotificationEmailRequest) (*pb.SendPaymentNotificationEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.PublishPaymentNotification(queue.PaymentNotificationMessage{
		Email:         req.Email,
		FirstName:     req.FirstName,
		Direction:     req.Direction,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Counterparty:  req.Counterparty,
		AccountNumber: req.AccountNumber,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendPaymentNotificationEmailResponse{}, nil
}

func (s *EmailServer) SendCardBlockedEmail(_ context.Context, req *pb.SendCardBlockedEmailRequest) (*pb.SendCardBlockedEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.PublishCardBlocked(queue.CardBlockedMessage{
		Email:      req.Email,
		FirstName:  req.FirstName,
		CardNumber: req.CardNumber,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendCardBlockedEmailResponse{}, nil
}

func (s *EmailServer) SendLoanApprovedEmail(_ context.Context, req *pb.SendLoanApprovedEmailRequest) (*pb.SendLoanApprovedEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.PublishLoanApproved(queue.LoanApprovedMessage{
		Email:              req.Email,
		FirstName:          req.FirstName,
		LoanAmount:         req.LoanAmount,
		Currency:           req.Currency,
		MonthlyInstallment: req.MonthlyInstallment,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendLoanApprovedEmailResponse{}, nil
}

func (s *EmailServer) SendLimitChangeEmail(_ context.Context, req *pb.SendLimitChangeEmailRequest) (*pb.SendLimitChangeEmailResponse, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	err := s.Producer.PublishLimitChange(queue.LimitChangeMessage{
		Email:        req.Email,
		FirstName:    req.FirstName,
		DailyLimit:   req.DailyLimit,
		MonthlyLimit: req.MonthlyLimit,
		Currency:     req.Currency,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}
	return &pb.SendLimitChangeEmailResponse{}, nil
}
