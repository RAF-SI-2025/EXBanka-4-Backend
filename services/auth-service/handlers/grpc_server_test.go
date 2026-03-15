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
		Maybe().Return(&pb_email.SendActivationEmailResponse{}, nil)

	s := newAuthServer(db, empClient, emailClient)
	resp, err := s.ActivateAccount(context.Background(), &pb_auth.ActivateAccountRequest{
		Token: "valid-token", Password: "Abcdef12", ConfirmPassword: "Abcdef12",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	empClient.AssertExpectations(t)
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
