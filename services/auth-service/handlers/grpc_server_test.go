package handlers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb_auth "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_email "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
)

// ---- Mock gRPC clients ----

type mockEmployeeClient struct {
	mock.Mock
}

func (m *mockEmployeeClient) GetAllEmployees(ctx context.Context, in *pb_emp.GetAllEmployeesRequest, opts ...grpc.CallOption) (*pb_emp.GetAllEmployeesResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_emp.GetAllEmployeesResponse), args.Error(1)
}

func (m *mockEmployeeClient) SearchEmployees(ctx context.Context, in *pb_emp.SearchEmployeesRequest, opts ...grpc.CallOption) (*pb_emp.SearchEmployeesResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_emp.SearchEmployeesResponse), args.Error(1)
}

func (m *mockEmployeeClient) GetEmployeeCredentials(ctx context.Context, in *pb_emp.GetEmployeeCredentialsRequest, opts ...grpc.CallOption) (*pb_emp.GetEmployeeCredentialsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_emp.GetEmployeeCredentialsResponse), args.Error(1)
}

func (m *mockEmployeeClient) CreateEmployee(ctx context.Context, in *pb_emp.CreateEmployeeRequest, opts ...grpc.CallOption) (*pb_emp.CreateEmployeeResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_emp.CreateEmployeeResponse), args.Error(1)
}

func (m *mockEmployeeClient) GetEmployeeById(ctx context.Context, in *pb_emp.GetEmployeeByIdRequest, opts ...grpc.CallOption) (*pb_emp.GetEmployeeByIdResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_emp.GetEmployeeByIdResponse), args.Error(1)
}

func (m *mockEmployeeClient) UpdateEmployee(ctx context.Context, in *pb_emp.UpdateEmployeeRequest, opts ...grpc.CallOption) (*pb_emp.UpdateEmployeeResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_emp.UpdateEmployeeResponse), args.Error(1)
}

func (m *mockEmployeeClient) ActivateEmployee(ctx context.Context, in *pb_emp.ActivateEmployeeRequest, opts ...grpc.CallOption) (*pb_emp.ActivateEmployeeResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_emp.ActivateEmployeeResponse), args.Error(1)
}

func (m *mockEmployeeClient) GetEmployeeByEmail(ctx context.Context, in *pb_emp.GetEmployeeByEmailRequest, opts ...grpc.CallOption) (*pb_emp.GetEmployeeByEmailResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_emp.GetEmployeeByEmailResponse), args.Error(1)
}

func (m *mockEmployeeClient) UpdatePassword(ctx context.Context, in *pb_emp.UpdatePasswordRequest, opts ...grpc.CallOption) (*pb_emp.UpdatePasswordResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_emp.UpdatePasswordResponse), args.Error(1)
}

type mockEmailClient struct {
	mock.Mock
}

func (m *mockEmailClient) SendActivationEmail(ctx context.Context, in *pb_email.SendActivationEmailRequest, opts ...grpc.CallOption) (*pb_email.SendActivationEmailResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_email.SendActivationEmailResponse), args.Error(1)
}

func (m *mockEmailClient) SendPasswordResetEmail(ctx context.Context, in *pb_email.SendPasswordResetEmailRequest, opts ...grpc.CallOption) (*pb_email.SendPasswordResetEmailResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_email.SendPasswordResetEmailResponse), args.Error(1)
}

func (m *mockEmailClient) SendPasswordConfirmationEmail(ctx context.Context, in *pb_email.SendActivationEmailRequest, opts ...grpc.CallOption) (*pb_email.SendActivationEmailResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_email.SendActivationEmailResponse), args.Error(1)
}

func (m *mockEmailClient) SendAccountCreatedEmail(ctx context.Context, in *pb_email.SendAccountCreatedEmailRequest, opts ...grpc.CallOption) (*pb_email.SendAccountCreatedEmailResponse, error) {
	return &pb_email.SendAccountCreatedEmailResponse{}, nil
}

type mockClientClient struct {
	mock.Mock
}

func (m *mockClientClient) GetAllClients(ctx context.Context, in *pb_client.GetAllClientsRequest, opts ...grpc.CallOption) (*pb_client.GetAllClientsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_client.GetAllClientsResponse), args.Error(1)
}

func (m *mockClientClient) GetClientById(ctx context.Context, in *pb_client.GetClientByIdRequest, opts ...grpc.CallOption) (*pb_client.GetClientByIdResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_client.GetClientByIdResponse), args.Error(1)
}

func (m *mockClientClient) CreateClient(ctx context.Context, in *pb_client.CreateClientRequest, opts ...grpc.CallOption) (*pb_client.CreateClientResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_client.CreateClientResponse), args.Error(1)
}

func (m *mockClientClient) UpdateClient(ctx context.Context, in *pb_client.UpdateClientRequest, opts ...grpc.CallOption) (*pb_client.UpdateClientResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_client.UpdateClientResponse), args.Error(1)
}

func (m *mockClientClient) GetClientCredentials(ctx context.Context, in *pb_client.GetClientCredentialsRequest, opts ...grpc.CallOption) (*pb_client.GetClientCredentialsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_client.GetClientCredentialsResponse), args.Error(1)
}

func (m *mockClientClient) ActivateClient(ctx context.Context, in *pb_client.ActivateClientRequest, opts ...grpc.CallOption) (*pb_client.ActivateClientResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb_client.ActivateClientResponse), args.Error(1)
}

// ---- helpers ----

func newAuthServer(db *sql.DB, empClient *mockEmployeeClient, emailClient *mockEmailClient) *AuthServer {
	return &AuthServer{
		DB:             db,
		EmployeeClient: empClient,
		EmailClient:    emailClient,
	}
}

// ---- validatePassword tests ----

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantNil  bool
		wantCode codes.Code
	}{
		{"valid",                    "Abcdef12",                           true,  codes.OK},
		{"valid 32 chars",           "Abcdefgh12345678Abcdefgh12345678",   true,  codes.OK},
		{"valid exactly 8",          "Abcde12!",                           true,  codes.OK},
		{"too short",                "Ab1",                                false, codes.InvalidArgument},
		{"too long 33 chars",        "Abcdefgh123456789012345678901234x",  false, codes.InvalidArgument},
		{"only one digit",           "Abcdefg1",                           false, codes.InvalidArgument},
		{"no digits",                "Abcdefgh",                           false, codes.InvalidArgument},
		{"no uppercase",             "abcdef12",                           false, codes.InvalidArgument},
		{"no lowercase",             "ABCDEF12",                           false, codes.InvalidArgument},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePassword(tc.password)
			if tc.wantNil {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.wantCode, status.Code(err))
			}
		})
	}
}

// ---- generateToken tests ----

func TestGenerateToken_AccessToken(t *testing.T) {
	tokenStr, err := generateToken(42, "john@example.com", "access", []string{"ADMIN"}, "John", "Doe", "john@example.com", 15*time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, float64(42), claims["user_id"])
	assert.Equal(t, "john@example.com", claims["username"])
	assert.Equal(t, "access", claims["type"])
	assert.Equal(t, "John", claims["first_name"])
	assert.Equal(t, "Doe", claims["last_name"])

	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), exp, 5*time.Second)
}

func TestGenerateToken_RefreshToken(t *testing.T) {
	tokenStr, err := generateToken(1, "user@example.com", "refresh", nil, "", "", "", 7*24*time.Hour)
	require.NoError(t, err)

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	require.NoError(t, err)

	claims, _ := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "refresh", claims["type"])

	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	assert.WithinDuration(t, time.Now().Add(7*24*time.Hour), exp, 5*time.Second)
}

// ---- Refresh tests ----

func TestRefresh_ValidToken(t *testing.T) {
	refreshToken, err := generateToken(99, "user@example.com", "refresh", []string{"OPERATOR"}, "Alice", "Smith", "user@example.com", 7*24*time.Hour)
	require.NoError(t, err)

	s := &AuthServer{}
	resp, err := s.Refresh(context.Background(), &pb_auth.RefreshRequest{RefreshToken: refreshToken})
	require.NoError(t, err)
	require.NotEmpty(t, resp.AccessToken)

	token, _ := jwt.Parse(resp.AccessToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	claims, _ := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "access", claims["type"])
	assert.Equal(t, float64(99), claims["user_id"])
}

func TestRefresh_WrongTokenType(t *testing.T) {
	accessToken, err := generateToken(1, "user@example.com", "access", nil, "", "", "", 15*time.Minute)
	require.NoError(t, err)

	s := &AuthServer{}
	_, err = s.Refresh(context.Background(), &pb_auth.RefreshRequest{RefreshToken: accessToken})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestRefresh_ExpiredToken(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id":  float64(1),
		"username": "user@example.com",
		"type":     "refresh",
		"dozvole":  []string{},
		"exp":      time.Now().Add(-time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))

	s := &AuthServer{}
	_, err := s.Refresh(context.Background(), &pb_auth.RefreshRequest{RefreshToken: tokenStr})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestRefresh_InvalidSignature(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id":  float64(1),
		"username": "user@example.com",
		"type":     "refresh",
		"exp":      time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("wrong-secret"))

	s := &AuthServer{}
	_, err := s.Refresh(context.Background(), &pb_auth.RefreshRequest{RefreshToken: tokenStr})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// ---- ActivateAccount tests ----

func TestActivateAccount_TokenNotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("activation_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"employee_id", "expires_at"}))

	s := newAuthServer(db, &mockEmployeeClient{}, &mockEmailClient{})
	_, err = s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "no-such-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestActivateAccount_ExpiredToken(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(-time.Hour))
	dbMock.ExpectQuery("activation_tokens").WillReturnRows(rows)
	dbMock.ExpectExec("DELETE FROM activation_tokens").WillReturnResult(sqlmock.NewResult(1, 1))

	s := newAuthServer(db, &mockEmployeeClient{}, &mockEmailClient{})
	_, err = s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "expired-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestActivateAccount_AlreadyActivated(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("activation_tokens").WillReturnRows(rows)

	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeById", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeByIdResponse{Employee: &pb_emp.Employee{Aktivan: true}}, nil,
	)

	s := newAuthServer(db, empClient, &mockEmailClient{})
	_, err = s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestActivateAccount_PasswordMismatch(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("activation_tokens").WillReturnRows(rows)

	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeById", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeByIdResponse{Employee: &pb_emp.Employee{Aktivan: false}}, nil,
	)

	s := newAuthServer(db, empClient, &mockEmailClient{})
	_, err = s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Different12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestActivateAccount_InvalidPassword(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("activation_tokens").WillReturnRows(rows)

	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeById", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeByIdResponse{Employee: &pb_emp.Employee{Aktivan: false}}, nil,
	)

	s := newAuthServer(db, empClient, &mockEmailClient{})
	_, err = s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "valid-token", Password: "weak", ConfirmPassword: "weak",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestActivateAccount_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("activation_tokens").WillReturnRows(rows)
	dbMock.ExpectExec("DELETE FROM activation_tokens").WillReturnResult(sqlmock.NewResult(1, 1))

	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeById", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeByIdResponse{Employee: &pb_emp.Employee{
			Id: 1, Aktivan: false, Email: "emp@example.com", Ime: "John",
		}}, nil,
	)
	empClient.On("ActivateEmployee", mock.Anything, mock.Anything).Return(
		&pb_emp.ActivateEmployeeResponse{}, nil,
	)

	emailClient := &mockEmailClient{}
	emailClient.On("SendPasswordConfirmationEmail", mock.Anything, mock.Anything).
		Return(&pb_email.SendActivationEmailResponse{}, nil)

	s := newAuthServer(db, empClient, emailClient)
	resp, err := s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	time.Sleep(50 * time.Millisecond) // wait for email goroutine
	empClient.AssertExpectations(t)
	emailClient.AssertExpectations(t)
}

// ---- ResetPassword tests ----

func TestResetPassword_TokenNotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("password_reset_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"employee_id", "expires_at"}))

	s := newAuthServer(db, &mockEmployeeClient{}, &mockEmailClient{})
	_, err = s.ResetPassword(context.Background(), &pb_auth.ResetPasswordRequest{
		Token: "bad-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(-time.Hour))
	dbMock.ExpectQuery("password_reset_tokens").WillReturnRows(rows)
	dbMock.ExpectExec("DELETE FROM password_reset_tokens").WillReturnResult(sqlmock.NewResult(1, 1))

	s := newAuthServer(db, &mockEmployeeClient{}, &mockEmailClient{})
	_, err = s.ResetPassword(context.Background(), &pb_auth.ResetPasswordRequest{
		Token: "expired-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestResetPassword_PasswordMismatch(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("password_reset_tokens").WillReturnRows(rows)

	s := newAuthServer(db, &mockEmployeeClient{}, &mockEmailClient{})
	_, err = s.ResetPassword(context.Background(), &pb_auth.ResetPasswordRequest{
		Token: "good-token", Password: "Abcdef12", ConfirmPassword: "Other456",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestResetPassword_InvalidPassword(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("password_reset_tokens").WillReturnRows(rows)

	s := newAuthServer(db, &mockEmployeeClient{}, &mockEmailClient{})
	_, err = s.ResetPassword(context.Background(), &pb_auth.ResetPasswordRequest{
		Token: "good-token", Password: "nouppercase1", ConfirmPassword: "nouppercase1",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestResetPassword_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(5), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("password_reset_tokens").WillReturnRows(rows)
	dbMock.ExpectExec("DELETE FROM password_reset_tokens").WillReturnResult(sqlmock.NewResult(1, 1))

	empClient := &mockEmployeeClient{}
	empClient.On("UpdatePassword", mock.Anything, mock.Anything).Return(
		&pb_emp.UpdatePasswordResponse{}, nil,
	)

	s := newAuthServer(db, empClient, &mockEmailClient{})
	resp, err := s.ResetPassword(context.Background(), &pb_auth.ResetPasswordRequest{
		Token: "good-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	empClient.AssertExpectations(t)
}

// ---- Login tests ----

func TestLogin_CredentialsNotFound(t *testing.T) {
	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeCredentials", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "not found"))

	s := &AuthServer{EmployeeClient: empClient, EmailClient: &mockEmailClient{}}
	_, err := s.Login(context.Background(), &pb_auth.LoginRequest{Email: "user@example.com", Password: "Abcdef12"})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestLogin_EmptyPasswordHash(t *testing.T) {
	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeCredentials", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeCredentialsResponse{Id: 1, PasswordHash: "", Aktivan: true, Dozvole: []string{}}, nil,
	)

	s := &AuthServer{EmployeeClient: empClient, EmailClient: &mockEmailClient{}}
	_, err := s.Login(context.Background(), &pb_auth.LoginRequest{Email: "user@example.com", Password: "Abcdef12"})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestLogin_AccountNotActive(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("Abcdef12"), bcrypt.MinCost)
	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeCredentials", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeCredentialsResponse{Id: 1, PasswordHash: string(hash), Aktivan: false, Dozvole: []string{}}, nil,
	)

	s := &AuthServer{EmployeeClient: empClient, EmailClient: &mockEmailClient{}}
	_, err := s.Login(context.Background(), &pb_auth.LoginRequest{Email: "user@example.com", Password: "Abcdef12"})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestLogin_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("OtherPass12"), bcrypt.MinCost)
	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeCredentials", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeCredentialsResponse{Id: 1, PasswordHash: string(hash), Aktivan: true, Dozvole: []string{}}, nil,
	)

	s := &AuthServer{EmployeeClient: empClient, EmailClient: &mockEmailClient{}}
	_, err := s.Login(context.Background(), &pb_auth.LoginRequest{Email: "user@example.com", Password: "Abcdef12"})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestLogin_HappyPath(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("Abcdef12"), bcrypt.MinCost)
	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeCredentials", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeCredentialsResponse{Id: 1, PasswordHash: string(hash), Aktivan: true, Dozvole: []string{"ADMIN"}}, nil,
	)
	empClient.On("GetEmployeeById", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeByIdResponse{Employee: &pb_emp.Employee{
			Id: 1, Email: "user@example.com", Ime: "John", Prezime: "Doe",
		}}, nil,
	)

	s := &AuthServer{EmployeeClient: empClient, EmailClient: &mockEmailClient{}}
	resp, err := s.Login(context.Background(), &pb_auth.LoginRequest{Email: "user@example.com", Password: "Abcdef12"})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	empClient.AssertExpectations(t)
}

// ---- ClientLogin tests ----

func TestClientLogin_CredentialsNotFound(t *testing.T) {
	clientClient := &mockClientClient{}
	clientClient.On("GetClientCredentials", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "not found"))

	s := &AuthServer{ClientClient: clientClient}
	_, err := s.ClientLogin(context.Background(), &pb_auth.ClientLoginRequest{Email: "nobody@example.com", Password: "Abcdef12"})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestClientLogin_AccountNotActive(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("Abcdef12"), bcrypt.MinCost)
	clientClient := &mockClientClient{}
	clientClient.On("GetClientCredentials", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientCredentialsResponse{Id: 1, PasswordHash: string(hash), Active: false}, nil,
	)

	s := &AuthServer{ClientClient: clientClient}
	_, err := s.ClientLogin(context.Background(), &pb_auth.ClientLoginRequest{Email: "ana@example.com", Password: "Abcdef12"})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestClientLogin_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("OtherPass12"), bcrypt.MinCost)
	clientClient := &mockClientClient{}
	clientClient.On("GetClientCredentials", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientCredentialsResponse{Id: 1, PasswordHash: string(hash), Active: true}, nil,
	)

	s := &AuthServer{ClientClient: clientClient}
	_, err := s.ClientLogin(context.Background(), &pb_auth.ClientLoginRequest{Email: "ana@example.com", Password: "Abcdef12"})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestClientLogin_HappyPath(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("Abcdef12"), bcrypt.MinCost)
	clientClient := &mockClientClient{}
	clientClient.On("GetClientCredentials", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientCredentialsResponse{Id: 1, PasswordHash: string(hash), Active: true}, nil,
	)
	clientClient.On("GetClientById", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientByIdResponse{Client: &pb_client.Client{
			Id: 1, Email: "ana@example.com", FirstName: "Ana", LastName: "Anić",
		}}, nil,
	)

	s := &AuthServer{ClientClient: clientClient}
	resp, err := s.ClientLogin(context.Background(), &pb_auth.ClientLoginRequest{Email: "ana@example.com", Password: "Abcdef12"})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)

	// access token must have role=CLIENT and no dozvole
	token, _ := jwt.Parse(resp.AccessToken, func(t *jwt.Token) (interface{}, error) { return []byte(jwtSecret), nil })
	claims, _ := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "CLIENT", claims["role"])
	assert.Equal(t, "access", claims["type"])
	_, hasDozvole := claims["dozvole"]
	assert.False(t, hasDozvole)
	clientClient.AssertExpectations(t)
}

// ---- ClientRefresh tests ----

func TestClientRefresh_ValidToken(t *testing.T) {
	refreshToken, err := generateClientToken(7, "ana@example.com", "refresh", "Ana", "Anić", 7*24*time.Hour)
	require.NoError(t, err)

	s := &AuthServer{}
	resp, err := s.ClientRefresh(context.Background(), &pb_auth.ClientRefreshRequest{RefreshToken: refreshToken})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)

	token, _ := jwt.Parse(resp.AccessToken, func(t *jwt.Token) (interface{}, error) { return []byte(jwtSecret), nil })
	claims, _ := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "CLIENT", claims["role"])
	assert.Equal(t, "access", claims["type"])
	assert.Equal(t, float64(7), claims["user_id"])
}

func TestClientRefresh_EmployeeTokenRejected(t *testing.T) {
	// employee refresh token must not work on ClientRefresh
	empToken, err := generateToken(1, "emp@example.com", "refresh", []string{"ADMIN"}, "John", "Doe", "emp@example.com", 7*24*time.Hour)
	require.NoError(t, err)

	s := &AuthServer{}
	_, err = s.ClientRefresh(context.Background(), &pb_auth.ClientRefreshRequest{RefreshToken: empToken})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestClientRefresh_ExpiredToken(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": float64(1), "email": "ana@example.com",
		"role": "CLIENT", "type": "refresh",
		"exp": time.Now().Add(-time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))

	s := &AuthServer{}
	_, err := s.ClientRefresh(context.Background(), &pb_auth.ClientRefreshRequest{RefreshToken: tokenStr})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// ---- CreateActivationToken tests ----

func TestCreateActivationToken_DBFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("activation_tokens").
		WillReturnError(status.Error(codes.Internal, "db error"))

	s := newAuthServer(db, &mockEmployeeClient{}, &mockEmailClient{})
	_, err = s.CreateActivationToken(context.Background(), &pb_auth.CreateActivationTokenRequest{EmployeeId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateActivationToken_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("activation_tokens").
		WillReturnResult(sqlmock.NewResult(1, 1))

	s := newAuthServer(db, &mockEmployeeClient{}, &mockEmailClient{})
	resp, err := s.CreateActivationToken(context.Background(), &pb_auth.CreateActivationTokenRequest{EmployeeId: 1})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Len(t, resp.Token, 64) // 32 random bytes → 64 hex chars
}

// ---- RequestPasswordReset tests ----

func TestRequestPasswordReset_EmployeeNotFound(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeByEmail", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "not found"))

	s := newAuthServer(db, empClient, &mockEmailClient{})
	_, err = s.RequestPasswordReset(context.Background(), &pb_auth.RequestPasswordResetRequest{Email: "unknown@example.com"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestRequestPasswordReset_DBFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("password_reset_tokens").
		WillReturnError(status.Error(codes.Internal, "db error"))

	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeByEmail", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeByEmailResponse{Id: 1, FirstName: "John", Email: "john@example.com"}, nil,
	)

	s := newAuthServer(db, empClient, &mockEmailClient{})
	_, err = s.RequestPasswordReset(context.Background(), &pb_auth.RequestPasswordResetRequest{Email: "john@example.com"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestRequestPasswordReset_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("password_reset_tokens").
		WillReturnResult(sqlmock.NewResult(1, 1))

	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeByEmail", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeByEmailResponse{Id: 1, FirstName: "John", Email: "john@example.com"}, nil,
	)

	s := newAuthServer(db, empClient, &mockEmailClient{})
	resp, err := s.RequestPasswordReset(context.Background(), &pb_auth.RequestPasswordResetRequest{Email: "john@example.com"})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "John", resp.FirstName)
	assert.Equal(t, "john@example.com", resp.Email)
}

// ---- CreateClientActivationToken tests ----

func TestCreateClientActivationToken_DBFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("client_activation_tokens").
		WillReturnError(status.Error(codes.Internal, "db error"))

	s := &AuthServer{DB: db}
	_, err = s.CreateClientActivationToken(context.Background(), &pb_auth.CreateClientActivationTokenRequest{ClientId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCreateClientActivationToken_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectExec("client_activation_tokens").
		WillReturnResult(sqlmock.NewResult(1, 1))

	s := &AuthServer{DB: db}
	resp, err := s.CreateClientActivationToken(context.Background(), &pb_auth.CreateClientActivationTokenRequest{ClientId: 5})
	require.NoError(t, err)
	assert.Len(t, resp.Token, 64)
}

// ---- ActivateClient tests ----

func TestActivateClientAuth_TokenNotFound(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("client_activation_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "expires_at"}))

	s := &AuthServer{DB: db, ClientClient: &mockClientClient{}}
	_, err = s.ActivateClient(context.Background(), &pb_auth.ActivateClientRequest{
		Token: "no-such-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestActivateClientAuth_ExpiredToken(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("client_activation_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "expires_at"}).
			AddRow(int64(1), time.Now().Add(-time.Hour)))
	dbMock.ExpectExec("DELETE FROM client_activation_tokens").WillReturnResult(sqlmock.NewResult(1, 1))

	s := &AuthServer{DB: db, ClientClient: &mockClientClient{}}
	_, err = s.ActivateClient(context.Background(), &pb_auth.ActivateClientRequest{
		Token: "expired-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestActivateClientAuth_AlreadyActivated(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("client_activation_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "expires_at"}).
			AddRow(int64(1), time.Now().Add(time.Hour)))

	clientClient := &mockClientClient{}
	clientClient.On("GetClientById", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientByIdResponse{Client: &pb_client.Client{Id: 1, Active: true}}, nil,
	)

	s := &AuthServer{DB: db, ClientClient: clientClient}
	_, err = s.ActivateClient(context.Background(), &pb_auth.ActivateClientRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestActivateClientAuth_PasswordMismatch(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("client_activation_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "expires_at"}).
			AddRow(int64(1), time.Now().Add(time.Hour)))

	clientClient := &mockClientClient{}
	clientClient.On("GetClientById", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientByIdResponse{Client: &pb_client.Client{Id: 1, Active: false}}, nil,
	)

	s := &AuthServer{DB: db, ClientClient: clientClient}
	_, err = s.ActivateClient(context.Background(), &pb_auth.ActivateClientRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Different12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// ---- Additional edge-case tests ----

// Login: GetEmployeeById returns error
func TestLogin_GetEmployeeByIdError(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("Abcdef12"), bcrypt.MinCost)
	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeCredentials", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeCredentialsResponse{Id: 1, PasswordHash: string(hash), Aktivan: true, Dozvole: []string{}}, nil,
	)
	empClient.On("GetEmployeeById", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Internal, "db down"))

	s := &AuthServer{EmployeeClient: empClient, EmailClient: &mockEmailClient{}}
	_, err := s.Login(context.Background(), &pb_auth.LoginRequest{Email: "user@example.com", Password: "Abcdef12"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// Refresh: non-HMAC signing method
func TestRefresh_InvalidSigningMethod(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": float64(1), "username": "user@example.com",
		"type": "refresh", "exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)

	s := &AuthServer{}
	_, err := s.Refresh(context.Background(), &pb_auth.RefreshRequest{RefreshToken: tokenStr})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// Refresh: CLIENT role token rejected
func TestRefresh_ClientRoleRejected(t *testing.T) {
	clientToken, err := generateClientToken(1, "client@example.com", "refresh", "Ana", "Anić", 7*24*time.Hour)
	require.NoError(t, err)

	s := &AuthServer{}
	_, err = s.Refresh(context.Background(), &pb_auth.RefreshRequest{RefreshToken: clientToken})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// Refresh: user_id claim is not a float64
func TestRefresh_UserIdWrongType(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": "not-a-number", "username": "user@example.com",
		"type": "refresh", "exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))

	s := &AuthServer{}
	_, err := s.Refresh(context.Background(), &pb_auth.RefreshRequest{RefreshToken: tokenStr})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// Refresh: username claim is not a string
func TestRefresh_UsernameWrongType(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": float64(1), "username": 12345,
		"type": "refresh", "exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))

	s := &AuthServer{}
	_, err := s.Refresh(context.Background(), &pb_auth.RefreshRequest{RefreshToken: tokenStr})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// ActivateAccount: DB returns unexpected error (not ErrNoRows)
func TestActivateAccount_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("activation_tokens").WillReturnError(sql.ErrConnDone)

	s := newAuthServer(db, &mockEmployeeClient{}, &mockEmailClient{})
	_, err = s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "some-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ActivateAccount: GetEmployeeById returns error
func TestActivateAccount_GetEmployeeByIdError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("activation_tokens").WillReturnRows(rows)

	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeById", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Internal, "db down"))

	s := newAuthServer(db, empClient, &mockEmailClient{})
	_, err = s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ActivateAccount: ActivateEmployee returns error
func TestActivateAccount_ActivateEmployeeError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("activation_tokens").WillReturnRows(rows)

	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeById", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeByIdResponse{Employee: &pb_emp.Employee{Id: 1, Aktivan: false}}, nil,
	)
	empClient.On("ActivateEmployee", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Internal, "activate failed"))

	s := newAuthServer(db, empClient, &mockEmailClient{})
	_, err = s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ResetPassword: DB returns unexpected error
func TestResetPassword_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("password_reset_tokens").WillReturnError(sql.ErrConnDone)

	s := newAuthServer(db, &mockEmployeeClient{}, &mockEmailClient{})
	_, err = s.ResetPassword(context.Background(), &pb_auth.ResetPasswordRequest{
		Token: "some-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ResetPassword: UpdatePassword returns error
func TestResetPassword_UpdatePasswordError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("password_reset_tokens").WillReturnRows(rows)

	empClient := &mockEmployeeClient{}
	empClient.On("UpdatePassword", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Internal, "update failed"))

	s := newAuthServer(db, empClient, &mockEmailClient{})
	_, err = s.ResetPassword(context.Background(), &pb_auth.ResetPasswordRequest{
		Token: "good-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ClientLogin: GetClientById returns error
func TestClientLogin_GetClientByIdError(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("Abcdef12"), bcrypt.MinCost)
	clientClient := &mockClientClient{}
	clientClient.On("GetClientCredentials", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientCredentialsResponse{Id: 1, PasswordHash: string(hash), Active: true}, nil,
	)
	clientClient.On("GetClientById", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Internal, "db down"))

	s := &AuthServer{ClientClient: clientClient}
	_, err := s.ClientLogin(context.Background(), &pb_auth.ClientLoginRequest{Email: "ana@example.com", Password: "Abcdef12"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ClientRefresh: non-HMAC signing method
func TestClientRefresh_InvalidSigningMethod(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": float64(1), "email": "ana@example.com",
		"role": "CLIENT", "type": "refresh",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)

	s := &AuthServer{}
	_, err := s.ClientRefresh(context.Background(), &pb_auth.ClientRefreshRequest{RefreshToken: tokenStr})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// ClientRefresh: access token (type != refresh) rejected
func TestClientRefresh_AccessTokenRejected(t *testing.T) {
	accessToken, err := generateClientToken(1, "ana@example.com", "access", "Ana", "Anić", 15*time.Minute)
	require.NoError(t, err)

	s := &AuthServer{}
	_, err = s.ClientRefresh(context.Background(), &pb_auth.ClientRefreshRequest{RefreshToken: accessToken})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// ClientRefresh: non-CLIENT role rejected
func TestClientRefresh_NonClientRoleRejected(t *testing.T) {
	empToken, err := generateToken(1, "emp@example.com", "refresh", []string{"ADMIN"}, "John", "Doe", "emp@example.com", 7*24*time.Hour)
	require.NoError(t, err)

	s := &AuthServer{}
	_, err = s.ClientRefresh(context.Background(), &pb_auth.ClientRefreshRequest{RefreshToken: empToken})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// ClientRefresh: user_id not a float64
func TestClientRefresh_UserIdWrongType(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": "not-a-number", "email": "ana@example.com",
		"role": "CLIENT", "type": "refresh",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))

	s := &AuthServer{}
	_, err := s.ClientRefresh(context.Background(), &pb_auth.ClientRefreshRequest{RefreshToken: tokenStr})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// ActivateClient: DB returns unexpected error
func TestActivateClient_DBError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("client_activation_tokens").WillReturnError(sql.ErrConnDone)

	s := &AuthServer{DB: db, ClientClient: &mockClientClient{}}
	_, err = s.ActivateClient(context.Background(), &pb_auth.ActivateClientRequest{
		Token: "some-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ActivateClient: GetClientById returns error
func TestActivateClient_GetClientByIdError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("client_activation_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "expires_at"}).
			AddRow(int64(1), time.Now().Add(time.Hour)))

	clientClient := &mockClientClient{}
	clientClient.On("GetClientById", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Internal, "db down"))

	s := &AuthServer{DB: db, ClientClient: clientClient}
	_, err = s.ActivateClient(context.Background(), &pb_auth.ActivateClientRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ActivateClient: invalid password
func TestActivateClient_InvalidPassword(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("client_activation_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "expires_at"}).
			AddRow(int64(1), time.Now().Add(time.Hour)))

	clientClient := &mockClientClient{}
	clientClient.On("GetClientById", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientByIdResponse{Client: &pb_client.Client{Id: 1, Active: false}}, nil,
	)

	s := &AuthServer{DB: db, ClientClient: clientClient}
	_, err = s.ActivateClient(context.Background(), &pb_auth.ActivateClientRequest{
		Token: "valid-token", Password: "weak", ConfirmPassword: "weak",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// ActivateClient: ActivateClient gRPC call returns error
func TestActivateClient_ActivateClientError(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("client_activation_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "expires_at"}).
			AddRow(int64(1), time.Now().Add(time.Hour)))

	clientClient := &mockClientClient{}
	clientClient.On("GetClientById", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientByIdResponse{Client: &pb_client.Client{Id: 1, Active: false}}, nil,
	)
	clientClient.On("ActivateClient", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Internal, "activate failed"))

	s := &AuthServer{DB: db, ClientClient: clientClient}
	_, err = s.ActivateClient(context.Background(), &pb_auth.ActivateClientRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ClientLogin: empty password hash
func TestClientLogin_EmptyPasswordHash(t *testing.T) {
	clientClient := &mockClientClient{}
	clientClient.On("GetClientCredentials", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientCredentialsResponse{Id: 1, PasswordHash: "", Active: true}, nil,
	)

	s := &AuthServer{ClientClient: clientClient}
	_, err := s.ClientLogin(context.Background(), &pb_auth.ClientLoginRequest{Email: "ana@example.com", Password: "Abcdef12"})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// ActivateAccount: expired token DELETE fails — op still returns FailedPrecondition
func TestActivateAccount_ExpiredToken_DeleteFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(-time.Hour))
	dbMock.ExpectQuery("activation_tokens").WillReturnRows(rows)
	dbMock.ExpectExec("DELETE FROM activation_tokens").WillReturnError(sql.ErrConnDone) // logged, not returned

	s := newAuthServer(db, &mockEmployeeClient{}, &mockEmailClient{})
	_, err = s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "expired-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// ActivateAccount: used-token DELETE fails — op still succeeds
func TestActivateAccount_DeleteUsedTokenFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("activation_tokens").WillReturnRows(rows)
	dbMock.ExpectExec("DELETE FROM activation_tokens").WillReturnError(sql.ErrConnDone) // logged, not returned

	empClient := &mockEmployeeClient{}
	empClient.On("GetEmployeeById", mock.Anything, mock.Anything).Return(
		&pb_emp.GetEmployeeByIdResponse{Employee: &pb_emp.Employee{
			Id: 1, Aktivan: false, Email: "emp@example.com", Ime: "John",
		}}, nil,
	)
	empClient.On("ActivateEmployee", mock.Anything, mock.Anything).Return(
		&pb_emp.ActivateEmployeeResponse{}, nil,
	)

	emailClient := &mockEmailClient{}
	emailClient.On("SendPasswordConfirmationEmail", mock.Anything, mock.Anything).
		Return(&pb_email.SendActivationEmailResponse{}, nil)

	s := newAuthServer(db, empClient, emailClient)
	resp, err := s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	time.Sleep(50 * time.Millisecond)
}

// ResetPassword: used-token DELETE fails — op still succeeds
func TestResetPassword_DeleteUsedTokenFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"employee_id", "expires_at"}).
		AddRow(int64(1), time.Now().Add(time.Hour))
	dbMock.ExpectQuery("password_reset_tokens").WillReturnRows(rows)
	dbMock.ExpectExec("DELETE FROM password_reset_tokens").WillReturnError(sql.ErrConnDone) // logged

	empClient := &mockEmployeeClient{}
	empClient.On("UpdatePassword", mock.Anything, mock.Anything).Return(
		&pb_emp.UpdatePasswordResponse{}, nil,
	)

	s := newAuthServer(db, empClient, &mockEmailClient{})
	resp, err := s.ResetPassword(context.Background(), &pb_auth.ResetPasswordRequest{
		Token: "good-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ActivateClient: used-token DELETE fails — op still succeeds
func TestActivateClient_DeleteUsedTokenFails(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("client_activation_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "expires_at"}).
			AddRow(int64(1), time.Now().Add(time.Hour)))
	dbMock.ExpectExec("DELETE FROM client_activation_tokens").WillReturnError(sql.ErrConnDone) // logged

	clientClient := &mockClientClient{}
	clientClient.On("GetClientById", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientByIdResponse{Client: &pb_client.Client{
			Id: 1, Active: false, Email: "ana@example.com", FirstName: "Ana",
		}}, nil,
	)
	clientClient.On("ActivateClient", mock.Anything, mock.Anything).Return(
		&pb_client.ActivateClientResponse{}, nil,
	)

	emailClient := &mockEmailClient{}
	emailClient.On("SendPasswordConfirmationEmail", mock.Anything, mock.Anything).Return(
		&pb_email.SendActivationEmailResponse{}, nil,
	)

	s := &AuthServer{DB: db, ClientClient: clientClient, EmailClient: emailClient}
	resp, err := s.ActivateClient(context.Background(), &pb_auth.ActivateClientRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	time.Sleep(50 * time.Millisecond)
}

func TestActivateClientAuth_HappyPath(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectQuery("client_activation_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "expires_at"}).
			AddRow(int64(1), time.Now().Add(time.Hour)))
	dbMock.ExpectExec("DELETE FROM client_activation_tokens").WillReturnResult(sqlmock.NewResult(1, 1))

	clientClient := &mockClientClient{}
	clientClient.On("GetClientById", mock.Anything, mock.Anything).Return(
		&pb_client.GetClientByIdResponse{Client: &pb_client.Client{
			Id: 1, Active: false, Email: "ana@example.com", FirstName: "Ana",
		}}, nil,
	)
	clientClient.On("ActivateClient", mock.Anything, mock.Anything).Return(
		&pb_client.ActivateClientResponse{}, nil,
	)

	emailClient := &mockEmailClient{}
	emailClient.On("SendPasswordConfirmationEmail", mock.Anything, mock.Anything).Return(
		&pb_email.SendActivationEmailResponse{}, nil,
	)

	s := &AuthServer{DB: db, ClientClient: clientClient, EmailClient: emailClient}
	resp, err := s.ActivateClient(context.Background(), &pb_auth.ActivateClientRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	time.Sleep(50 * time.Millisecond) // wait for email goroutine
	clientClient.AssertExpectations(t)
	emailClient.AssertExpectations(t)
}
