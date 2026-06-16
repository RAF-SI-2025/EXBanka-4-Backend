package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	emailpb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_fund "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- stub employee client ----

type stubEmpClient struct {
	getAllFn                    func(context.Context, *pb.GetAllEmployeesRequest, ...grpc.CallOption) (*pb.GetAllEmployeesResponse, error)
	searchFn                    func(context.Context, *pb.SearchEmployeesRequest, ...grpc.CallOption) (*pb.SearchEmployeesResponse, error)
	credentialsFn               func(context.Context, *pb.GetEmployeeCredentialsRequest, ...grpc.CallOption) (*pb.GetEmployeeCredentialsResponse, error)
	createFn                    func(context.Context, *pb.CreateEmployeeRequest, ...grpc.CallOption) (*pb.CreateEmployeeResponse, error)
	getByIdFn                   func(context.Context, *pb.GetEmployeeByIdRequest, ...grpc.CallOption) (*pb.GetEmployeeByIdResponse, error)
	updateFn                    func(context.Context, *pb.UpdateEmployeeRequest, ...grpc.CallOption) (*pb.UpdateEmployeeResponse, error)
	activateFn                  func(context.Context, *pb.ActivateEmployeeRequest, ...grpc.CallOption) (*pb.ActivateEmployeeResponse, error)
	getByEmailFn                func(context.Context, *pb.GetEmployeeByEmailRequest, ...grpc.CallOption) (*pb.GetEmployeeByEmailResponse, error)
	updatePasswordFn            func(context.Context, *pb.UpdatePasswordRequest, ...grpc.CallOption) (*pb.UpdatePasswordResponse, error)
	getActuariesFn              func(context.Context, *pb.GetActuariesRequest, ...grpc.CallOption) (*pb.GetActuariesResponse, error)
	setAgentLimitFn             func(context.Context, *pb.SetAgentLimitRequest, ...grpc.CallOption) (*pb.SetAgentLimitResponse, error)
	resetUsedLimitFn            func(context.Context, *pb.ResetAgentUsedLimitRequest, ...grpc.CallOption) (*pb.ResetAgentUsedLimitResponse, error)
	setNeedApprovalFn           func(context.Context, *pb.SetNeedApprovalRequest, ...grpc.CallOption) (*pb.SetNeedApprovalResponse, error)
	resetAllActuaryUsedLimitsFn func(context.Context, *pb.ResetAllActuaryUsedLimitsRequest, ...grpc.CallOption) (*pb.ResetAllActuaryUsedLimitsResponse, error)
	getActuaryPerformersFn      func(context.Context, *pb.GetActuaryPerformersRequest, ...grpc.CallOption) (*pb.GetActuaryPerformersResponse, error)
}

func (s *stubEmpClient) GetAllEmployees(ctx context.Context, in *pb.GetAllEmployeesRequest, opts ...grpc.CallOption) (*pb.GetAllEmployeesResponse, error) {
	return s.getAllFn(ctx, in, opts...)
}
func (s *stubEmpClient) SearchEmployees(ctx context.Context, in *pb.SearchEmployeesRequest, opts ...grpc.CallOption) (*pb.SearchEmployeesResponse, error) {
	return s.searchFn(ctx, in, opts...)
}
func (s *stubEmpClient) GetEmployeeCredentials(ctx context.Context, in *pb.GetEmployeeCredentialsRequest, opts ...grpc.CallOption) (*pb.GetEmployeeCredentialsResponse, error) {
	if s.credentialsFn != nil {
		return s.credentialsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) CreateEmployee(ctx context.Context, in *pb.CreateEmployeeRequest, opts ...grpc.CallOption) (*pb.CreateEmployeeResponse, error) {
	return s.createFn(ctx, in, opts...)
}
func (s *stubEmpClient) GetEmployeeById(ctx context.Context, in *pb.GetEmployeeByIdRequest, opts ...grpc.CallOption) (*pb.GetEmployeeByIdResponse, error) {
	return s.getByIdFn(ctx, in, opts...)
}
func (s *stubEmpClient) UpdateEmployee(ctx context.Context, in *pb.UpdateEmployeeRequest, opts ...grpc.CallOption) (*pb.UpdateEmployeeResponse, error) {
	return s.updateFn(ctx, in, opts...)
}
func (s *stubEmpClient) ActivateEmployee(ctx context.Context, in *pb.ActivateEmployeeRequest, opts ...grpc.CallOption) (*pb.ActivateEmployeeResponse, error) {
	if s.activateFn != nil {
		return s.activateFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) GetEmployeeByEmail(ctx context.Context, in *pb.GetEmployeeByEmailRequest, opts ...grpc.CallOption) (*pb.GetEmployeeByEmailResponse, error) {
	if s.getByEmailFn != nil {
		return s.getByEmailFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) UpdatePassword(ctx context.Context, in *pb.UpdatePasswordRequest, opts ...grpc.CallOption) (*pb.UpdatePasswordResponse, error) {
	if s.updatePasswordFn != nil {
		return s.updatePasswordFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) GetActuaries(ctx context.Context, in *pb.GetActuariesRequest, opts ...grpc.CallOption) (*pb.GetActuariesResponse, error) {
	if s.getActuariesFn != nil {
		return s.getActuariesFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) SetAgentLimit(ctx context.Context, in *pb.SetAgentLimitRequest, opts ...grpc.CallOption) (*pb.SetAgentLimitResponse, error) {
	if s.setAgentLimitFn != nil {
		return s.setAgentLimitFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) ResetAgentUsedLimit(ctx context.Context, in *pb.ResetAgentUsedLimitRequest, opts ...grpc.CallOption) (*pb.ResetAgentUsedLimitResponse, error) {
	if s.resetUsedLimitFn != nil {
		return s.resetUsedLimitFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) SetNeedApproval(ctx context.Context, in *pb.SetNeedApprovalRequest, opts ...grpc.CallOption) (*pb.SetNeedApprovalResponse, error) {
	if s.setNeedApprovalFn != nil {
		return s.setNeedApprovalFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) ResetAllActuaryUsedLimits(ctx context.Context, in *pb.ResetAllActuaryUsedLimitsRequest, opts ...grpc.CallOption) (*pb.ResetAllActuaryUsedLimitsResponse, error) {
	if s.resetAllActuaryUsedLimitsFn != nil {
		return s.resetAllActuaryUsedLimitsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) GetActuaryPerformers(ctx context.Context, in *pb.GetActuaryPerformersRequest, opts ...grpc.CallOption) (*pb.GetActuaryPerformersResponse, error) {
	if s.getActuaryPerformersFn != nil {
		return s.getActuaryPerformersFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) GetSupervisors(ctx context.Context, in *pb.GetSupervisorsRequest, opts ...grpc.CallOption) (*pb.GetSupervisorsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) UpdateLoginAttempts(ctx context.Context, in *pb.UpdateLoginAttemptsRequest, opts ...grpc.CallOption) (*pb.UpdateLoginAttemptsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) LogAuditEvent(ctx context.Context, in *pb.AuditLogRequest, opts ...grpc.CallOption) (*pb.AuditLogResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmpClient) ListAuditLogs(ctx context.Context, in *pb.ListAuditLogsRequest, opts ...grpc.CallOption) (*pb.ListAuditLogsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// ---- stub auth client ----

type stubAuthClient struct {
	createActivationTokenFn       func(context.Context, *authpb.CreateActivationTokenRequest, ...grpc.CallOption) (*authpb.CreateActivationTokenResponse, error)
	loginFn                       func(context.Context, *authpb.LoginRequest, ...grpc.CallOption) (*authpb.LoginResponse, error)
	refreshFn                     func(context.Context, *authpb.RefreshRequest, ...grpc.CallOption) (*authpb.RefreshResponse, error)
	activateAccountFn             func(context.Context, *authpb.ActivateAccountRequest, ...grpc.CallOption) (*authpb.ActivateAccountResponse, error)
	requestPasswordResetFn        func(context.Context, *authpb.RequestPasswordResetRequest, ...grpc.CallOption) (*authpb.RequestPasswordResetResponse, error)
	resetPasswordFn               func(context.Context, *authpb.ResetPasswordRequest, ...grpc.CallOption) (*authpb.ResetPasswordResponse, error)
	clientLoginFn                 func(context.Context, *authpb.ClientLoginRequest, ...grpc.CallOption) (*authpb.ClientLoginResponse, error)
	clientRefreshFn               func(context.Context, *authpb.ClientRefreshRequest, ...grpc.CallOption) (*authpb.ClientRefreshResponse, error)
	activateClientFn              func(context.Context, *authpb.ActivateClientRequest, ...grpc.CallOption) (*authpb.ActivateClientResponse, error)
	pollApprovalFn                func(context.Context, *authpb.PollApprovalRequest, ...grpc.CallOption) (*authpb.PollApprovalResponse, error)
	createApprovalFn              func(context.Context, *authpb.CreateApprovalRequest, ...grpc.CallOption) (*authpb.CreateApprovalResponse, error)
	getApprovalFn                 func(context.Context, *authpb.GetApprovalRequest, ...grpc.CallOption) (*authpb.GetApprovalResponse, error)
	getClientApprovalsFn          func(context.Context, *authpb.GetClientApprovalsRequest, ...grpc.CallOption) (*authpb.GetClientApprovalsResponse, error)
	updateApprovalStatusFn        func(context.Context, *authpb.UpdateApprovalStatusRequest, ...grpc.CallOption) (*authpb.UpdateApprovalStatusResponse, error)
	registerPushTokenFn           func(context.Context, *authpb.RegisterPushTokenRequest, ...grpc.CallOption) (*authpb.RegisterPushTokenResponse, error)
	unregisterPushTokenFn         func(context.Context, *authpb.UnregisterPushTokenRequest, ...grpc.CallOption) (*authpb.UnregisterPushTokenResponse, error)
	getPushTokenFn                func(context.Context, *authpb.GetPushTokenRequest, ...grpc.CallOption) (*authpb.GetPushTokenResponse, error)
	createClientActivationTokenFn func(context.Context, *authpb.CreateClientActivationTokenRequest, ...grpc.CallOption) (*authpb.CreateClientActivationTokenResponse, error)
}

func (s *stubAuthClient) Login(ctx context.Context, in *authpb.LoginRequest, opts ...grpc.CallOption) (*authpb.LoginResponse, error) {
	if s.loginFn != nil {
		return s.loginFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) Refresh(ctx context.Context, in *authpb.RefreshRequest, opts ...grpc.CallOption) (*authpb.RefreshResponse, error) {
	if s.refreshFn != nil {
		return s.refreshFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) CreateActivationToken(ctx context.Context, in *authpb.CreateActivationTokenRequest, opts ...grpc.CallOption) (*authpb.CreateActivationTokenResponse, error) {
	return s.createActivationTokenFn(ctx, in, opts...)
}
func (s *stubAuthClient) ActivateAccount(ctx context.Context, in *authpb.ActivateAccountRequest, opts ...grpc.CallOption) (*authpb.ActivateAccountResponse, error) {
	if s.activateAccountFn != nil {
		return s.activateAccountFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) RequestPasswordReset(ctx context.Context, in *authpb.RequestPasswordResetRequest, opts ...grpc.CallOption) (*authpb.RequestPasswordResetResponse, error) {
	if s.requestPasswordResetFn != nil {
		return s.requestPasswordResetFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) ResetPassword(ctx context.Context, in *authpb.ResetPasswordRequest, opts ...grpc.CallOption) (*authpb.ResetPasswordResponse, error) {
	if s.resetPasswordFn != nil {
		return s.resetPasswordFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) ClientLogin(ctx context.Context, in *authpb.ClientLoginRequest, opts ...grpc.CallOption) (*authpb.ClientLoginResponse, error) {
	if s.clientLoginFn != nil {
		return s.clientLoginFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) ClientRefresh(ctx context.Context, in *authpb.ClientRefreshRequest, opts ...grpc.CallOption) (*authpb.ClientRefreshResponse, error) {
	if s.clientRefreshFn != nil {
		return s.clientRefreshFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) CreateClientActivationToken(ctx context.Context, in *authpb.CreateClientActivationTokenRequest, opts ...grpc.CallOption) (*authpb.CreateClientActivationTokenResponse, error) {
	if s.createClientActivationTokenFn != nil {
		return s.createClientActivationTokenFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) ActivateClient(ctx context.Context, in *authpb.ActivateClientRequest, opts ...grpc.CallOption) (*authpb.ActivateClientResponse, error) {
	if s.activateClientFn != nil {
		return s.activateClientFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) PollApproval(ctx context.Context, in *authpb.PollApprovalRequest, opts ...grpc.CallOption) (*authpb.PollApprovalResponse, error) {
	if s.pollApprovalFn != nil {
		return s.pollApprovalFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) CreateApproval(ctx context.Context, in *authpb.CreateApprovalRequest, opts ...grpc.CallOption) (*authpb.CreateApprovalResponse, error) {
	if s.createApprovalFn != nil {
		return s.createApprovalFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) GetApproval(ctx context.Context, in *authpb.GetApprovalRequest, opts ...grpc.CallOption) (*authpb.GetApprovalResponse, error) {
	if s.getApprovalFn != nil {
		return s.getApprovalFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) GetClientApprovals(ctx context.Context, in *authpb.GetClientApprovalsRequest, opts ...grpc.CallOption) (*authpb.GetClientApprovalsResponse, error) {
	if s.getClientApprovalsFn != nil {
		return s.getClientApprovalsFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) UpdateApprovalStatus(ctx context.Context, in *authpb.UpdateApprovalStatusRequest, opts ...grpc.CallOption) (*authpb.UpdateApprovalStatusResponse, error) {
	if s.updateApprovalStatusFn != nil {
		return s.updateApprovalStatusFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) RegisterPushToken(ctx context.Context, in *authpb.RegisterPushTokenRequest, opts ...grpc.CallOption) (*authpb.RegisterPushTokenResponse, error) {
	if s.registerPushTokenFn != nil {
		return s.registerPushTokenFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) UnregisterPushToken(ctx context.Context, in *authpb.UnregisterPushTokenRequest, opts ...grpc.CallOption) (*authpb.UnregisterPushTokenResponse, error) {
	if s.unregisterPushTokenFn != nil {
		return s.unregisterPushTokenFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) GetPushToken(ctx context.Context, in *authpb.GetPushTokenRequest, opts ...grpc.CallOption) (*authpb.GetPushTokenResponse, error) {
	if s.getPushTokenFn != nil {
		return s.getPushTokenFn(ctx, in, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) Logout(ctx context.Context, in *authpb.LogoutRequest, opts ...grpc.CallOption) (*authpb.LogoutResponse, error) {
	return &authpb.LogoutResponse{}, nil
}
func (s *stubAuthClient) GenerateTOTPSecret(ctx context.Context, in *authpb.GenerateTOTPRequest, opts ...grpc.CallOption) (*authpb.GenerateTOTPResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) VerifyTOTP(ctx context.Context, in *authpb.VerifyTOTPRequest, opts ...grpc.CallOption) (*authpb.VerifyTOTPResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) EnableTOTP(ctx context.Context, in *authpb.EnableTOTPRequest, opts ...grpc.CallOption) (*authpb.EnableTOTPResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) ValidateTOTPLogin(ctx context.Context, in *authpb.ValidateTOTPLoginRequest, opts ...grpc.CallOption) (*authpb.ValidateTOTPLoginResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) DisableTOTP(ctx context.Context, in *authpb.DisableTOTPRequest, opts ...grpc.CallOption) (*authpb.DisableTOTPResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) CreateNotification(ctx context.Context, in *authpb.CreateNotificationRequest, opts ...grpc.CallOption) (*authpb.CreateNotificationResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) ListNotifications(ctx context.Context, in *authpb.ListNotificationsRequest, opts ...grpc.CallOption) (*authpb.ListNotificationsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) MarkNotificationRead(ctx context.Context, in *authpb.MarkNotificationReadRequest, opts ...grpc.CallOption) (*authpb.MarkNotificationReadResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) MarkAllRead(ctx context.Context, in *authpb.MarkAllReadRequest, opts ...grpc.CallOption) (*authpb.MarkAllReadResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubAuthClient) GetUnreadCount(ctx context.Context, in *authpb.GetUnreadCountRequest, opts ...grpc.CallOption) (*authpb.GetUnreadCountResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// ---- stub email client ----

type stubEmailClient struct {
	sendActivationFn func(context.Context, *emailpb.SendActivationEmailRequest, ...grpc.CallOption) (*emailpb.SendActivationEmailResponse, error)
}

func (s *stubEmailClient) SendActivationEmail(ctx context.Context, in *emailpb.SendActivationEmailRequest, opts ...grpc.CallOption) (*emailpb.SendActivationEmailResponse, error) {
	if s.sendActivationFn != nil {
		return s.sendActivationFn(ctx, in, opts...)
	}
	return &emailpb.SendActivationEmailResponse{}, nil
}
func (s *stubEmailClient) SendPasswordResetEmail(ctx context.Context, in *emailpb.SendPasswordResetEmailRequest, opts ...grpc.CallOption) (*emailpb.SendPasswordResetEmailResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmailClient) SendPasswordConfirmationEmail(ctx context.Context, in *emailpb.SendActivationEmailRequest, opts ...grpc.CallOption) (*emailpb.SendActivationEmailResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmailClient) SendAccountCreatedEmail(ctx context.Context, in *emailpb.SendAccountCreatedEmailRequest, opts ...grpc.CallOption) (*emailpb.SendAccountCreatedEmailResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmailClient) SendCardConfirmationEmail(ctx context.Context, in *emailpb.SendCardConfirmationEmailRequest, opts ...grpc.CallOption) (*emailpb.SendCardConfirmationEmailResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmailClient) SendLoanLatePaymentEmail(ctx context.Context, in *emailpb.SendLoanLatePaymentEmailRequest, opts ...grpc.CallOption) (*emailpb.SendLoanLatePaymentEmailResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmailClient) SendAccountLockedEmail(ctx context.Context, in *emailpb.SendAccountLockedEmailRequest, opts ...grpc.CallOption) (*emailpb.SendAccountLockedEmailResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmailClient) SendPaymentNotificationEmail(ctx context.Context, in *emailpb.SendPaymentNotificationEmailRequest, opts ...grpc.CallOption) (*emailpb.SendPaymentNotificationEmailResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmailClient) SendCardBlockedEmail(ctx context.Context, in *emailpb.SendCardBlockedEmailRequest, opts ...grpc.CallOption) (*emailpb.SendCardBlockedEmailResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmailClient) SendLoanApprovedEmail(ctx context.Context, in *emailpb.SendLoanApprovedEmailRequest, opts ...grpc.CallOption) (*emailpb.SendLoanApprovedEmailResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubEmailClient) SendLimitChangeEmail(ctx context.Context, in *emailpb.SendLimitChangeEmailRequest, opts ...grpc.CallOption) (*emailpb.SendLimitChangeEmailResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// ---- helpers ----

func sampleEmployee() *pb.Employee {
	return &pb.Employee{
		Id:          1,
		FirstName:   "Marko",
		LastName:    "Marković",
		DateOfBirth: "1990-01-15",
		Gender:      "M",
		Email:       "marko@exbanka.rs",
		PhoneNumber: "+381641234567",
		Address:     "Bulevar Kralja Aleksandra 73",
		Username:    "mmarkovic",
		Position:    "Teller",
		Department:  "Retail",
		Active:      true,
		Permissions: []string{"LOANS"},
		Jmbg:        "0101990710006",
	}
}

func serveHandler(handler gin.HandlerFunc, method, path, urlPath string, body string) *httptest.ResponseRecorder {
	router := gin.New()
	router.Handle(method, path, handler)
	w := httptest.NewRecorder()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, urlPath, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, urlPath, nil)
	}
	router.ServeHTTP(w, req)
	return w
}

// ---- GetEmployeeById ----

func TestGetEmployeeById_InvalidId(t *testing.T) {
	client := &stubEmpClient{}
	w := serveHandler(GetEmployeeById(client), "GET", "/employees/:id", "/employees/abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetEmployeeById_NotFound(t *testing.T) {
	client := &stubEmpClient{
		getByIdFn: func(ctx context.Context, in *pb.GetEmployeeByIdRequest, opts ...grpc.CallOption) (*pb.GetEmployeeByIdResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(GetEmployeeById(client), "GET", "/employees/:id", "/employees/99", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetEmployeeById_InternalError(t *testing.T) {
	client := &stubEmpClient{
		getByIdFn: func(ctx context.Context, in *pb.GetEmployeeByIdRequest, opts ...grpc.CallOption) (*pb.GetEmployeeByIdResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(GetEmployeeById(client), "GET", "/employees/:id", "/employees/1", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetEmployeeById_Happy(t *testing.T) {
	emp := sampleEmployee()
	client := &stubEmpClient{
		getByIdFn: func(ctx context.Context, in *pb.GetEmployeeByIdRequest, opts ...grpc.CallOption) (*pb.GetEmployeeByIdResponse, error) {
			return &pb.GetEmployeeByIdResponse{Employee: emp}, nil
		},
	}
	w := serveHandler(GetEmployeeById(client), "GET", "/employees/:id", "/employees/1", "")
	require.Equal(t, http.StatusOK, w.Code)
	var resp employeeResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Id)
	assert.Equal(t, "Marko", resp.FirstName)
}

// ---- GetEmployees ----

func TestGetEmployees_Error(t *testing.T) {
	client := &stubEmpClient{
		getAllFn: func(ctx context.Context, in *pb.GetAllEmployeesRequest, opts ...grpc.CallOption) (*pb.GetAllEmployeesResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(GetEmployees(client), "GET", "/employees", "/employees", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetEmployees_Happy(t *testing.T) {
	client := &stubEmpClient{
		getAllFn: func(ctx context.Context, in *pb.GetAllEmployeesRequest, opts ...grpc.CallOption) (*pb.GetAllEmployeesResponse, error) {
			return &pb.GetAllEmployeesResponse{Employees: []*pb.Employee{sampleEmployee()}, TotalCount: 1}, nil
		},
	}
	w := serveHandler(GetEmployees(client), "GET", "/employees", "/employees", "")
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total_count"])
}

// ---- SearchEmployees ----

func TestSearchEmployees_Error(t *testing.T) {
	client := &stubEmpClient{
		searchFn: func(ctx context.Context, in *pb.SearchEmployeesRequest, opts ...grpc.CallOption) (*pb.SearchEmployeesResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(SearchEmployees(client), "GET", "/employees/search", "/employees/search?email=x", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSearchEmployees_Happy(t *testing.T) {
	client := &stubEmpClient{
		searchFn: func(ctx context.Context, in *pb.SearchEmployeesRequest, opts ...grpc.CallOption) (*pb.SearchEmployeesResponse, error) {
			return &pb.SearchEmployeesResponse{Employees: []*pb.Employee{sampleEmployee()}, TotalCount: 1}, nil
		},
	}
	w := serveHandler(SearchEmployees(client), "GET", "/employees/search", "/employees/search", "")
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total_count"])
}

// ---- UpdateEmployee ----

var validUpdateBody = `{
	"first_name":"Marko","last_name":"Marković",
	"email":"marko@exbanka.rs","username":"mmarkovic"
}`

func TestUpdateEmployee_InvalidId(t *testing.T) {
	client := &stubEmpClient{}
	w := serveHandler(UpdateEmployee(client, &stubFundClient{}), "PUT", "/employees/:id", "/employees/abc", validUpdateBody)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateEmployee_BadJSON(t *testing.T) {
	client := &stubEmpClient{}
	w := serveHandler(UpdateEmployee(client, &stubFundClient{}), "PUT", "/employees/:id", "/employees/1", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateEmployee_MissingRequiredField(t *testing.T) {
	client := &stubEmpClient{}
	// missing last_name
	w := serveHandler(UpdateEmployee(client, &stubFundClient{}), "PUT", "/employees/:id", "/employees/1", `{"first_name":"Marko","email":"m@e.rs","username":"u"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateEmployee_InvalidEmail(t *testing.T) {
	client := &stubEmpClient{}
	body := `{"first_name":"Marko","last_name":"M","email":"not-an-email","username":"u"}`
	w := serveHandler(UpdateEmployee(client, &stubFundClient{}), "PUT", "/employees/:id", "/employees/1", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateEmployee_NotFound(t *testing.T) {
	client := &stubEmpClient{
		updateFn: func(ctx context.Context, in *pb.UpdateEmployeeRequest, opts ...grpc.CallOption) (*pb.UpdateEmployeeResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	w := serveHandler(UpdateEmployee(client, &stubFundClient{}), "PUT", "/employees/:id", "/employees/1", validUpdateBody)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateEmployee_Conflict(t *testing.T) {
	client := &stubEmpClient{
		updateFn: func(ctx context.Context, in *pb.UpdateEmployeeRequest, opts ...grpc.CallOption) (*pb.UpdateEmployeeResponse, error) {
			return nil, status.Error(codes.AlreadyExists, "email already in use")
		},
	}
	w := serveHandler(UpdateEmployee(client, &stubFundClient{}), "PUT", "/employees/:id", "/employees/1", validUpdateBody)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestUpdateEmployee_FailedPrecondition(t *testing.T) {
	client := &stubEmpClient{
		updateFn: func(ctx context.Context, in *pb.UpdateEmployeeRequest, opts ...grpc.CallOption) (*pb.UpdateEmployeeResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "invalid state")
		},
	}
	w := serveHandler(UpdateEmployee(client, &stubFundClient{}), "PUT", "/employees/:id", "/employees/1", validUpdateBody)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateEmployee_InternalError(t *testing.T) {
	client := &stubEmpClient{
		updateFn: func(ctx context.Context, in *pb.UpdateEmployeeRequest, opts ...grpc.CallOption) (*pb.UpdateEmployeeResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	w := serveHandler(UpdateEmployee(client, &stubFundClient{}), "PUT", "/employees/:id", "/employees/1", validUpdateBody)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateEmployee_Happy(t *testing.T) {
	emp := sampleEmployee()
	client := &stubEmpClient{
		updateFn: func(ctx context.Context, in *pb.UpdateEmployeeRequest, opts ...grpc.CallOption) (*pb.UpdateEmployeeResponse, error) {
			return &pb.UpdateEmployeeResponse{Employee: emp}, nil
		},
	}
	w := serveHandler(UpdateEmployee(client, &stubFundClient{}), "PUT", "/employees/:id", "/employees/1", validUpdateBody)
	require.Equal(t, http.StatusOK, w.Code)
	var resp employeeResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Id)
}

// ---- CreateEmployee ----

var validCreateBody = `{
	"first_name":"Marko","last_name":"Marković",
	"date_of_birth":"1990-01-15","gender":"M",
	"email":"marko@exbanka.rs","phone_number":"+381641234567",
	"address":"Bulevar 73","username":"mmarkovic",
	"position":"Teller","department":"Retail","jmbg":"0101990710006"
}`

func TestCreateEmployee_BadJSON(t *testing.T) {
	empClient := &stubEmpClient{}
	authClient := &stubAuthClient{}
	emailClient := &stubEmailClient{}
	w := serveHandler(CreateEmployee(empClient, authClient, emailClient), "POST", "/employees", "/employees", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEmployee_MissingField(t *testing.T) {
	empClient := &stubEmpClient{}
	authClient := &stubAuthClient{}
	emailClient := &stubEmailClient{}
	// missing jmbg
	body := `{"first_name":"M","last_name":"M","date_of_birth":"1990-01-15","gender":"M","email":"m@e.rs","phone_number":"123","address":"a","username":"u","position":"p","department":"d"}`
	w := serveHandler(CreateEmployee(empClient, authClient, emailClient), "POST", "/employees", "/employees", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEmployee_InvalidEmail(t *testing.T) {
	empClient := &stubEmpClient{}
	authClient := &stubAuthClient{}
	emailClient := &stubEmailClient{}
	body := `{"first_name":"M","last_name":"M","date_of_birth":"1990-01-15","gender":"M","email":"bad-email","phone_number":"123","address":"a","username":"u","position":"p","department":"d","jmbg":"1234567890123"}`
	w := serveHandler(CreateEmployee(empClient, authClient, emailClient), "POST", "/employees", "/employees", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEmployee_Conflict(t *testing.T) {
	empClient := &stubEmpClient{
		createFn: func(ctx context.Context, in *pb.CreateEmployeeRequest, opts ...grpc.CallOption) (*pb.CreateEmployeeResponse, error) {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		},
	}
	authClient := &stubAuthClient{}
	emailClient := &stubEmailClient{}
	w := serveHandler(CreateEmployee(empClient, authClient, emailClient), "POST", "/employees", "/employees", validCreateBody)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateEmployee_EmpServiceError(t *testing.T) {
	empClient := &stubEmpClient{
		createFn: func(ctx context.Context, in *pb.CreateEmployeeRequest, opts ...grpc.CallOption) (*pb.CreateEmployeeResponse, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	authClient := &stubAuthClient{}
	emailClient := &stubEmailClient{}
	w := serveHandler(CreateEmployee(empClient, authClient, emailClient), "POST", "/employees", "/employees", validCreateBody)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateEmployee_AuthTokenError(t *testing.T) {
	empClient := &stubEmpClient{
		createFn: func(ctx context.Context, in *pb.CreateEmployeeRequest, opts ...grpc.CallOption) (*pb.CreateEmployeeResponse, error) {
			return &pb.CreateEmployeeResponse{Employee: sampleEmployee()}, nil
		},
	}
	authClient := &stubAuthClient{
		createActivationTokenFn: func(ctx context.Context, in *authpb.CreateActivationTokenRequest, opts ...grpc.CallOption) (*authpb.CreateActivationTokenResponse, error) {
			return nil, fmt.Errorf("auth service down")
		},
	}
	emailClient := &stubEmailClient{}
	w := serveHandler(CreateEmployee(empClient, authClient, emailClient), "POST", "/employees", "/employees", validCreateBody)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateEmployee_Happy(t *testing.T) {
	empClient := &stubEmpClient{
		createFn: func(ctx context.Context, in *pb.CreateEmployeeRequest, opts ...grpc.CallOption) (*pb.CreateEmployeeResponse, error) {
			return &pb.CreateEmployeeResponse{Employee: sampleEmployee()}, nil
		},
	}
	authClient := &stubAuthClient{
		createActivationTokenFn: func(ctx context.Context, in *authpb.CreateActivationTokenRequest, opts ...grpc.CallOption) (*authpb.CreateActivationTokenResponse, error) {
			return &authpb.CreateActivationTokenResponse{Token: "tok123"}, nil
		},
	}
	emailClient := &stubEmailClient{}
	w := serveHandler(CreateEmployee(empClient, authClient, emailClient), "POST", "/employees", "/employees", validCreateBody)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp employeeResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Id)
}

// ---- toEmployeeResponse ----

func TestToEmployeeResponse_NilPermissions(t *testing.T) {
	emp := &pb.Employee{Id: 1, FirstName: "A", Permissions: nil}
	r := toEmployeeResponse(emp)
	assert.NotNil(t, r.Permissions)
	assert.Len(t, r.Permissions, 0)
}

// ---- UpdateEmployee — supervisor removal / fund transfer ----

func makeAdminToken() string {
	claims := jwt.MapClaims{
		"user_id": float64(99),
		"role":    "EMPLOYEE",
		"dozvole": []interface{}{"ADMIN", "SUPERVISOR"},
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(""))
	return "Bearer " + tok
}

var supervisorRemovalBody = `{
	"first_name":"Ana","last_name":"Anić",
	"email":"ana@exbanka.rs","username":"aanc",
	"permissions":["READ"]
}`

func TestUpdateEmployee_SupervisorRemovedTransfersFunds(t *testing.T) {
	transferCalled := false
	fundStub := &stubFundClient{
		transferFundsByManagerFn: func(_ context.Context, req *pb_fund.TransferFundsByManagerRequest, _ ...grpc.CallOption) (*pb_fund.TransferFundsByManagerResponse, error) {
			transferCalled = true
			assert.Equal(t, int64(1), req.OldManagerId)
			assert.Equal(t, int64(99), req.NewManagerId) // admin user_id from token
			return &pb_fund.TransferFundsByManagerResponse{FundsTransferred: 2}, nil
		},
	}
	empStub := &stubEmpClient{
		updateFn: func(_ context.Context, _ *pb.UpdateEmployeeRequest, _ ...grpc.CallOption) (*pb.UpdateEmployeeResponse, error) {
			return &pb.UpdateEmployeeResponse{Employee: sampleEmployee()}, nil
		},
	}
	w := serveHandlerFull(UpdateEmployee(empStub, fundStub), "PUT", "/employees/:id", "/employees/1", supervisorRemovalBody, makeAdminToken())
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, transferCalled, "fund transfer should have been called")
}

func TestUpdateEmployee_SupervisorRemovedFundTransferFails(t *testing.T) {
	empUpdateCalled := false
	fundStub := &stubFundClient{
		transferFundsByManagerFn: func(_ context.Context, _ *pb_fund.TransferFundsByManagerRequest, _ ...grpc.CallOption) (*pb_fund.TransferFundsByManagerResponse, error) {
			return nil, fmt.Errorf("fund db error")
		},
	}
	empStub := &stubEmpClient{
		updateFn: func(_ context.Context, _ *pb.UpdateEmployeeRequest, _ ...grpc.CallOption) (*pb.UpdateEmployeeResponse, error) {
			empUpdateCalled = true
			return &pb.UpdateEmployeeResponse{Employee: sampleEmployee()}, nil
		},
	}
	w := serveHandlerFull(UpdateEmployee(empStub, fundStub), "PUT", "/employees/:id", "/employees/1", supervisorRemovalBody, makeAdminToken())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.False(t, empUpdateCalled, "employee update should not have been called after fund transfer failure")
}

func TestUpdateEmployee_SupervisorKeptNoTransfer(t *testing.T) {
	transferCalled := false
	fundStub := &stubFundClient{
		transferFundsByManagerFn: func(_ context.Context, _ *pb_fund.TransferFundsByManagerRequest, _ ...grpc.CallOption) (*pb_fund.TransferFundsByManagerResponse, error) {
			transferCalled = true
			return &pb_fund.TransferFundsByManagerResponse{}, nil
		},
	}
	empStub := &stubEmpClient{
		updateFn: func(_ context.Context, _ *pb.UpdateEmployeeRequest, _ ...grpc.CallOption) (*pb.UpdateEmployeeResponse, error) {
			return &pb.UpdateEmployeeResponse{Employee: sampleEmployee()}, nil
		},
	}
	body := `{"first_name":"Ana","last_name":"Anić","email":"ana@exbanka.rs","username":"aanc","permissions":["SUPERVISOR"]}`
	w := serveHandlerFull(UpdateEmployee(empStub, fundStub), "PUT", "/employees/:id", "/employees/1", body, makeAdminToken())
	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, transferCalled, "fund transfer should NOT be called when SUPERVISOR is kept")
}
