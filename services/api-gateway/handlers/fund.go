package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	pb_exchange "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/exchange"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	pb_order "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func fundToJSON(f *pb.FundResponse) gin.H {
	return gin.H{
		"id":                  f.Id,
		"name":                f.Name,
		"description":         f.Description,
		"minimumContribution": f.MinimumContribution,
		"managerId":           f.ManagerId,
		"managerName":         f.ManagerName,
		"liquidAssets":        f.LiquidAssets,
		"fundValue":           f.FundValue,
		"profit":              f.Profit,
		"accountNumber":       f.AccountNumber,
		"createdAt":           f.CreatedAt,
		"active":              f.Active,
		"accountId":           f.AccountId,
	}
}

func mapFundError(c *gin.Context, err error) {
	switch status.Code(err) {
	case codes.PermissionDenied:
		c.JSON(http.StatusForbidden, gin.H{"error": status.Convert(err).Message()})
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
	case codes.AlreadyExists:
		c.JSON(http.StatusConflict, gin.H{"error": status.Convert(err).Message()})
	case codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
	case codes.FailedPrecondition:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": status.Convert(err).Message()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": status.Convert(err).Message()})
	}
}

// CreateFund godoc
// @Summary      Create an investment fund (SUPERVISOR)
// @Tags         investment-funds
// @Accept       json
// @Produce      json
// @Success      201  {object}  map[string]interface{}
// @Router       /investment/funds [post]
func CreateFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		employeeID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		var req struct {
			Name                string  `json:"name"                binding:"required"`
			Description         string  `json:"description"`
			MinimumContribution float64 `json:"minimumContribution" binding:"required"`
			ManagerId           int64   `json:"managerId"           binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		resp, err := client.CreateFund(ctx, &pb.CreateFundRequest{
			Name:                req.Name,
			Description:         req.Description,
			MinimumContribution: req.MinimumContribution,
			ManagerId:           req.ManagerId,
			CreatedById:         employeeID,
		})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusCreated, fundToJSON(resp))
	}
}

// ListFunds godoc
// @Summary      List investment funds
// @Tags         investment-funds
// @Produce      json
// @Param        managerId  query  int  false  "Filter by manager ID"
// @Success      200  {array}  map[string]interface{}
// @Router       /investment/funds [get]
func ListFunds(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		var managerIDFilter int64
		if v := c.Query("managerId"); v != "" {
			managerIDFilter, _ = strconv.ParseInt(v, 10, 64)
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.ListFunds(ctx, &pb.ListFundsRequest{ManagerIdFilter: managerIDFilter})
		if err != nil {
			mapFundError(c, err)
			return
		}

		result := make([]gin.H, 0, len(resp.Funds))
		for _, f := range resp.Funds {
			result = append(result, fundToJSON(f))
		}
		c.JSON(http.StatusOK, result)
	}
}

// GetFund godoc
// @Summary      Get a single investment fund
// @Tags         investment-funds
// @Produce      json
// @Param        id  path  int  true  "Fund ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /investment/funds/{id} [get]
func GetFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		fundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetFund(ctx, &pb.GetFundRequest{Id: fundID})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusOK, fundToJSON(resp))
	}
}

// UpdateFund godoc
// @Summary      Update an investment fund (SUPERVISOR)
// @Tags         investment-funds
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "Fund ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /investment/funds/{id} [put]
func UpdateFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		fundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		var req struct {
			Name                string  `json:"name"`
			Description         string  `json:"description"`
			MinimumContribution float64 `json:"minimumContribution"`
			ManagerId           int64   `json:"managerId"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.UpdateFund(ctx, &pb.UpdateFundRequest{
			Id:                  fundID,
			Name:                req.Name,
			Description:         req.Description,
			MinimumContribution: req.MinimumContribution,
			ManagerId:           req.ManagerId,
		})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusOK, fundToJSON(resp))
	}
}

// DeleteFund godoc
// @Summary      Delete an investment fund (SUPERVISOR)
// @Tags         investment-funds
// @Produce      json
// @Param        id  path  int  true  "Fund ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /investment/funds/{id} [delete]
func DeleteFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := middleware.GetUserIDFromToken(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		fundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		_, err = client.DeleteFund(ctx, &pb.DeleteFundRequest{Id: fundID})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "fund deleted"})
	}
}

func InvestFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		callerRole := middleware.GetCallerRoleFromToken(c)
		clientType := "CLIENT"
		if callerRole == "EMPLOYEE" {
			clientType = "BANK"
		}

		fundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		var req struct {
			SourceAccountId int64   `json:"sourceAccountId" binding:"required"`
			Amount          float64 `json:"amount"          binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		clientID := userID
		if callerRole == "EMPLOYEE" {
			clientID = 0
		}
		resp, err := client.InvestFund(ctx, &pb.InvestFundRequest{
			FundId:          fundID,
			ClientId:        clientID,
			ClientType:      clientType,
			SourceAccountId: req.SourceAccountId,
			Amount:          req.Amount,
		})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusOK, fundToJSON(resp))
	}
}

func WithdrawFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		callerRole := middleware.GetCallerRoleFromToken(c)
		clientType := "CLIENT"
		if callerRole == "EMPLOYEE" {
			clientType = "BANK"
		}

		fundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		var req struct {
			DestinationAccountId int64   `json:"destinationAccountId" binding:"required"`
			Amount               float64 `json:"amount"`
			WithdrawAll          bool    `json:"withdrawAll"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		amount := req.Amount
		if req.WithdrawAll {
			amount = 0
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		withdrawClientID := userID
		if callerRole == "EMPLOYEE" {
			withdrawClientID = 0
		}
		resp, err := client.WithdrawFund(ctx, &pb.WithdrawFundRequest{
			FundId:               fundID,
			ClientId:             withdrawClientID,
			ClientType:           clientType,
			DestinationAccountId: req.DestinationAccountId,
			Amount:               amount,
		})
		if err != nil {
			mapFundError(c, err)
			return
		}
		if resp.Pending {
			c.JSON(http.StatusAccepted, gin.H{"message": resp.Message})
			return
		}
		c.JSON(http.StatusOK, fundToJSON(resp.Fund))
	}
}

// GetBankFundPositions godoc
// @Summary      Bank profit portal — bank's fund positions
// @Description  Returns all investment fund positions held by the bank, with share % and profit. SUPERVISOR only.
// @Tags         bank-profit
// @Produce      json
// @Success      200  {array}   object
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /bank/profit/fund-positions [get]
func GetBankFundPositions(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetBankPositions(ctx, &pb.GetBankPositionsRequest{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		result := make([]gin.H, 0, len(resp.Positions))
		for _, p := range resp.Positions {
			result = append(result, gin.H{
				"fundId":           p.FundId,
				"fundName":         p.FundName,
				"managerName":      p.ManagerName,
				"bankSharePercent": p.BankSharePercent,
				"bankShareRSD":     p.BankShareRsd,
				"profitRSD":        p.ProfitRsd,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

// BankInvestFund godoc
// @Summary      Bank profit portal — bank invests in a fund
// @Description  Invests bank money into the specified fund. SUPERVISOR only. sourceAccountId must be a bank account.
// @Tags         bank-profit
// @Accept       json
// @Produce      json
// @Param        fundId  path      int     true  "Fund ID"
// @Param        body    body      object  true  "amount and sourceAccountId"
// @Success      200  {object}  object
// @Failure      400  {object}  map[string]string
// @Security     BearerAuth
// @Router       /bank/profit/fund-positions/{fundId}/invest [post]
func BankInvestFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		fundID, err := strconv.ParseInt(c.Param("fundId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fundId"})
			return
		}
		var req struct {
			Amount          float64 `json:"amount"          binding:"required"`
			SourceAccountId int64   `json:"sourceAccountId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		resp, err := client.InvestFund(ctx, &pb.InvestFundRequest{
			FundId:          fundID,
			ClientId:        0,
			ClientType:      "BANK",
			SourceAccountId: req.SourceAccountId,
			Amount:          req.Amount,
		})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusOK, fundToJSON(resp))
	}
}

// BankRedeemFund godoc
// @Summary      Bank profit portal — bank redeems from a fund
// @Description  Redeems bank investment from the specified fund. SUPERVISOR only.
// @Tags         bank-profit
// @Accept       json
// @Produce      json
// @Param        fundId  path      int     true  "Fund ID"
// @Param        body    body      object  true  "amount and destinationAccountId"
// @Success      200  {object}  object
// @Failure      400  {object}  map[string]string
// @Security     BearerAuth
// @Router       /bank/profit/fund-positions/{fundId}/redeem [post]
func BankRedeemFund(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		fundID, err := strconv.ParseInt(c.Param("fundId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fundId"})
			return
		}
		var req struct {
			Amount               float64 `json:"amount"               binding:"required"`
			DestinationAccountId int64   `json:"destinationAccountId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		resp, err := client.WithdrawFund(ctx, &pb.WithdrawFundRequest{
			FundId:               fundID,
			ClientId:             0,
			ClientType:           "BANK",
			DestinationAccountId: req.DestinationAccountId,
			Amount:               req.Amount,
		})
		if err != nil {
			mapFundError(c, err)
			return
		}
		c.JSON(http.StatusOK, fundToJSON(resp.Fund))
	}
}

func GetFundSecurities(fundClient pb.FundServiceClient, secClient pb_sec.SecuritiesServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		portfolio, err := fundClient.GetFundPortfolio(ctx, &pb.GetFundPortfolioRequest{FundId: id})
		if err != nil {
			mapFundError(c, err)
			return
		}

		result := make([]gin.H, 0, len(portfolio.Positions))
		for _, pos := range portfolio.Positions {
			secResp, err := secClient.GetListingById(ctx, &pb_sec.GetListingByIdRequest{Id: pos.ListingId})
			if err != nil || secResp.Summary == nil {
				continue
			}
			s := secResp.Summary
			result = append(result, gin.H{
				"ticker":            s.Ticker,
				"name":              s.Name,
				"amount":            pos.Quantity,
				"currentPrice":      s.Price,
				"change":            s.ChangePercent,
				"volume":            s.Volume,
				"initialMarginCost": s.InitialMarginCost,
				"acquisitionDate":   pos.AcquisitionDate,
				"averageCost":       pos.AverageCost,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

func priceToRSD(ctx context.Context, exchangeClient pb_exchange.ExchangeServiceClient, currency string, price float64) float64 {
	if currency == "" || currency == "RSD" || exchangeClient == nil {
		return price
	}
	resp, err := exchangeClient.PreviewConversion(ctx, &pb_exchange.PreviewConversionRequest{
		FromCurrency: currency,
		ToCurrency:   "RSD",
		Amount:       price,
	})
	if err != nil {
		return price
	}
	return resp.ToAmount
}

func BuyFundSecurities(fundClient pb.FundServiceClient, secClient pb_sec.SecuritiesServiceClient, orderClient pb_order.OrderServiceClient, exchangeClient pb_exchange.ExchangeServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}
		var body struct {
			Ticker string `json:"ticker" binding:"required"`
			Amount int32  `json:"amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		secResp, err := secClient.GetListings(ctx, &pb_sec.GetListingsRequest{TickerPrefix: body.Ticker, PageSize: 1})
		if err != nil || len(secResp.Listings) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticker not found"})
			return
		}
		listing := secResp.Listings[0]

		priceRSD := priceToRSD(ctx, exchangeClient, listing.Currency, listing.Price)
		requiredAmount := float64(body.Amount) * priceRSD
		var accountID int64

		if middleware.CallerHasPermission(c, "ADMIN") {
			fundResp, err := fundClient.GetFund(ctx, &pb.GetFundRequest{Id: id})
			if err != nil {
				mapFundError(c, err)
				return
			}
			if fundResp.LiquidAssets < requiredAmount {
				c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient fund liquidity"})
				return
			}
			accountID = fundResp.AccountId
		} else {
			vResp, vErr := fundClient.ValidateFundAccount(ctx, &pb.ValidateFundAccountRequest{
				FundId:         id,
				ManagerId:      userID,
				RequiredAmount: requiredAmount,
			})
			if vErr != nil {
				switch status.Code(vErr) {
				case codes.NotFound:
					c.JSON(http.StatusNotFound, gin.H{"error": "fund not found"})
				case codes.PermissionDenied:
					c.JSON(http.StatusForbidden, gin.H{"error": "you are not the manager of this fund"})
				default:
					c.JSON(http.StatusInternalServerError, gin.H{"error": vErr.Error()})
				}
				return
			}
			if !vResp.IsLiquid {
				c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient fund liquidity"})
				return
			}
			accountID = vResp.AccountId
		}

		resp, err := orderClient.CreateOrder(ctx, &pb_order.CreateOrderRequest{
			UserId:    userID,
			UserType:  middleware.GetCallerRoleFromToken(c),
			AssetId:   listing.Id,
			Quantity:  body.Amount,
			AccountId: accountID,
			Direction: "BUY",
			FundId:    id,
		})
		if err != nil {
			orderError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"orderId":          resp.OrderId,
			"orderType":        resp.OrderType,
			"status":           resp.Status,
			"approximatePrice": resp.ApproximatePrice,
		})
	}
}

func SellFundSecurities(fundClient pb.FundServiceClient, secClient pb_sec.SecuritiesServiceClient, orderClient pb_order.OrderServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}
		var body struct {
			Ticker string `json:"ticker" binding:"required"`
			Amount int32  `json:"amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		secResp, err := secClient.GetListings(ctx, &pb_sec.GetListingsRequest{TickerPrefix: body.Ticker, PageSize: 1})
		if err != nil || len(secResp.Listings) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticker not found"})
			return
		}
		listing := secResp.Listings[0]

		portfolio, err := fundClient.GetFundPortfolio(ctx, &pb.GetFundPortfolioRequest{FundId: id})
		if err != nil {
			mapFundError(c, err)
			return
		}
		var heldQty float64
		for _, pos := range portfolio.Positions {
			if pos.ListingId == listing.Id {
				heldQty = pos.Quantity
				break
			}
		}
		if heldQty == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "fund does not hold this ticker"})
			return
		}
		if heldQty < float64(body.Amount) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient amount in fund portfolio"})
			return
		}

		var accountID int64
		if middleware.CallerHasPermission(c, "ADMIN") {
			fundResp, err := fundClient.GetFund(ctx, &pb.GetFundRequest{Id: id})
			if err != nil {
				mapFundError(c, err)
				return
			}
			accountID = fundResp.AccountId
		} else {
			vResp, vErr := fundClient.ValidateFundAccount(ctx, &pb.ValidateFundAccountRequest{
				FundId:         id,
				ManagerId:      userID,
				RequiredAmount: 0,
			})
			if vErr != nil {
				switch status.Code(vErr) {
				case codes.NotFound:
					c.JSON(http.StatusNotFound, gin.H{"error": "fund not found"})
				case codes.PermissionDenied:
					c.JSON(http.StatusForbidden, gin.H{"error": "you are not the manager of this fund"})
				default:
					c.JSON(http.StatusInternalServerError, gin.H{"error": vErr.Error()})
				}
				return
			}
			accountID = vResp.AccountId
		}

		resp, err := orderClient.CreateOrder(ctx, &pb_order.CreateOrderRequest{
			UserId:    userID,
			UserType:  middleware.GetCallerRoleFromToken(c),
			AssetId:   listing.Id,
			Quantity:  body.Amount,
			AccountId: accountID,
			Direction: "SELL",
			FundId:    id,
		})
		if err != nil {
			orderError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"orderId":          resp.OrderId,
			"orderType":        resp.OrderType,
			"status":           resp.Status,
			"approximatePrice": resp.ApproximatePrice,
		})
	}
}

func GetMyPositions(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not extract identity from token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := client.GetMyPositions(ctx, &pb.GetMyPositionsRequest{
			ClientId:   userID,
			ClientType: "CLIENT",
		})
		if err != nil {
			mapFundError(c, err)
			return
		}

		result := make([]gin.H, 0, len(resp.Positions))
		for _, p := range resp.Positions {
			result = append(result, gin.H{
				"fundId":               p.FundId,
				"fundName":             p.FundName,
				"description":          p.Description,
				"fundValue":            p.FundValue,
				"fundPercentage":       p.FundPercentage,
				"currentPositionValue": p.CurrentPositionValue,
				"totalInvestedAmount":  p.TotalInvestedAmount,
				"profit":               p.Profit,
				"minimumContribution":  p.MinimumContribution,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

func GetFundPerformanceHistory(client pb.FundServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fund id"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := client.GetFundPerformanceHistory(ctx, &pb.GetFundPerformanceRequest{
			FundId: id,
			From:   c.Query("from"),
			To:     c.Query("to"),
		})
		if err != nil {
			mapFundError(c, err)
			return
		}

		type record struct {
			Date      string  `json:"date"`
			FundValue float64 `json:"fundValue"`
			Profit    float64 `json:"profit"`
		}
		out := make([]record, len(resp.Records))
		for i, r := range resp.Records {
			out[i] = record{Date: r.Date, FundValue: r.FundValue, Profit: r.Profit}
		}
		c.JSON(http.StatusOK, out)
	}
}
