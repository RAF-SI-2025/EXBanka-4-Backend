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

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/email-service/queue"
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
