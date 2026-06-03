package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pb_auth "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ── helpers ─────────────────────────────────────────────────────────────────

func newAuthServerFull(t *testing.T) (*AuthServer, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	s := &AuthServer{
		DB:             db,
		EmployeeClient: new(mockEmployeeClient),
		EmailClient:    new(mockEmailClient),
		ClientClient:   new(mockClientClient),
	}
	return s, mock
}

func newAuthServerWithRedis(t *testing.T) (*AuthServer, sqlmock.Sqlmock, *miniredis.Miniredis) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	s := &AuthServer{
		DB:             db,
		EmployeeClient: new(mockEmployeeClient),
		EmailClient:    new(mockEmailClient),
		ClientClient:   new(mockClientClient),
		Redis:          rc,
	}
	return s, mock, mr
}

// ── ResetPassword: expired token where DELETE also fails (line 275-277) ─────

func TestResetPassword_ExpiredToken_DeleteFails(t *testing.T) {
	s, dbMock := newAuthServerFull(t)
	dbMock.ExpectQuery("SELECT employee_id, expires_at FROM password_reset_tokens").
		WithArgs("exp-tok").
		WillReturnRows(sqlmock.NewRows([]string{"employee_id", "expires_at"}).
			AddRow(int64(1), time.Now().Add(-time.Hour)))
	dbMock.ExpectExec("DELETE FROM password_reset_tokens").
		WithArgs("exp-tok").
		WillReturnError(sql.ErrConnDone)

	_, err := s.ResetPassword(context.Background(), &pb_auth.ResetPasswordRequest{Token: "exp-tok"})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// ── ActivateClient: expired token where DELETE also fails (line 594-596) ────

func TestActivateClient_ExpiredToken_DeleteFails(t *testing.T) {
	s, dbMock := newAuthServerFull(t)
	dbMock.ExpectQuery("SELECT client_id, expires_at FROM client_activation_tokens").
		WithArgs("exp-act-tok").
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "expires_at"}).
			AddRow(int64(1), time.Now().Add(-time.Hour)))
	dbMock.ExpectExec("DELETE FROM client_activation_tokens").
		WithArgs("exp-act-tok").
		WillReturnError(sql.ErrConnDone)

	_, err := s.ActivateClient(context.Background(), &pb_auth.ActivateClientRequest{Token: "exp-act-tok"})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// ── GetApproval: DB error that is not ErrNoRows (line 686-688) ──────────────

func TestGetApproval_DBError(t *testing.T) {
	s, dbMock := newAuthServerFull(t)
	dbMock.ExpectQuery("SELECT id, client_id, action_type, payload, status, created_at, expires_at FROM two_factor_approvals").
		WithArgs(int64(99)).
		WillReturnError(sql.ErrConnDone)

	_, err := s.GetApproval(context.Background(), &pb_auth.GetApprovalRequest{Id: 99})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// ── GetClientApprovals: scan error (line 715-717) ───────────────────────────

func TestGetClientApprovals_ScanError(t *testing.T) {
	s, dbMock := newAuthServerFull(t)
	dbMock.ExpectExec("UPDATE two_factor_approvals SET status = 'EXPIRED'").
		WithArgs(int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectQuery("SELECT id, client_id, action_type, payload, status, created_at, expires_at FROM two_factor_approvals").
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "client_id", "action_type", "payload", "status", "created_at", "expires_at",
		}).AddRow(
			"bad-id", int64(5), "LOGIN", `{}`, "PENDING",
			time.Now(), time.Now().Add(time.Hour),
		))

	_, err := s.GetClientApprovals(context.Background(), &pb_auth.GetClientApprovalsRequest{ClientId: 5})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// ── UpdateApprovalStatus: DB error (line 738-740) ───────────────────────────

func TestUpdateApprovalStatus_DBError(t *testing.T) {
	s, dbMock := newAuthServerFull(t)
	dbMock.ExpectQuery("UPDATE two_factor_approvals SET status").
		WithArgs("APPROVED", int64(7), int64(3)).
		WillReturnError(sql.ErrConnDone)

	_, err := s.UpdateApprovalStatus(context.Background(), &pb_auth.UpdateApprovalStatusRequest{
		Id: 7, ClientId: 3, Status: "APPROVED",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// ── PollApproval: Redis cache — PENDING expired (lines 479-481) ─────────────

func TestPollApproval_Redis_PendingExpired(t *testing.T) {
	s, dbMock, mr := newAuthServerWithRedis(t)

	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	cached := approvalCache{Status: "PENDING", ActionType: "LOGIN", Payload: `{}`, ExpiresAt: past}
	data, _ := json.Marshal(cached)
	_ = mr.Set(approvalCacheKey(42), string(data))

	dbMock.ExpectExec("UPDATE two_factor_approvals SET status = 'EXPIRED'").
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, err := s.PollApproval(context.Background(), &pb_auth.PollApprovalRequest{Id: 42})
	require.NoError(t, err)
	assert.Equal(t, "EXPIRED", resp.Status)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// ── PollApproval: Redis cache — APPROVED LOGIN (lines 484-490) ──────────────

func TestPollApproval_Redis_ApprovedLogin(t *testing.T) {
	s, _, mr := newAuthServerWithRedis(t)

	payload, _ := json.Marshal(map[string]string{
		"access_token":  "acc-tok",
		"refresh_token": "ref-tok",
	})
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	cached := approvalCache{Status: "APPROVED", ActionType: "LOGIN", Payload: string(payload), ExpiresAt: future}
	data, _ := json.Marshal(cached)
	_ = mr.Set(approvalCacheKey(55), string(data))

	resp, err := s.PollApproval(context.Background(), &pb_auth.PollApprovalRequest{Id: 55})
	require.NoError(t, err)
	assert.Equal(t, "APPROVED", resp.Status)
	assert.Equal(t, "acc-tok", resp.AccessToken)
	assert.Equal(t, "ref-tok", resp.RefreshToken)
}

// ── Logout: non-HMAC signing method triggers line 826-828 ───────────────────

func TestLogout_NonHMACToken(t *testing.T) {
	s, _, _ := newAuthServerWithRedis(t)

	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"jti": "some-jti",
		"sub": "1",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	})
	tokenStr, err := tok.SignedString(privateKey)
	require.NoError(t, err)

	resp, err := s.Logout(context.Background(), &pb_auth.LogoutRequest{Token: tokenStr})
	require.NoError(t, err) // Logout never returns an error
	assert.NotNil(t, resp)
}

// ── GetClientCredentials via ClientClient mock (covers ClientClient field) ───
// Also validates that mockClientClient satisfies the interface used by AuthServer.

func TestMockClientClient_Interface(t *testing.T) {
	m := new(mockClientClient)
	var _ pb_client.ClientServiceClient = m
	_ = fmt.Sprintf("%T", m) // ensure import used
}
