package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb_auth "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newAuthRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return mr, rdb
}

// makeSignedToken produces a signed JWT for use in Logout tests.
func makeSignedToken(t *testing.T, jtiVal string, expOffset time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"jti":     jtiVal,
		"user_id": float64(1),
		"type":    "access",
		"exp":     time.Now().Add(expOffset).Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	require.NoError(t, err)
	return tok
}

// makeSignedTokenNoJTI produces a signed JWT without a jti claim.
func makeSignedTokenNoJTI(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id": float64(1),
		"type":    "access",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	require.NoError(t, err)
	return tok
}

// ── generateToken / generateClientToken — jti claim ──────────────────────────

func TestGenerateToken_ContainsJTI(t *testing.T) {
	tok, err := generateToken(1, "user@test.com", "access", []string{"AGENT"},
		"John", "Doe", "user@test.com", time.Hour)
	require.NoError(t, err)

	parsed, err := jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	require.NoError(t, err)
	claims := parsed.Claims.(jwt.MapClaims)
	jtiVal, ok := claims["jti"].(string)
	assert.True(t, ok, "jti claim must be a string")
	assert.NotEmpty(t, jtiVal, "jti must not be empty")
}

func TestGenerateToken_EachCallProducesUniqueJTI(t *testing.T) {
	tok1, _ := generateToken(1, "u@t.com", "access", nil, "A", "B", "u@t.com", time.Hour)
	tok2, _ := generateToken(1, "u@t.com", "access", nil, "A", "B", "u@t.com", time.Hour)

	parse := func(s string) string {
		p, _ := jwt.Parse(s, func(t *jwt.Token) (interface{}, error) { return []byte(jwtSecret), nil })
		return p.Claims.(jwt.MapClaims)["jti"].(string)
	}
	assert.NotEqual(t, parse(tok1), parse(tok2), "each token must have a unique jti")
}

func TestGenerateClientToken_ContainsJTI(t *testing.T) {
	tok, err := generateClientToken(42, "client@test.com", "access", "Jane", "Doe", time.Hour)
	require.NoError(t, err)

	parsed, err := jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	require.NoError(t, err)
	claims := parsed.Claims.(jwt.MapClaims)
	jtiVal, ok := claims["jti"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, jtiVal)
}

// ── Logout — token blacklisting ───────────────────────────────────────────────

func TestLogout_ValidToken_StoresBlacklistKey(t *testing.T) {
	mr, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}

	jtiVal := "test-jti-logout-001"
	tok := makeSignedToken(t, jtiVal, time.Hour)

	_, err := s.Logout(context.Background(), &pb_auth.LogoutRequest{Token: tok})
	require.NoError(t, err)

	// Key must exist in Redis.
	val, redisErr := mr.Get("blacklist:" + jtiVal)
	require.NoError(t, redisErr, "blacklist key must exist after logout")
	assert.Equal(t, "1", val)
}

func TestLogout_ValidToken_TTLIsRemainingLifetime(t *testing.T) {
	mr, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}

	expOffset := 30 * time.Minute
	jtiVal := "test-jti-logout-ttl-002"
	tok := makeSignedToken(t, jtiVal, expOffset)

	_, err := s.Logout(context.Background(), &pb_auth.LogoutRequest{Token: tok})
	require.NoError(t, err)

	ttl := mr.TTL("blacklist:" + jtiVal)
	// TTL must be close to the remaining token lifetime (within 5s tolerance).
	assert.Greater(t, ttl, expOffset-5*time.Second, "TTL must be close to token expiry")
	assert.LessOrEqual(t, ttl, expOffset)
}

func TestLogout_AlreadyExpiredToken_DoesNotWriteToRedis(t *testing.T) {
	mr, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}

	// Token already expired.
	jtiVal := "test-jti-expired-003"
	tok := makeSignedToken(t, jtiVal, -time.Hour)

	_, err := s.Logout(context.Background(), &pb_auth.LogoutRequest{Token: tok})
	// Implementation silently returns OK for invalid/expired tokens.
	require.NoError(t, err)

	// No key should be written for an expired token.
	keys := mr.Keys()
	assert.Empty(t, keys, "expired token must not create a blacklist entry")
}

func TestLogout_TokenWithoutJTI_DoesNotWriteToRedis(t *testing.T) {
	mr, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}

	tok := makeSignedTokenNoJTI(t)
	_, err := s.Logout(context.Background(), &pb_auth.LogoutRequest{Token: tok})
	require.NoError(t, err)

	assert.Empty(t, mr.Keys())
}

func TestLogout_InvalidToken_ReturnsOKWithoutWriting(t *testing.T) {
	mr, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}

	_, err := s.Logout(context.Background(), &pb_auth.LogoutRequest{Token: "not.a.real.jwt"})
	require.NoError(t, err)
	assert.Empty(t, mr.Keys())
}

func TestLogout_EmptyToken_ReturnsOKWithoutWriting(t *testing.T) {
	mr, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}

	_, err := s.Logout(context.Background(), &pb_auth.LogoutRequest{Token: ""})
	require.NoError(t, err)
	assert.Empty(t, mr.Keys())
}

func TestLogout_NilRedis_ReturnsOK(t *testing.T) {
	s := &AuthServer{Redis: nil}
	tok := makeSignedToken(t, "jti-nil-redis-005", time.Hour)
	_, err := s.Logout(context.Background(), &pb_auth.LogoutRequest{Token: tok})
	require.NoError(t, err)
}

// ── approvalCache — store/load round-trip ─────────────────────────────────────

func TestStoreApprovalCache_NilRedis_NoOp(t *testing.T) {
	s := &AuthServer{Redis: nil}
	s.storeApprovalCache(context.Background(), 1, &approvalCache{
		ActionType: "LOGIN",
		Payload:    "{}",
		Status:     "PENDING",
		ExpiresAt:  time.Now().Add(5 * time.Minute).Format(time.RFC3339),
	})
}

func TestStoreLoadApprovalCache_RoundTrip(t *testing.T) {
	_, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}

	original := &approvalCache{
		ActionType: "LOGIN",
		Payload:    `{"access_token":"tok123"}`,
		Status:     "PENDING",
		ExpiresAt:  time.Now().Add(5 * time.Minute).Format(time.RFC3339),
	}
	s.storeApprovalCache(context.Background(), 77, original)

	result := s.loadApprovalCache(context.Background(), 77)
	require.NotNil(t, result)
	assert.Equal(t, "LOGIN", result.ActionType)
	assert.Equal(t, `{"access_token":"tok123"}`, result.Payload)
	assert.Equal(t, "PENDING", result.Status)
}

func TestStoreApprovalCache_TTLMatchesExpiresAt(t *testing.T) {
	mr, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}

	ttlTarget := 4 * time.Minute
	a := &approvalCache{
		ActionType: "LOGIN",
		Payload:    "{}",
		Status:     "PENDING",
		ExpiresAt:  time.Now().Add(ttlTarget).Format(time.RFC3339),
	}
	s.storeApprovalCache(context.Background(), 88, a)

	ttl := mr.TTL(approvalCacheKey(88))
	assert.Greater(t, ttl, ttlTarget-5*time.Second)
	assert.LessOrEqual(t, ttl, ttlTarget)
}

func TestLoadApprovalCache_KeyMissing_ReturnsNil(t *testing.T) {
	_, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}
	assert.Nil(t, s.loadApprovalCache(context.Background(), 9999))
}

func TestLoadApprovalCache_NilRedis_ReturnsNil(t *testing.T) {
	s := &AuthServer{Redis: nil}
	assert.Nil(t, s.loadApprovalCache(context.Background(), 1))
}

func TestLoadApprovalCache_InvalidJSON_ReturnsNil(t *testing.T) {
	mr, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}
	_ = mr.Set(approvalCacheKey(50), "not-json")
	assert.Nil(t, s.loadApprovalCache(context.Background(), 50))
}

func TestStoreApprovalCache_AlreadyExpired_DoesNotStore(t *testing.T) {
	mr, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}

	// ExpiresAt in the past → storeApprovalCache must skip writing.
	a := &approvalCache{
		ActionType: "LOGIN",
		Payload:    "{}",
		Status:     "PENDING",
		ExpiresAt:  time.Now().Add(-time.Minute).Format(time.RFC3339),
	}
	s.storeApprovalCache(context.Background(), 99, a)
	assert.Empty(t, mr.Keys(), "expired approval must not be stored")
}

func TestStoreApprovalCache_UpdatedStatus_OverwritesPreviousEntry(t *testing.T) {
	_, rdb := newAuthRedis(t)
	s := &AuthServer{Redis: rdb}

	exp := time.Now().Add(5 * time.Minute).Format(time.RFC3339)
	s.storeApprovalCache(context.Background(), 11, &approvalCache{
		ActionType: "LOGIN", Payload: "{}", Status: "PENDING", ExpiresAt: exp,
	})
	s.storeApprovalCache(context.Background(), 11, &approvalCache{
		ActionType: "LOGIN", Payload: "{}", Status: "APPROVED", ExpiresAt: exp,
	})

	result := s.loadApprovalCache(context.Background(), 11)
	require.NotNil(t, result)
	assert.Equal(t, "APPROVED", result.Status)
}
