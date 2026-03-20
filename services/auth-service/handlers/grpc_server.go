package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb_auth "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_email "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
)

const jwtSecret = "secret-key-change-in-production"

type AuthServer struct {
	pb_auth.UnimplementedAuthServiceServer
	DB             *sql.DB
	EmployeeClient pb_emp.EmployeeServiceClient
	EmailClient    pb_email.EmailServiceClient
	ClientClient   pb_client.ClientServiceClient
}

func (s *AuthServer) Login(ctx context.Context, req *pb_auth.LoginRequest) (*pb_auth.LoginResponse, error) {
	creds, err := s.EmployeeClient.GetEmployeeCredentials(ctx, &pb_emp.GetEmployeeCredentialsRequest{
		Email: req.Email,
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	if creds.PasswordHash == "" {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if !creds.Aktivan {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(creds.PasswordHash), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	empResp, err := s.EmployeeClient.GetEmployeeById(ctx, &pb_emp.GetEmployeeByIdRequest{Id: creds.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch employee")
	}
	emp := empResp.Employee

	accessToken, err := generateToken(creds.Id, emp.Email, "access", creds.Dozvole, emp.Ime, emp.Prezime, emp.Email, 15*time.Minute)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	refreshToken, err := generateToken(creds.Id, emp.Email, "refresh", creds.Dozvole, emp.Ime, emp.Prezime, emp.Email, 7*24*time.Hour)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &pb_auth.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthServer) Refresh(_ context.Context, req *pb_auth.RefreshRequest) (*pb_auth.RefreshResponse, error) {
	token, err := jwt.Parse(req.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid token claims")
	}

	if claims["type"] != "refresh" {
		return nil, status.Error(codes.Unauthenticated, "invalid token type")
	}
	if claims["role"] == "CLIENT" {
		return nil, status.Error(codes.Unauthenticated, "invalid token type")
	}

	userIDRaw, ok := claims["user_id"].(float64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid token claims")
	}
	usernameRaw, ok := claims["username"].(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid token claims")
	}
	userID := int64(userIDRaw)
	username := usernameRaw

	firstName, _ := claims["first_name"].(string)
	lastName, _ := claims["last_name"].(string)
	email, _ := claims["email"].(string)

	var dozvole []string
	if raw, ok := claims["dozvole"].([]interface{}); ok {
		for _, d := range raw {
			if s, ok := d.(string); ok {
				dozvole = append(dozvole, s)
			}
		}
	}

	accessToken, err := generateToken(userID, username, "access", dozvole, firstName, lastName, email, 15*time.Minute)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &pb_auth.RefreshResponse{AccessToken: accessToken}, nil
}

func generateToken(userID int64, username, tokenType string, dozvole []string, firstName, lastName, email string, d time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id":    userID,
		"username":   username,
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
		"type":       tokenType,
		"dozvole":    dozvole,
		"exp":        time.Now().Add(d).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
}

func (s *AuthServer) CreateActivationToken(ctx context.Context, req *pb_auth.CreateActivationTokenRequest) (*pb_auth.CreateActivationTokenResponse, error) {
	token, err := generateActivationToken()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO activation_tokens (token, employee_id, expires_at) VALUES ($1, $2, now() + interval '24 hours')`,
		token, req.EmployeeId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store activation token: %v", err)
	}
	return &pb_auth.CreateActivationTokenResponse{Token: token}, nil
}

func (s *AuthServer) ActivateAccount(ctx context.Context, req *pb_auth.ActivateAccountRequest) (*pb_auth.ActivateAccountResponse, error) {
	var employeeID int64
	var expiresAt time.Time
	err := s.DB.QueryRowContext(ctx,
		`SELECT employee_id, expires_at FROM activation_tokens WHERE token = $1`,
		req.Token,
	).Scan(&employeeID, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "invalid or expired token")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to look up token: %v", err)
	}

	if time.Now().After(expiresAt) {
		if _, err := s.DB.ExecContext(ctx, `DELETE FROM activation_tokens WHERE token = $1`, req.Token); err != nil {
			log.Printf("failed to delete expired activation token: %v", err)
		}
		return nil, status.Error(codes.FailedPrecondition, "activation token has expired")
	}

	empResp, err := s.EmployeeClient.GetEmployeeById(ctx, &pb_emp.GetEmployeeByIdRequest{Id: employeeID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch employee: %v", err)
	}
	if empResp.Employee.Aktivan {
		return nil, status.Error(codes.FailedPrecondition, "account already activated")
	}

	if req.Password != req.ConfirmPassword {
		return nil, status.Error(codes.InvalidArgument, "passwords do not match")
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	_, err = s.EmployeeClient.ActivateEmployee(ctx, &pb_emp.ActivateEmployeeRequest{
		EmployeeId:   employeeID,
		PasswordHash: string(hash),
	})
	if err != nil {
		return nil, err
	}

	if _, err := s.DB.ExecContext(ctx, `DELETE FROM activation_tokens WHERE token = $1`, req.Token); err != nil {
		log.Printf("failed to delete used activation token: %v", err)
	}

	emp := empResp.Employee
	go func() {
		_, err := s.EmailClient.SendPasswordConfirmationEmail(context.Background(), &pb_email.SendActivationEmailRequest{
			Email:     emp.Email,
			FirstName: emp.Ime,
		})
		if err != nil {
			log.Printf("failed to send password confirmation email: %v", err)
		}
	}()

	return &pb_auth.ActivateAccountResponse{}, nil
}

func (s *AuthServer) RequestPasswordReset(ctx context.Context, req *pb_auth.RequestPasswordResetRequest) (*pb_auth.RequestPasswordResetResponse, error) {
	empResp, err := s.EmployeeClient.GetEmployeeByEmail(ctx, &pb_emp.GetEmployeeByEmailRequest{Email: req.Email})
	if err != nil {
		return nil, err
	}

	token, err := generateActivationToken()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (token, employee_id, expires_at) VALUES ($1, $2, now() + interval '24 hours')`,
		token, empResp.Id,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store password reset token: %v", err)
	}

	return &pb_auth.RequestPasswordResetResponse{
		Token:     token,
		FirstName: empResp.FirstName,
		Email:     empResp.Email,
	}, nil
}

func (s *AuthServer) ResetPassword(ctx context.Context, req *pb_auth.ResetPasswordRequest) (*pb_auth.ResetPasswordResponse, error) {
	var employeeID int64
	var expiresAt time.Time
	err := s.DB.QueryRowContext(ctx,
		`SELECT employee_id, expires_at FROM password_reset_tokens WHERE token = $1`,
		req.Token,
	).Scan(&employeeID, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "invalid or expired token")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to look up token: %v", err)
	}

	if time.Now().After(expiresAt) {
		if _, err := s.DB.ExecContext(ctx, `DELETE FROM password_reset_tokens WHERE token = $1`, req.Token); err != nil {
			log.Printf("failed to delete expired password reset token: %v", err)
		}
		return nil, status.Error(codes.FailedPrecondition, "password reset token has expired")
	}

	if req.Password != req.ConfirmPassword {
		return nil, status.Error(codes.InvalidArgument, "passwords do not match")
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	_, err = s.EmployeeClient.UpdatePassword(ctx, &pb_emp.UpdatePasswordRequest{
		EmployeeId:   employeeID,
		PasswordHash: string(hash),
	})
	if err != nil {
		return nil, err
	}

	if _, err := s.DB.ExecContext(ctx, `DELETE FROM password_reset_tokens WHERE token = $1`, req.Token); err != nil {
		log.Printf("failed to delete used password reset token: %v", err)
	}
	return &pb_auth.ResetPasswordResponse{}, nil
}

func validatePassword(p string) error {
	if len(p) < 8 {
		return status.Error(codes.InvalidArgument, "password must be at least 8 characters")
	}
	if len(p) > 32 {
		return status.Error(codes.InvalidArgument, "password must be at most 32 characters")
	}
	var digits, upper, lower int
	for _, r := range p {
		switch {
		case unicode.IsDigit(r):
			digits++
		case unicode.IsUpper(r):
			upper++
		case unicode.IsLower(r):
			lower++
		}
	}
	if digits < 2 {
		return status.Error(codes.InvalidArgument, "password must contain at least 2 numbers")
	}
	if upper < 1 {
		return status.Error(codes.InvalidArgument, "password must contain at least 1 uppercase letter")
	}
	if lower < 1 {
		return status.Error(codes.InvalidArgument, "password must contain at least 1 lowercase letter")
	}
	return nil
}

func generateClientToken(userID int64, email, tokenType, firstName, lastName string, d time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id":    userID,
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"role":       "CLIENT",
		"type":       tokenType,
		"exp":        time.Now().Add(d).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
}

func (s *AuthServer) ClientLogin(ctx context.Context, req *pb_auth.ClientLoginRequest) (*pb_auth.ClientLoginResponse, error) {
	creds, err := s.ClientClient.GetClientCredentials(ctx, &pb_client.GetClientCredentialsRequest{
		Email: req.Email,
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	if creds.PasswordHash == "" {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if !creds.Active {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(creds.PasswordHash), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	clientResp, err := s.ClientClient.GetClientById(ctx, &pb_client.GetClientByIdRequest{Id: creds.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch client")
	}
	cl := clientResp.Client

	accessToken, err := generateClientToken(creds.Id, cl.Email, "access", cl.FirstName, cl.LastName, 15*time.Minute)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	refreshToken, err := generateClientToken(creds.Id, cl.Email, "refresh", cl.FirstName, cl.LastName, 7*24*time.Hour)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &pb_auth.ClientLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthServer) ClientRefresh(_ context.Context, req *pb_auth.ClientRefreshRequest) (*pb_auth.ClientRefreshResponse, error) {
	token, err := jwt.Parse(req.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid token claims")
	}
	if claims["type"] != "refresh" {
		return nil, status.Error(codes.Unauthenticated, "invalid token type")
	}
	if claims["role"] != "CLIENT" {
		return nil, status.Error(codes.Unauthenticated, "invalid token type")
	}

	userIDRaw, ok := claims["user_id"].(float64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid token claims")
	}
	email, _ := claims["email"].(string)
	firstName, _ := claims["first_name"].(string)
	lastName, _ := claims["last_name"].(string)

	accessToken, err := generateClientToken(int64(userIDRaw), email, "access", firstName, lastName, 15*time.Minute)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &pb_auth.ClientRefreshResponse{AccessToken: accessToken}, nil
}

func (s *AuthServer) CreateClientActivationToken(ctx context.Context, req *pb_auth.CreateClientActivationTokenRequest) (*pb_auth.CreateClientActivationTokenResponse, error) {
	token, err := generateActivationToken()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO client_activation_tokens (token, client_id, expires_at) VALUES ($1, $2, now() + interval '24 hours')`,
		token, req.ClientId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store client activation token: %v", err)
	}
	return &pb_auth.CreateClientActivationTokenResponse{Token: token}, nil
}

func (s *AuthServer) ActivateClient(ctx context.Context, req *pb_auth.ActivateClientRequest) (*pb_auth.ActivateClientResponse, error) {
	var clientID int64
	var expiresAt time.Time
	err := s.DB.QueryRowContext(ctx,
		`SELECT client_id, expires_at FROM client_activation_tokens WHERE token = $1`,
		req.Token,
	).Scan(&clientID, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "invalid or expired token")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to look up token: %v", err)
	}

	if time.Now().After(expiresAt) {
		if _, err := s.DB.ExecContext(ctx, `DELETE FROM client_activation_tokens WHERE token = $1`, req.Token); err != nil {
			log.Printf("failed to delete expired client activation token: %v", err)
		}
		return nil, status.Error(codes.FailedPrecondition, "activation token has expired")
	}

	clientResp, err := s.ClientClient.GetClientById(ctx, &pb_client.GetClientByIdRequest{Id: clientID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch client: %v", err)
	}
	if clientResp.Client.Active {
		return nil, status.Error(codes.FailedPrecondition, "account already activated")
	}

	if req.Password != req.ConfirmPassword {
		return nil, status.Error(codes.InvalidArgument, "passwords do not match")
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	if _, err := s.ClientClient.ActivateClient(ctx, &pb_client.ActivateClientRequest{
		ClientId:     clientID,
		PasswordHash: string(hash),
	}); err != nil {
		return nil, err
	}

	if _, err := s.DB.ExecContext(ctx, `DELETE FROM client_activation_tokens WHERE token = $1`, req.Token); err != nil {
		log.Printf("failed to delete used client activation token: %v", err)
	}

	cl := clientResp.Client
	go func() {
		_, err := s.EmailClient.SendPasswordConfirmationEmail(context.Background(), &pb_email.SendActivationEmailRequest{
			Email:     cl.Email,
			FirstName: cl.FirstName,
		})
		if err != nil {
			log.Printf("failed to send password confirmation email to client: %v", err)
		}
	}()

	return &pb_auth.ActivateClientResponse{}, nil
}

func generateActivationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
