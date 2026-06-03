package middleware

import (
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

// newRateLimitRedis starts miniredis and returns a connected go-redis client + cleanup fn.
func newRateLimitRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

// makeRLToken makes a valid access token with the given user ID.
func makeRLToken(t *testing.T, userID int64) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id": float64(userID),
		"type":    "access",
		"dozvole": []string{"AGENT"},
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	require.NoError(t, err)
	return tok
}

// runRL builds a router with RateLimit middleware and runs n requests against /test.
// Returns the HTTP status code of each response.
func runRL(rdb *redis.Client, limit int, window time.Duration, authHeader string, n int) []int {
	router := gin.New()
	router.GET("/test", RateLimit(rdb, limit, window), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	codes := make([]int, n)
	for i := range n {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		router.ServeHTTP(w, req)
		codes[i] = w.Code
	}
	return codes
}

func TestRateLimit_UnderLimit_AllPass(t *testing.T) {
	rdb := newRateLimitRedis(t)
	codes := runRL(rdb, 5, time.Minute, "", 5)
	for i, code := range codes {
		assert.Equal(t, http.StatusOK, code, "request %d should pass", i+1)
	}
}

func TestRateLimit_ExactlyAtLimit_AllPass(t *testing.T) {
	rdb := newRateLimitRedis(t)
	codes := runRL(rdb, 3, time.Minute, "", 3)
	for _, code := range codes {
		assert.Equal(t, http.StatusOK, code)
	}
}

func TestRateLimit_OverLimit_Returns429(t *testing.T) {
	rdb := newRateLimitRedis(t)
	codes := runRL(rdb, 3, time.Minute, "", 5)
	assert.Equal(t, http.StatusOK, codes[0])
	assert.Equal(t, http.StatusOK, codes[1])
	assert.Equal(t, http.StatusOK, codes[2])
	assert.Equal(t, http.StatusTooManyRequests, codes[3])
	assert.Equal(t, http.StatusTooManyRequests, codes[4])
}

func TestRateLimit_RetryAfterHeader(t *testing.T) {
	rdb := newRateLimitRedis(t)
	router := gin.New()
	router.GET("/test", RateLimit(rdb, 1, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// First request consumes the quota.
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request hits the limit.
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	assert.Equal(t, "60", w2.Header().Get("Retry-After"))
}

func TestRateLimit_DifferentPaths_IndependentCounters(t *testing.T) {
	rdb := newRateLimitRedis(t)
	router := gin.New()
	router.GET("/a", RateLimit(rdb, 1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/b", RateLimit(rdb, 1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	// /a: first request OK, second limited.
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/a", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/a", nil))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// /b counter is independent — first request must still pass.
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/b", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimit_DifferentUsers_IndependentCounters(t *testing.T) {
	rdb := newRateLimitRedis(t)
	router := gin.New()
	router.GET("/test", RateLimit(rdb, 1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	tok1 := "Bearer " + makeRLToken(t, 1)
	tok2 := "Bearer " + makeRLToken(t, 2)

	// User 1: first request OK, second limited.
	sendAuth := func(auth string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", auth)
		router.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, sendAuth(tok1))
	assert.Equal(t, http.StatusTooManyRequests, sendAuth(tok1))

	// User 2 has its own counter — still OK.
	assert.Equal(t, http.StatusOK, sendAuth(tok2))
}

func TestRateLimit_AuthenticatedKeyedByUserID(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	tok := makeRLToken(t, 42)
	router := gin.New()
	router.GET("/test", RateLimit(rdb, 5, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Key must be uid-based.
	keys := mr.Keys()
	assert.Len(t, keys, 1)
	assert.Contains(t, keys[0], "uid:42")
}

func TestRateLimit_UnauthenticatedKeyedByIP(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	router := gin.New()
	router.GET("/test", RateLimit(rdb, 5, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	keys := mr.Keys()
	assert.Len(t, keys, 1)
	assert.Contains(t, keys[0], "ip:")
}

func TestRateLimit_WindowExpiry_ResetsCounter(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	router := gin.New()
	router.GET("/test", RateLimit(rdb, 2, 5*time.Second), func(c *gin.Context) { c.Status(http.StatusOK) })

	send := func() int {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
		return w.Code
	}

	assert.Equal(t, http.StatusOK, send())
	assert.Equal(t, http.StatusOK, send())
	assert.Equal(t, http.StatusTooManyRequests, send())

	// Fast-forward miniredis time past the window.
	mr.FastForward(6 * time.Second)

	// Counter has expired — next request should pass again.
	assert.Equal(t, http.StatusOK, send())
}

func TestRateLimit_RedisDown_FailsOpen(t *testing.T) {
	// Point client at a port with nothing listening; no retries so the test is fast.
	rdb := redis.NewClient(&redis.Options{
		Addr:        "localhost:19999",
		DialTimeout: 50 * time.Millisecond,
		MaxRetries:  0,
	})
	codes := runRL(rdb, 1, time.Minute, "", 5)
	for _, code := range codes {
		assert.Equal(t, http.StatusOK, code, "should fail open when Redis is unreachable")
	}
}

func TestRateLimit_KeyFormat(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	tok := makeRLToken(t, 7)
	router := gin.New()
	router.GET("/exchange/rates", RateLimit(rdb, 10, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/exchange/rates", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(w, req)

	keys := mr.Keys()
	require.Len(t, keys, 1)
	expected := "ratelimit:uid:7:/exchange/rates"
	assert.Equal(t, expected, keys[0])
}
