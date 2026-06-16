package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/email-service/queue"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
)

// ---- mock Publisher ----

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) Publish(msg queue.ActivationMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *mockPublisher) PublishPasswordReset(msg queue.PasswordResetMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *mockPublisher) PublishPasswordConfirmation(msg queue.PasswordConfirmationMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *mockPublisher) PublishAccountCreated(msg queue.AccountCreatedMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *mockPublisher) PublishCardConfirmation(msg queue.CardConfirmationMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}
func (m *mockPublisher) PublishLoanLatePayment(msg queue.LoanLatePaymentMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}
func (m *mockPublisher) PublishAccountLocked(msg queue.AccountLockedMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}
func (m *mockPublisher) PublishPaymentNotification(msg queue.PaymentNotificationMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}
func (m *mockPublisher) PublishCardBlocked(msg queue.CardBlockedMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}
func (m *mockPublisher) PublishLoanApproved(msg queue.LoanApprovedMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}
func (m *mockPublisher) PublishLimitChange(msg queue.LimitChangeMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}

// ---- SendActivationEmail tests ----

func TestSendActivationEmail_InvalidEmail(t *testing.T) {
	s := &EmailServer{Producer: &mockPublisher{}}
	_, err := s.SendActivationEmail(context.Background(), &pb.SendActivationEmailRequest{
		Email: "not-an-email",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSendActivationEmail_PublisherFails(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("Publish", mock.Anything).Return(errors.New("amqp error"))

	s := &EmailServer{Producer: pub}
	_, err := s.SendActivationEmail(context.Background(), &pb.SendActivationEmailRequest{
		Email: "user@example.com", FirstName: "John", ActivationLink: "http://link",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestSendActivationEmail_HappyPath(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("Publish", queue.ActivationMessage{
		Email:          "user@example.com",
		FirstName:      "John",
		ActivationLink: "http://activate",
	}).Return(nil)

	s := &EmailServer{Producer: pub}
	resp, err := s.SendActivationEmail(context.Background(), &pb.SendActivationEmailRequest{
		Email: "user@example.com", FirstName: "John", ActivationLink: "http://activate",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	pub.AssertExpectations(t)
}

// ---- SendPasswordResetEmail tests ----

func TestSendPasswordResetEmail_InvalidEmail(t *testing.T) {
	s := &EmailServer{Producer: &mockPublisher{}}
	_, err := s.SendPasswordResetEmail(context.Background(), &pb.SendPasswordResetEmailRequest{
		Email: "bad",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSendPasswordResetEmail_PublisherFails(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("PublishPasswordReset", mock.Anything).Return(errors.New("amqp error"))

	s := &EmailServer{Producer: pub}
	_, err := s.SendPasswordResetEmail(context.Background(), &pb.SendPasswordResetEmailRequest{
		Email: "user@example.com", FirstName: "John", ResetLink: "http://reset",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestSendPasswordResetEmail_HappyPath(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("PublishPasswordReset", queue.PasswordResetMessage{
		Email:     "user@example.com",
		FirstName: "John",
		ResetLink: "http://reset",
	}).Return(nil)

	s := &EmailServer{Producer: pub}
	resp, err := s.SendPasswordResetEmail(context.Background(), &pb.SendPasswordResetEmailRequest{
		Email: "user@example.com", FirstName: "John", ResetLink: "http://reset",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	pub.AssertExpectations(t)
}

// ---- SendPasswordConfirmationEmail tests ----

func TestSendPasswordConfirmationEmail_InvalidEmail(t *testing.T) {
	s := &EmailServer{Producer: &mockPublisher{}}
	_, err := s.SendPasswordConfirmationEmail(context.Background(), &pb.SendActivationEmailRequest{
		Email: "@bad",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSendPasswordConfirmationEmail_PublisherFails(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("PublishPasswordConfirmation", mock.Anything).Return(errors.New("amqp error"))

	s := &EmailServer{Producer: pub}
	_, err := s.SendPasswordConfirmationEmail(context.Background(), &pb.SendActivationEmailRequest{
		Email: "user@example.com", FirstName: "Jane",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestSendPasswordConfirmationEmail_HappyPath(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("PublishPasswordConfirmation", queue.PasswordConfirmationMessage{
		Email:     "user@example.com",
		FirstName: "Jane",
	}).Return(nil)

	s := &EmailServer{Producer: pub}
	resp, err := s.SendPasswordConfirmationEmail(context.Background(), &pb.SendActivationEmailRequest{
		Email: "user@example.com", FirstName: "Jane",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	pub.AssertExpectations(t)
}

// ---- SendAccountCreatedEmail tests ----

func TestSendAccountCreatedEmail_InvalidEmail(t *testing.T) {
	s := &EmailServer{Producer: &mockPublisher{}}
	_, err := s.SendAccountCreatedEmail(context.Background(), &pb.SendAccountCreatedEmailRequest{
		Email: "not-an-email",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSendAccountCreatedEmail_PublisherFails(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("PublishAccountCreated", mock.Anything).Return(errors.New("queue down"))

	s := &EmailServer{Producer: pub}
	_, err := s.SendAccountCreatedEmail(context.Background(), &pb.SendAccountCreatedEmailRequest{
		Email: "client@example.com", FirstName: "Ana",
		AccountName: "Tekuci", AccountNumber: "265000100000000101", CurrencyCode: "RSD",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	pub.AssertExpectations(t)
}

func TestSendAccountCreatedEmail_HappyPath(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("PublishAccountCreated", queue.AccountCreatedMessage{
		Email:         "client@example.com",
		FirstName:     "Ana",
		AccountName:   "Tekuci",
		AccountNumber: "265000100000000101",
		CurrencyCode:  "RSD",
	}).Return(nil)

	s := &EmailServer{Producer: pub}
	resp, err := s.SendAccountCreatedEmail(context.Background(), &pb.SendAccountCreatedEmailRequest{
		Email: "client@example.com", FirstName: "Ana",
		AccountName: "Tekuci", AccountNumber: "265000100000000101", CurrencyCode: "RSD",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	pub.AssertExpectations(t)
}

// ---- SendCardConfirmationEmail tests ----

func TestSendCardConfirmationEmail_InvalidEmail(t *testing.T) {
	s := &EmailServer{Producer: &mockPublisher{}}
	_, err := s.SendCardConfirmationEmail(context.Background(), &pb.SendCardConfirmationEmailRequest{
		Email: "not-an-email",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSendCardConfirmationEmail_PublisherFails(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("PublishCardConfirmation", mock.Anything).Return(errors.New("amqp error"))

	s := &EmailServer{Producer: pub}
	_, err := s.SendCardConfirmationEmail(context.Background(), &pb.SendCardConfirmationEmailRequest{
		Email: "user@example.com", FirstName: "Marko", ConfirmationCode: "482019",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestSendCardConfirmationEmail_HappyPath(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("PublishCardConfirmation", queue.CardConfirmationMessage{
		Email:            "user@example.com",
		FirstName:        "Marko",
		ConfirmationCode: "482019",
	}).Return(nil)

	s := &EmailServer{Producer: pub}
	resp, err := s.SendCardConfirmationEmail(context.Background(), &pb.SendCardConfirmationEmailRequest{
		Email: "user@example.com", FirstName: "Marko", ConfirmationCode: "482019",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	pub.AssertExpectations(t)
}

// ---- SendLoanLatePaymentEmail tests ----

func TestSendLoanLatePaymentEmail_InvalidEmail(t *testing.T) {
	s := &EmailServer{Producer: &mockPublisher{}}
	_, err := s.SendLoanLatePaymentEmail(context.Background(), &pb.SendLoanLatePaymentEmailRequest{
		Email: "bad@@",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSendLoanLatePaymentEmail_PublisherFails(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("PublishLoanLatePayment", mock.Anything).Return(errors.New("amqp error"))

	s := &EmailServer{Producer: pub}
	_, err := s.SendLoanLatePaymentEmail(context.Background(), &pb.SendLoanLatePaymentEmailRequest{
		Email: "user@example.com", FirstName: "Petar",
		LoanNumber: "123", AmountDue: 5000.0, Currency: "RSD", RetryCount: 1,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestSendLoanLatePaymentEmail_HappyPath(t *testing.T) {
	pub := &mockPublisher{}
	pub.On("PublishLoanLatePayment", queue.LoanLatePaymentMessage{
		Email:      "user@example.com",
		FirstName:  "Petar",
		LoanNumber: "123",
		AmountDue:  5000.0,
		Currency:   "RSD",
		RetryCount: 1,
	}).Return(nil)

	s := &EmailServer{Producer: pub}
	resp, err := s.SendLoanLatePaymentEmail(context.Background(), &pb.SendLoanLatePaymentEmailRequest{
		Email: "user@example.com", FirstName: "Petar",
		LoanNumber: "123", AmountDue: 5000.0, Currency: "RSD", RetryCount: 1,
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	pub.AssertExpectations(t)
}
