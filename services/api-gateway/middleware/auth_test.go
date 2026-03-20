package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
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
