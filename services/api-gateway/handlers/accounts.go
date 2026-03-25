package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	pbcard "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/card"
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
	AccountSubtype string      `json:"accountSubtype"`
	DailyLimit     float64     `json:"dailyLimit"`
	MonthlyLimit   float64     `json:"monthlyLimit"`
	CreateCard     bool        `json:"createCard"`
	CardName       string      `json:"cardName"`
	CardLimit      float64     `json:"cardLimit"`
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
		body := gin.H{
			"accountName":      a.AccountName,
			"accountNumber":    a.AccountNumber,
			"owner":            a.Owner,
			"balance":          a.Balance,
			"availableBalance": a.AvailableBalance,
			"reservedFunds":    a.ReservedFunds,
			"currency":         a.CurrencyCode,
			"status":           a.Status,
			"accountType":      a.AccountType,
			"accountSubtype":   a.AccountSubtype,
			"dailyLimit":       a.DailyLimit,
			"monthlyLimit":     a.MonthlyLimit,
			"dailySpent":       a.DailySpent,
			"monthlySpent":     a.MonthlySpent,
		}
		if a.CompanyData != nil {
			body["company"] = gin.H{
				"name":               a.CompanyData.Name,
				"registrationNumber": a.CompanyData.RegistrationNumber,
				"pib":                a.CompanyData.Pib,
				"activityCode":       a.CompanyData.ActivityCode,
				"address":            a.CompanyData.Address,
			}
		}
		c.JSON(http.StatusOK, body)
	}
}

// GetAccountAdmin godoc
// @Summary      Get account details (employee)
// @Description  Returns full account details for any account. Requires employee authentication.
// @Tags         accounts
// @Produce      json
// @Param        accountId  path      int  true  "Account ID"
// @Success      200        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/admin/accounts/{accountId} [get]
func GetAccountAdmin(accountClient pb.AccountServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, err := parseID(c, "accountId")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid accountId"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// OwnerId=0 bypasses the ownership check in account-service
		resp, err := accountClient.GetAccount(ctx, &pb.GetAccountRequest{AccountId: accountID, OwnerId: 0})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		a := resp.Account
		body := gin.H{
			"accountName":      a.AccountName,
			"accountNumber":    a.AccountNumber,
			"owner":            a.Owner,
			"balance":          a.Balance,
			"availableBalance": a.AvailableBalance,
			"reservedFunds":    a.ReservedFunds,
			"currency":         a.CurrencyCode,
			"status":           a.Status,
			"accountType":      a.AccountType,
			"accountSubtype":   a.AccountSubtype,
			"dailyLimit":       a.DailyLimit,
			"monthlyLimit":     a.MonthlyLimit,
			"dailySpent":       a.DailySpent,
			"monthlySpent":     a.MonthlySpent,
		}
		if a.CompanyData != nil {
			body["company"] = gin.H{
				"name":               a.CompanyData.Name,
				"registrationNumber": a.CompanyData.RegistrationNumber,
				"pib":                a.CompanyData.Pib,
				"activityCode":       a.CompanyData.ActivityCode,
				"address":            a.CompanyData.Address,
			}
		}
		c.JSON(http.StatusOK, body)
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

// UpdateAccountLimits godoc
// @Summary      Update account limits
// @Description  Sets the daily and monthly spending limits for an account. Requires employee authentication.
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        accountId  path      int                    true  "Account ID"
// @Param        body       body      map[string]number      true  "Limits"
// @Success      200        {object}  map[string]string
// @Failure      400        {object}  map[string]string
// @Failure      404        {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/accounts/{accountId}/limits [put]
func UpdateAccountLimits(accountClient pb.AccountServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, err := parseID(c, "accountId")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid accountId"})
			return
		}

		var body struct {
			DailyLimit   float64 `json:"dailyLimit"   binding:"required"`
			MonthlyLimit float64 `json:"monthlyLimit" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dailyLimit and monthlyLimit are required"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// OwnerId=0 signals employee/admin access — no ownership check in account-service
		_, err = accountClient.UpdateAccountLimits(ctx, &pb.UpdateAccountLimitsRequest{
			AccountId:    accountID,
			OwnerId:      0,
			DailyLimit:   body.DailyLimit,
			MonthlyLimit: body.MonthlyLimit,
		})
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "limits updated successfully"})
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
func CreateAccount(accountClient pb.AccountServiceClient, cardClient pbcard.CardServiceClient) gin.HandlerFunc {
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
			ClientId:        req.ClientID,
			AccountType:     req.AccountType,
			AccountSubtype:  req.AccountSubtype,
			CurrencyCode:    req.CurrencyCode,
			InitialBalance:  req.InitialBalance,
			AccountName:     req.AccountName,
			CreateCard:      req.CreateCard,
			EmployeeId:      employeeID,
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

		// Set transaction limits if provided
		if req.DailyLimit > 0 || req.MonthlyLimit > 0 {
			limCtx, limCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer limCancel()
			_, limErr := accountClient.UpdateAccountLimits(limCtx, &pb.UpdateAccountLimitsRequest{
				AccountId:    a.Id,
				OwnerId:      0, // bypass ownership check
				DailyLimit:   req.DailyLimit,
				MonthlyLimit: req.MonthlyLimit,
			})
			if limErr != nil {
				log.Printf("failed to set limits for account %s: %v", a.AccountNumber, limErr)
			}
		}

		if req.CreateCard {
			cardName := req.CardName
			if cardName == "" {
				cardName = "VISA"
			}
			cardCtx, cardCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cardCancel()
			cardResp, cardErr := cardClient.CreateCard(cardCtx, &pbcard.CreateCardRequest{
				AccountNumber:  a.AccountNumber,
				CardName:       cardName,
				CallerClientId: 0,
				ForSelf:        true,
			})
			if cardErr != nil {
				log.Printf("auto card creation failed for account %s: %v", a.AccountNumber, cardErr)
			} else if req.CardLimit > 0 {
				limitCtx, limitCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer limitCancel()
				_, limitErr := cardClient.UpdateCardLimit(limitCtx, &pbcard.UpdateCardLimitRequest{
					CardNumber: cardResp.Card.CardNumber,
					NewLimit:   req.CardLimit,
				})
				if limitErr != nil {
					log.Printf("failed to set card limit for %s: %v", cardResp.Card.CardNumber, limitErr)
				}
			}
		}

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
