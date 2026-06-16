package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func makeToken(tokenType string, roles []string, expOffset time.Duration) string {
	claims := jwt.MapClaims{
		"user_id":  float64(1),
		"username": "user@example.com",
		"type":     tokenType,
		"dozvole":  roles,
		"exp":      time.Now().Add(expOffset).Unix(),
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	return token
}

// runMiddleware builds a test router with RequireRole(role) and executes one GET /test.
// Pass authHeader as the full value of the Authorization header (empty = omit the header).
func runMiddleware(authHeader string, role string) int {
	router := gin.New()
	router.GET("/test", RequireRole(role), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	router.ServeHTTP(w, req)
	return w.Code
}

func TestRequireRole_NoHeader(t *testing.T) {
	assert.Equal(t, http.StatusUnauthorized, runMiddleware("", "ADMIN"))
}

func TestRequireRole_NonBearerPrefix(t *testing.T) {
	token := makeToken("access", []string{"ADMIN"}, time.Hour)
	assert.Equal(t, http.StatusUnauthorized, runMiddleware("Token "+token, "ADMIN"))
}

func TestRequireRole_MalformedToken(t *testing.T) {
	assert.Equal(t, http.StatusUnauthorized, runMiddleware("Bearer not.a.real.jwt", "ADMIN"))
}

func TestRequireRole_ExpiredToken(t *testing.T) {
	token := makeToken("access", []string{"ADMIN"}, -time.Hour)
	assert.Equal(t, http.StatusUnauthorized, runMiddleware("Bearer "+token, "ADMIN"))
}

func TestRequireRole_WrongTokenType(t *testing.T) {
	// A refresh token must be rejected even if the role matches
	token := makeToken("refresh", []string{"ADMIN"}, time.Hour)
	assert.Equal(t, http.StatusUnauthorized, runMiddleware("Bearer "+token, "ADMIN"))
}

func TestRequireRole_InsufficientRole(t *testing.T) {
	token := makeToken("access", []string{"OPERATOR"}, time.Hour)
	assert.Equal(t, http.StatusForbidden, runMiddleware("Bearer "+token, "ADMIN"))
}

func TestRequireRole_CorrectRole(t *testing.T) {
	token := makeToken("access", []string{"OPERATOR"}, time.Hour)
	assert.Equal(t, http.StatusOK, runMiddleware("Bearer "+token, "OPERATOR"))
}

func TestRequireRole_AdminBypassesRoleCheck(t *testing.T) {
	// ADMIN in token should allow access to any required role
	token := makeToken("access", []string{"ADMIN"}, time.Hour)
	assert.Equal(t, http.StatusOK, runMiddleware("Bearer "+token, "OPERATOR"))
}

func TestRequireRole_RoleCheckIsCaseInsensitive(t *testing.T) {
	// Token has lowercase role name; middleware should upper-case before comparing
	claims := jwt.MapClaims{
		"user_id":  float64(1),
		"username": "user@example.com",
		"type":     "access",
		"dozvole":  []string{"operator"},
		"exp":      time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	assert.Equal(t, http.StatusOK, runMiddleware("Bearer "+tokenStr, "OPERATOR"))
}

// ---- GetUserIDFromToken tests ----

func runGetUserID(authHeader string) (int64, error) {
	gin.SetMode(gin.TestMode)
	var (
		id  int64
		err error
	)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		id, err = GetUserIDFromToken(c)
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	router.ServeHTTP(w, req)
	return id, err
}

func TestGetUserIDFromToken_MissingHeader(t *testing.T) {
	id, err := runGetUserID("")
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
}

func TestGetUserIDFromToken_InvalidToken(t *testing.T) {
	id, err := runGetUserID("Bearer not.a.token")
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
}

func TestGetUserIDFromToken_HappyPath(t *testing.T) {
	token := makeToken("access", []string{"ADMIN"}, time.Hour)
	id, err := runGetUserID("Bearer " + token)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), id)
}

func TestRequireRole_InvalidSigningMethod(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": float64(1), "username": "user@example.com",
		"type": "access", "dozvole": []string{"ADMIN"},
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.Equal(t, http.StatusUnauthorized, runMiddleware("Bearer "+tokenStr, "ADMIN"))
}

func TestGetUserIDFromToken_InvalidSigningMethod(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": float64(1), "username": "user@example.com",
		"type": "access", "exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	id, err := runGetUserID("Bearer " + tokenStr)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
}

func TestGetUserIDFromToken_MissingUserIdClaim(t *testing.T) {
	claims := jwt.MapClaims{
		"username": "user@example.com",
		"type":     "access",
		"exp":      time.Now().Add(time.Hour).Unix(),
		// user_id intentionally omitted
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	id, err := runGetUserID("Bearer " + tokenStr)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
}

func TestGetUserIDFromToken_UserIdWrongType(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": "not-a-number",
		"type":    "access",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	id, err := runGetUserID("Bearer " + tokenStr)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
}

// ---- GetCallerRoleFromToken tests ----

func runGetCallerRole(authHeader string) string {
	var role string
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		role = GetCallerRoleFromToken(c)
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	router.ServeHTTP(w, req)
	return role
}

func TestGetCallerRoleFromToken_NoHeader(t *testing.T) {
	assert.Equal(t, "", runGetCallerRole(""))
}

func TestGetCallerRoleFromToken_NonBearerPrefix(t *testing.T) {
	token := makeToken("access", []string{"ADMIN"}, time.Hour)
	assert.Equal(t, "", runGetCallerRole("Token "+token))
}

func TestGetCallerRoleFromToken_InvalidToken(t *testing.T) {
	assert.Equal(t, "", runGetCallerRole("Bearer not.a.valid.jwt"))
}

func TestGetCallerRoleFromToken_WithRoleClaim(t *testing.T) {
	// Client token: has "role" claim, no "dozvole"
	claims := jwt.MapClaims{
		"user_id": float64(42),
		"role":    "CLIENT",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	assert.Equal(t, "CLIENT", runGetCallerRole("Bearer "+tokenStr))
}

func TestGetCallerRoleFromToken_WithDozvoleClaim(t *testing.T) {
	// Employee token: has "dozvole" claim, no "role"
	token := makeToken("access", []string{"OPERATOR"}, time.Hour)
	assert.Equal(t, "EMPLOYEE", runGetCallerRole("Bearer "+token))
}

func TestGetCallerRoleFromToken_NeitherClaim(t *testing.T) {
	// Token with neither "role" nor "dozvole"
	claims := jwt.MapClaims{
		"user_id": float64(1),
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	assert.Equal(t, "", runGetCallerRole("Bearer "+tokenStr))
}

func TestGetCallerRoleFromToken_InvalidSigningMethod(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": float64(1), "role": "CLIENT",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.Equal(t, "", runGetCallerRole("Bearer "+tokenStr))
}

// ---- JWT blacklist (token revocation) tests ----

func makeTokenWithJTI(jtiVal string, expOffset time.Duration) string {
	claims := jwt.MapClaims{
		"jti":     jtiVal,
		"user_id": float64(1),
		"type":    "access",
		"dozvole": []string{"AGENT"},
		"exp":     time.Now().Add(expOffset).Unix(),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	return tok
}

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

// runWithRedis wires rdb into the package-level redisClient, runs f, then restores.
func runWithRedis(t *testing.T, rdb *redis.Client, f func()) {
	t.Helper()
	prev := redisClient
	InitRedis(rdb)
	t.Cleanup(func() { redisClient = prev })
	f()
}

func TestRequireRole_ValidToken_NotBlacklisted_Passes(t *testing.T) {
	rdb := newTestRedis(t)
	tok := makeTokenWithJTI("jti-clean-001", time.Hour)
	var code int
	runWithRedis(t, rdb, func() { code = runMiddleware("Bearer "+tok, "AGENT") })
	assert.Equal(t, http.StatusOK, code)
}

func TestRequireRole_BlacklistedToken_Returns401(t *testing.T) {
	rdb := newTestRedis(t)
	jtiVal := "jti-revoked-002"
	err := rdb.Set(context.Background(), "blacklist:"+jtiVal, "1", time.Hour).Err()
	require.NoError(t, err)

	tok := makeTokenWithJTI(jtiVal, time.Hour)
	var code int
	runWithRedis(t, rdb, func() { code = runMiddleware("Bearer "+tok, "AGENT") })
	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestRequireRole_BlacklistedToken_ErrorMessageIsTokenRevoked(t *testing.T) {
	rdb := newTestRedis(t)
	jtiVal := "jti-revoked-msg-003"
	_ = rdb.Set(context.Background(), "blacklist:"+jtiVal, "1", time.Hour).Err()

	tok := makeTokenWithJTI(jtiVal, time.Hour)
	router := gin.New()
	runWithRedis(t, rdb, func() {
		router.GET("/test", RequireRole("AGENT"), func(c *gin.Context) { c.Status(http.StatusOK) })
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "token revoked")
}

func TestRequireRole_TokenWithoutJTI_Passes(t *testing.T) {
	rdb := newTestRedis(t)
	claims := jwt.MapClaims{
		"user_id": float64(1),
		"type":    "access",
		"dozvole": []string{"AGENT"},
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	var code int
	runWithRedis(t, rdb, func() { code = runMiddleware("Bearer "+tok, "AGENT") })
	assert.Equal(t, http.StatusOK, code)
}

func TestRequireRole_NilRedis_SkipsBlacklistCheck(t *testing.T) {
	prev := redisClient
	redisClient = nil
	defer func() { redisClient = prev }()

	tok := makeTokenWithJTI("jti-no-redis-004", time.Hour)
	assert.Equal(t, http.StatusOK, runMiddleware("Bearer "+tok, "AGENT"))
}

func TestRequireRole_ExpiredBlacklistEntry_Passes(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	jtiVal := "jti-expired-005"
	_ = rdb.Set(context.Background(), "blacklist:"+jtiVal, "1", time.Second).Err()

	// Fast-forward miniredis so the key has expired.
	mr.FastForward(2 * time.Second)

	tok := makeTokenWithJTI(jtiVal, time.Hour)
	var code int
	runWithRedis(t, rdb, func() { code = runMiddleware("Bearer "+tok, "AGENT") })
	assert.Equal(t, http.StatusOK, code)
}

// callerHasPermission is a test helper that builds a minimal gin.Context with the given
// Authorization header and calls CallerHasPermission.
func callerHasPermission(authHeader, perm string) bool {
	router := gin.New()
	var result bool
	router.GET("/test", func(c *gin.Context) {
		result = CallerHasPermission(c, perm)
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	router.ServeHTTP(w, req)
	return result
}

func TestCallerHasPermission_NoHeader(t *testing.T) {
	assert.False(t, callerHasPermission("", "SOME_PERMISSION"))
}

func TestCallerHasPermission_MalformedToken(t *testing.T) {
	assert.False(t, callerHasPermission("Bearer not.a.jwt", "SOME_PERMISSION"))
}

func TestCallerHasPermission_NoDozvole(t *testing.T) {
	token := makeToken("access", nil, time.Hour)
	assert.False(t, callerHasPermission("Bearer "+token, "SOME_PERMISSION"))
}

func TestCallerHasPermission_PermissionNotInList(t *testing.T) {
	token := makeToken("access", []string{"READ_USERS", "WRITE_USERS"}, time.Hour)
	assert.False(t, callerHasPermission("Bearer "+token, "DELETE_USERS"))
}

func TestCallerHasPermission_PermissionInList(t *testing.T) {
	token := makeToken("access", []string{"READ_USERS", "DELETE_USERS"}, time.Hour)
	assert.True(t, callerHasPermission("Bearer "+token, "DELETE_USERS"))
}

func TestCallerHasPermission_CaseInsensitive(t *testing.T) {
	token := makeToken("access", []string{"read_users"}, time.Hour)
	assert.True(t, callerHasPermission("Bearer "+token, "READ_USERS"))
}

func TestCallerHasPermission_AdminBypassesAll(t *testing.T) {
	token := makeToken("access", []string{"ADMIN"}, time.Hour)
	assert.True(t, callerHasPermission("Bearer "+token, "ANY_PERMISSION_WHATSOEVER"))
}
