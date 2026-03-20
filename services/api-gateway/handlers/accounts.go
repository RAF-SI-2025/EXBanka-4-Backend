package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateAccountRequest is the request body for creating a bank account.
type CreateAccountRequest struct {
	ClientID       int64       `json:"clientId"       binding:"required"`
	AccountType    string      `json:"accountType"    binding:"required"`
	CurrencyCode   string      `json:"currencyCode"   binding:"required"`
	InitialBalance float64     `json:"initialBalance"`
	AccountName    string      `json:"accountName"`
	CreateCard     bool        `json:"createCard"`
	CompanyData    *companyReq `json:"companyData"`
}

type companyReq struct {
	Name               string `json:"name"`
	RegistrationNumber string `json:"registrationNumber"`
	PIB                string `json:"pib"`
	ActivityCode       string `json:"activityCode"`
	Address            string `json:"address"`
}

// GetMyAccounts godoc
// @Summary      Get my accounts
// @Description  Returns all accounts for the currently authenticated client, sorted by available balance descending.
// @Tags         accounts
// @Produce      json
// @Success      200  {array}   map[string]interface{}
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/accounts/my [get]
func GetMyAccounts(accountClient pb.AccountServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ownerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := accountClient.GetMyAccounts(ctx, &pb.GetMyAccountsRequest{OwnerId: ownerID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		result := make([]gin.H, 0, len(resp.Accounts))
		for _, a := range resp.Accounts {
			result = append(result, gin.H{
				"accountId":        a.Id,
				"accountName":      a.AccountName,
				"accountNumber":    a.AccountNumber,
				"availableBalance": a.AvailableBalance,
				"currency":         a.CurrencyCode,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

// GetAccount godoc
// @Summary      Get account details
// @Description  Returns detailed info for a single account. The account must belong to the authenticated user.
// @Tags         accounts
// @Produce      json
// @Param        accountId  path      int  true  "Account ID"
// @Success      200        {object}  map[string]interface{}
// @Failure      403        {object}  map[string]string
// @Failure      404        {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/accounts/{accountId} [get]
func GetAccount(accountClient pb.AccountServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, err := parseID(c, "accountId")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid accountId"})
			return
		}
		ownerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := accountClient.GetAccount(ctx, &pb.GetAccountRequest{AccountId: accountID, OwnerId: ownerID})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.PermissionDenied:
				c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		a := resp.Account
		c.JSON(http.StatusOK, gin.H{
			"accountName":      a.AccountName,
			"accountNumber":    a.AccountNumber,
			"owner":            a.Owner,
			"balance":          a.Balance,
			"availableBalance": a.AvailableBalance,
			"reservedFunds":    a.ReservedFunds,
			"currency":         a.CurrencyCode,
			"status":           a.Status,
			"accountType":      a.AccountType,
			"dailyLimit":       a.DailyLimit,
			"monthlyLimit":     a.MonthlyLimit,
			"dailySpent":       a.DailySpent,
			"monthlySpent":     a.MonthlySpent,
		})
	}
}

// RenameAccount godoc
// @Summary      Rename account
// @Description  Changes the name of the account. The account must belong to the authenticated user.
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        accountId  path      int                    true  "Account ID"
// @Param        body       body      map[string]string      true  "New name"
// @Success      200        {object}  map[string]string
// @Failure      400        {object}  map[string]string
// @Failure      403        {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/accounts/{accountId}/name [put]
func RenameAccount(accountClient pb.AccountServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, err := parseID(c, "accountId")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid accountId"})
			return
		}
		ownerID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		var body struct {
			NewAccountName string `json:"newAccountName" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "newAccountName is required"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		_, err = accountClient.RenameAccount(ctx, &pb.RenameAccountRequest{
			AccountId: accountID,
			OwnerId:   ownerID,
			NewName:   body.NewAccountName,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			case codes.PermissionDenied:
				c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "account renamed successfully"})
	}
}

// GetAllAccounts godoc
// @Summary      List all accounts
// @Description  Returns all bank accounts. Requires employee authentication.
// @Tags         accounts
// @Produce      json
// @Success      200  {array}   map[string]interface{}
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/accounts [get]
func GetAllAccounts(accountClient pb.AccountServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := accountClient.GetAllAccounts(ctx, &pb.GetAllAccountsRequest{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		result := make([]gin.H, 0, len(resp.Accounts))
		for _, a := range resp.Accounts {
			result = append(result, gin.H{
				"id":               a.Id,
				"accountNumber":    a.AccountNumber,
				"accountName":      a.AccountName,
				"ownerId":          a.OwnerId,
				"ownerFirstName":   a.OwnerFirstName,
				"ownerLastName":    a.OwnerLastName,
				"accountType":      a.AccountType,
				"currencyCode":     a.CurrencyCode,
				"availableBalance": a.AvailableBalance,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

func parseID(c *gin.Context, param string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(c.Param(param), "%d", &id)
	return id, err
}

// CreateAccount godoc
// @Summary      Create bank account
// @Description  Creates a new bank account for a client. Requires employee authentication.
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        body  body      CreateAccountRequest  true  "Account creation data"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/accounts/create [post]
func CreateAccount(accountClient pb.AccountServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateAccountRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		employeeID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract employee identity from token"})
			return
		}

		grpcReq := &pb.CreateAccountRequest{
			ClientId:       req.ClientID,
			AccountType:    req.AccountType,
			CurrencyCode:   req.CurrencyCode,
			InitialBalance: req.InitialBalance,
			AccountName:    req.AccountName,
			CreateCard:     req.CreateCard,
			EmployeeId:     employeeID,
		}
		if req.CompanyData != nil {
			grpcReq.CompanyData = &pb.CompanyData{
				Name:               req.CompanyData.Name,
				RegistrationNumber: req.CompanyData.RegistrationNumber,
				Pib:                req.CompanyData.PIB,
				ActivityCode:       req.CompanyData.ActivityCode,
				Address:            req.CompanyData.Address,
			}
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := accountClient.CreateAccount(ctx, grpcReq)
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		a := resp.Account
		c.JSON(http.StatusCreated, gin.H{
			"id":                a.Id,
			"accountNumber":     a.AccountNumber,
			"accountName":       a.AccountName,
			"ownerId":           a.OwnerId,
			"employeeId":        a.EmployeeId,
			"currencyCode":      a.CurrencyCode,
			"accountType":       a.AccountType,
			"status":            a.Status,
			"balance":           a.Balance,
			"availableBalance":  a.AvailableBalance,
			"createdDate":       a.CreatedDate,
		})
	}
}
