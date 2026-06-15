package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	pb_email "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoginRequest contains credentials for login.
type LoginRequest struct {
	Email    string `json:"email"    example:"jdoe@ankabanka.com"`
	Password string `json:"password" example:"secret"`
}

// TokenResponse is returned on successful login.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshRequest contains the refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// AccessTokenResponse is returned on successful token refresh.
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// ActivateRequest contains the activation payload.
type ActivateRequest struct {
	Token           string `json:"token"            binding:"required"`
	Password        string `json:"password"         binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// Login godoc
// @Summary      Login
// @Description  Authenticate with email and password, receive JWT tokens.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "Login credentials"
// @Success      200  {object}  TokenResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Router       /login [post]
func Login(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.Login(ctx, &pb.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			if status.Code(err) == codes.PermissionDenied {
				c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			}
			return
		}

		if resp.RequiresTotp {
			c.JSON(http.StatusOK, gin.H{
				"requires_totp": true,
				"session_token": resp.SessionToken,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"access_token":  resp.AccessToken,
			"refresh_token": resp.RefreshToken,
		})
	}
}

// Activate godoc
// @Summary      Activate account
// @Description  Activate a new employee account using an activation token and set a password.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body ActivateRequest true "Activation payload"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/activate [post]
func Activate(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Token           string `json:"token"            binding:"required"`
			Password        string `json:"password"         binding:"required"`
			ConfirmPassword string `json:"confirm_password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		_, err := client.ActivateAccount(ctx, &pb.ActivateAccountRequest{
			Token:           req.Token,
			Password:        req.Password,
			ConfirmPassword: req.ConfirmPassword,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "invalid or expired token"})
			case codes.FailedPrecondition:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "account activated successfully"})
	}
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Exchange a valid refresh token for a new access token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body RefreshRequest true "Refresh token"
// @Success      200  {object}  AccessTokenResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Router       /refresh [post]
func Refresh(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.Refresh(ctx, &pb.RefreshRequest{
			RefreshToken: req.RefreshToken,
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"access_token": resp.AccessToken})
	}
}

// ForgotPassword godoc
// @Summary      Request password reset
// @Description  Send a password reset email to the given address.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body object true "Email address"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/forgot-password [post]
func ForgotPassword(authClient pb.AuthServiceClient, emailClient pb_email.EmailServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := authClient.RequestPasswordReset(ctx, &pb.RequestPasswordResetRequest{Email: req.Email})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		frontendURL := os.Getenv("FRONTEND_URL")
		if frontendURL == "" {
			frontendURL = "http://localhost:5173"
		}
		resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, resp.Token)
		go func() {
			emailCtx, emailCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer emailCancel()
			if _, err := emailClient.SendPasswordResetEmail(emailCtx, &pb_email.SendPasswordResetEmailRequest{
				Email:     resp.Email,
				FirstName: resp.FirstName,
				ResetLink: resetLink,
			}); err != nil {
				// log but don't fail the request
				_ = err
			}
		}()

		c.JSON(http.StatusOK, gin.H{"message": "password reset email sent"})
	}
}

// ClientLogin godoc
// @Summary      Client login
// @Description  Authenticate a client with email and password, receive JWT tokens.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "Login credentials"
// @Success      200  {object}  TokenResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Router       /client/login [post]
func ClientLogin(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Source   string `json:"source"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.ClientLogin(ctx, &pb.ClientLoginRequest{
			Email:    req.Email,
			Password: req.Password,
			Source:   req.Source,
		})
		if err != nil {
			if status.Code(err) == codes.PermissionDenied {
				c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			}
			return
		}
		if resp.RequiresTotp {
			c.JSON(http.StatusOK, gin.H{
				"requires_totp": true,
				"session_token": resp.SessionToken,
			})
			return
		}
		if resp.ApprovalRequestId != 0 {
			c.JSON(http.StatusOK, gin.H{"approvalRequestId": resp.ApprovalRequestId})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"access_token":  resp.AccessToken,
			"refresh_token": resp.RefreshToken,
		})
	}
}

func TOTPGenerate(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)
		email := getEmailFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.GenerateTOTPSecret(ctx, &pb.GenerateTOTPRequest{
			UserId:   userID,
			UserType: userType,
			Email:    email,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate TOTP secret"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"secret":      resp.Secret,
			"otpauth_uri": resp.OtpauthUri,
			"qr_code_png": resp.QrCodePng,
		})
	}
}

func TOTPVerify(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Code string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		verifyResp, err := client.VerifyTOTP(ctx, &pb.VerifyTOTPRequest{
			UserId:   userID,
			UserType: userType,
			Code:     req.Code,
		})
		if err != nil || !verifyResp.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid TOTP code"})
			return
		}

		_, err = client.EnableTOTP(ctx, &pb.EnableTOTPRequest{
			UserId:   userID,
			UserType: userType,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable TOTP"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "TOTP enabled"})
	}
}

func TOTPValidateLogin(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SessionToken string `json:"session_token" binding:"required"`
			Code         string `json:"code"          binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := client.ValidateTOTPLogin(ctx, &pb.ValidateTOTPLoginRequest{
			SessionToken: req.SessionToken,
			Code:         req.Code,
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid TOTP code"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"access_token":  resp.AccessToken,
			"refresh_token": resp.RefreshToken,
		})
	}
}

func TOTPDisable(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userType := middleware.GetCallerRoleFromToken(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if _, err := client.DisableTOTP(ctx, &pb.DisableTOTPRequest{
			UserId:   userID,
			UserType: userType,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable TOTP"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "TOTP disabled"})
	}
}

func getEmailFromToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	token, err := jwt.Parse(strings.TrimPrefix(header, "Bearer "), func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return ""
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ""
	}
	email, _ := claims["email"].(string)
	return email
}

// ClientRefresh godoc
// @Summary      Refresh client access token
// @Description  Exchange a valid client refresh token for a new access token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body RefreshRequest true "Refresh token"
// @Success      200  {object}  AccessTokenResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Router       /client/refresh [post]
func ClientRefresh(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.ClientRefresh(ctx, &pb.ClientRefreshRequest{
			RefreshToken: req.RefreshToken,
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"access_token": resp.AccessToken})
	}
}

// ActivateClient godoc
// @Summary      Activate client account
// @Description  Activate a new client account using an activation token and set a password.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body ActivateRequest true "Activation payload"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /client/activate [post]
func ActivateClient(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Token           string `json:"token"            binding:"required"`
			Password        string `json:"password"         binding:"required"`
			ConfirmPassword string `json:"confirm_password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		_, err := client.ActivateClient(ctx, &pb.ActivateClientRequest{
			Token:           req.Token,
			Password:        req.Password,
			ConfirmPassword: req.ConfirmPassword,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "invalid or expired token"})
			case codes.FailedPrecondition:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "account activated successfully"})
	}
}

// Logout godoc
// @Summary      Logout
// @Description  Revoke the current access token (adds jti to Redis blacklist).
// @Tags         auth
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /auth/logout [post]
func Logout(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		_, _ = client.Logout(ctx, &pb.LogoutRequest{Token: tokenStr})
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
	}
}

// ResetPassword godoc
// @Summary      Reset password
// @Description  Set a new password using a password reset token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body ActivateRequest true "Reset payload"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/reset-password [post]
func ResetPassword(client pb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Token           string `json:"token"            binding:"required"`
			Password        string `json:"password"         binding:"required"`
			ConfirmPassword string `json:"confirm_password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		_, err := client.ResetPassword(ctx, &pb.ResetPasswordRequest{
			Token:           req.Token,
			Password:        req.Password,
			ConfirmPassword: req.ConfirmPassword,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "invalid or expired token"})
			case codes.FailedPrecondition:
				c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
	}
}
