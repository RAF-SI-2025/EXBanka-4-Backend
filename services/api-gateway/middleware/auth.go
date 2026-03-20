package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// GetUserIDFromToken extracts the user_id claim from the Bearer token in the request.
func GetUserIDFromToken(c *gin.Context) (int64, error) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return 0, fmt.Errorf("missing token")
	}
	tokenStr := strings.TrimPrefix(header, "Bearer ")

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid token claims")
	}

	switch v := claims["user_id"].(type) {
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		return 0, fmt.Errorf("user_id not found in token")
	}
}

const jwtSecret = "secret-key-change-in-production"

// RequireRole returns a Gin middleware that validates the JWT and checks that
// the caller has the given role (or ADMIN, which bypasses all role checks).
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		if claims["type"] != "access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
			return
		}

		required := strings.ToUpper(role)
		dozvole, _ := claims["dozvole"].([]interface{})
		for _, d := range dozvole {
			s := strings.ToUpper(fmt.Sprintf("%v", d))
			if s == "ADMIN" || s == required {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}
