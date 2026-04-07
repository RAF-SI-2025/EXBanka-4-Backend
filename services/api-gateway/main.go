package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	gwgrpc "github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/grpc"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/handlers"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/middleware"
	_ "github.com/RAF-SI-2025/EXBanka-4-Backend/services/api-gateway/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           EXBanka API Gateway
// @version         1.0
// @description     REST API gateway for EXBanka microservices.
// @host            localhost:8083
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
func main() {
	clientClient, clientConn, err := gwgrpc.NewClientClient(os.Getenv("CLIENT_SERVICE_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to client-service: %v", err)
	}
	defer clientConn.Close()

	employeeClient, empConn, err := gwgrpc.NewEmployeeClient(os.Getenv("EMPLOYEE_SERVICE_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to employee-service: %v", err)
	}
	defer empConn.Close()

	paymentClient, pmConn, err := gwgrpc.NewPaymentClient(os.Getenv("PAYMENT_SERVICE_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to payment-service: %v", err)
	}
	defer pmConn.Close()

	accountClient, accConn, err := gwgrpc.NewAccountClient(os.Getenv("ACCOUNT_SERVICE_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to account-service: %v", err)
	}
	defer accConn.Close()

	authClient, authConn, err := gwgrpc.NewAuthClient(os.Getenv("AUTH_SERVICE_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to auth-service: %v", err)
	}
	defer authConn.Close()

	emailClient, emailConn, err := gwgrpc.NewEmailClient(os.Getenv("EMAIL_SERVICE_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to email-service: %v", err)
	}
	defer emailConn.Close()

	loanClient, loanConn, err := gwgrpc.NewLoanClient(os.Getenv("LOAN_SERVICE_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to loan-service: %v", err)
	}
	defer loanConn.Close()

	cardClient, cardConn, err := gwgrpc.NewCardClient(os.Getenv("CARD_SERVICE_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to card-service: %v", err)
	}
	defer cardConn.Close()

	exchangeClient, exchangeConn, err := gwgrpc.NewExchangeClient(os.Getenv("EXCHANGE_SERVICE_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to exchange-service: %v", err)
	}
	defer exchangeConn.Close()

	securitiesClient, securitiesConn, err := gwgrpc.NewSecuritiesClient(os.Getenv("SECURITIES_SERVICE_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to securities-service: %v", err)
	}
	defer securitiesConn.Close()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/employees/:id", middleware.RequireRole("ADMIN"), handlers.GetEmployeeById(employeeClient))
	r.GET("/employees", middleware.RequireRole("ADMIN"), handlers.GetEmployees(employeeClient))
	r.GET("/employees/search", middleware.RequireRole("ADMIN"), handlers.SearchEmployees(employeeClient))
	r.PUT("/employees/:id", middleware.RequireRole("ADMIN"), handlers.UpdateEmployee(employeeClient))
	r.POST("/employees", middleware.RequireRole("ADMIN"), handlers.CreateEmployee(employeeClient, authClient, emailClient))
	r.GET("/api/actuaries", middleware.RequireRole("SUPERVISOR"), handlers.GetActuaries(employeeClient))
	r.PUT("/api/actuaries/:id/limit", middleware.RequireRole("SUPERVISOR"), handlers.SetAgentLimit(employeeClient))
	r.POST("/api/actuaries/:id/reset-used-limit", middleware.RequireRole("SUPERVISOR"), handlers.ResetAgentUsedLimit(employeeClient))
	r.PUT("/api/actuaries/:id/need-approval", middleware.RequireRole("SUPERVISOR"), handlers.SetNeedApproval(employeeClient))
	r.POST("/api/payments/create", handlers.CreatePayment(paymentClient))
	r.GET("/api/payments", handlers.GetPayments(paymentClient))
	r.GET("/api/payments/:paymentId", handlers.GetPaymentById(paymentClient))
	r.POST("/api/transfers", handlers.CreateTransfer(paymentClient))
	r.GET("/api/transfers", handlers.GetTransfers(paymentClient))
	r.GET("/api/transfers/my", handlers.GetTransfers(paymentClient))
	r.POST("/api/recipients", handlers.CreatePaymentRecipient(paymentClient))
	r.GET("/api/recipients", handlers.GetPaymentRecipients(paymentClient))
	r.PUT("/api/recipients/:id", handlers.UpdatePaymentRecipient(paymentClient))
	r.DELETE("/api/recipients/:id", handlers.DeletePaymentRecipient(paymentClient))
	r.PUT("/api/recipients/reorder", handlers.ReorderPaymentRecipients(paymentClient))
	r.GET("/api/accounts", middleware.RequireRole("EMPLOYEE"), handlers.GetAllAccounts(accountClient))
	r.GET("/api/admin/accounts/:accountId", middleware.RequireRole("EMPLOYEE"), handlers.GetAccountAdmin(accountClient))
	r.GET("/api/accounts/my", handlers.GetMyAccounts(accountClient))
	r.GET("/api/accounts/:accountId", handlers.GetAccount(accountClient))
	r.PUT("/api/accounts/:accountId/name", handlers.RenameAccount(accountClient))
	r.PUT("/api/accounts/:accountId/limits", middleware.RequireRole("EMPLOYEE"), handlers.UpdateAccountLimits(accountClient))
	r.POST("/api/accounts/create", middleware.RequireRole("EMPLOYEE"), handlers.CreateAccount(accountClient, cardClient))
	r.DELETE("/api/accounts/:accountId", middleware.RequireRole("EMPLOYEE"), handlers.DeleteAccount(accountClient))
	r.GET("/api/bank-accounts", middleware.RequireRole("EMPLOYEE"), handlers.GetBankAccounts(accountClient))
	r.POST("/login", handlers.Login(authClient))
	r.POST("/refresh", handlers.Refresh(authClient))
	r.POST("/client/login", handlers.ClientLogin(authClient))
	r.POST("/client/refresh", handlers.ClientRefresh(authClient))
	r.GET("/client/me", handlers.GetMe(clientClient))
	r.POST("/auth/activate", handlers.Activate(authClient))
	r.POST("/auth/forgot-password", handlers.ForgotPassword(authClient, emailClient))
	r.POST("/auth/reset-password", handlers.ResetPassword(authClient))
	r.GET("/clients", middleware.RequireRole("EMPLOYEE"), handlers.GetClients(clientClient))
	r.GET("/clients/:id", middleware.RequireRole("EMPLOYEE"), handlers.GetClientById(clientClient))
	r.POST("/clients", middleware.RequireRole("EMPLOYEE"), handlers.CreateClient(clientClient, authClient, emailClient))
	r.PUT("/clients/:id", middleware.RequireRole("EMPLOYEE"), handlers.UpdateClient(clientClient))
	r.POST("/client/activate", handlers.ActivateClient(authClient))
	r.GET("/api/approvals/:id/poll", handlers.PollLoginApproval(authClient))
	r.POST("/api/mobile/approvals", handlers.CreateApproval(authClient))
	r.GET("/api/mobile/approvals", handlers.GetMyApprovals(authClient))
	r.GET("/api/mobile/approvals/:id", handlers.GetMyApprovalById(authClient))
	r.PUT("/api/twofactor/:id/approve", handlers.ApproveApproval(authClient, accountClient, paymentClient))
	r.PUT("/api/twofactor/:id/reject", handlers.RejectApproval(authClient))
	r.POST("/api/mobile/push-token", handlers.RegisterMobilePushToken(authClient))
	r.DELETE("/api/mobile/push-token", handlers.UnregisterMobilePushToken(authClient))
	r.GET("/exchange/rates", handlers.GetExchangeRates(exchangeClient))
	r.GET("/exchange/rate", handlers.GetExchangeRate(exchangeClient))
	r.POST("/exchange/convert", handlers.ConvertAmount(exchangeClient))
	r.GET("/exchange/history", handlers.GetExchangeHistory(exchangeClient))
	r.POST("/exchange/preview", handlers.PreviewConversion(exchangeClient))
	r.GET("/loans", handlers.GetMyLoans(loanClient))
	r.GET("/loans/:id", handlers.GetLoanDetails(loanClient))
	r.GET("/loans/:id/installments", handlers.GetLoanInstallments(loanClient))
	r.POST("/loans/apply", handlers.ApplyForLoan(loanClient))
	r.GET("/admin/loans/applications", middleware.RequireRole("ADMIN"), handlers.GetAllLoanApplications(loanClient))
	r.PUT("/admin/loans/:id/approve", middleware.RequireRole("ADMIN"), handlers.ApproveLoan(loanClient))
	r.PUT("/admin/loans/:id/reject", middleware.RequireRole("ADMIN"), handlers.RejectLoan(loanClient))
	r.GET("/admin/loans", middleware.RequireRole("ADMIN"), handlers.GetAllLoans(loanClient))
	r.POST("/admin/loans/trigger-installments", middleware.RequireRole("ADMIN"), handlers.TriggerInstallments(loanClient))
	r.GET("/api/cards", handlers.GetMyCards(accountClient, cardClient))
	r.GET("/api/cards/by-account/:accountNumber", middleware.RequireRole("EMPLOYEE"), handlers.GetCardsByAccount(cardClient))
	r.POST("/api/cards/request", handlers.InitiateCardRequest(cardClient, clientClient, emailClient))
	r.POST("/api/cards/request/confirm", handlers.ConfirmCardRequest(cardClient))
	r.GET("/api/cards/id/:id", handlers.GetCardById(accountClient, cardClient))
	r.GET("/api/cards/:number", handlers.GetCardByNumber(cardClient))
	r.PUT("/api/cards/:id/block", handlers.BlockCard(cardClient))
	r.PUT("/api/cards/:id/unblock", middleware.RequireRole("EMPLOYEE"), handlers.UnblockCard(cardClient))
	r.PUT("/api/cards/:id/deactivate", middleware.RequireRole("EMPLOYEE"), handlers.DeactivateCard(cardClient))
	r.PUT("/api/cards/:id/limit", middleware.RequireRole("EMPLOYEE"), handlers.UpdateCardLimit(cardClient))
	r.GET("/stock-exchanges", middleware.RequireRole("EMPLOYEE"), handlers.GetStockExchanges(securitiesClient))
	r.POST("/stock-exchanges", middleware.RequireRole("ADMIN"), handlers.CreateStockExchange(securitiesClient))
	r.GET("/stock-exchanges/test-mode", middleware.RequireRole("ADMIN"), handlers.GetTestMode(securitiesClient))
	r.POST("/stock-exchanges/test-mode", middleware.RequireRole("ADMIN"), handlers.SetTestMode(securitiesClient))
	r.GET("/stock-exchanges/:mic", middleware.RequireRole("EMPLOYEE"), handlers.GetStockExchangeByMIC(securitiesClient))
	r.PUT("/stock-exchanges/:mic", middleware.RequireRole("ADMIN"), handlers.UpdateStockExchange(securitiesClient))
	r.DELETE("/stock-exchanges/:mic", middleware.RequireRole("ADMIN"), handlers.DeleteStockExchange(securitiesClient))
	r.GET("/stock-exchanges/:mic/hours", middleware.RequireRole("EMPLOYEE"), handlers.GetWorkingHours(securitiesClient))
	r.POST("/stock-exchanges/hours", middleware.RequireRole("ADMIN"), handlers.SetWorkingHours(securitiesClient))
	r.GET("/stock-exchanges/:mic/holidays", middleware.RequireRole("EMPLOYEE"), handlers.GetHolidays(securitiesClient))
	r.POST("/stock-exchanges/holidays", middleware.RequireRole("ADMIN"), handlers.AddHoliday(securitiesClient))
	r.DELETE("/stock-exchanges/holidays/:polity/:date", middleware.RequireRole("ADMIN"), handlers.DeleteHoliday(securitiesClient))
	r.GET("/stock-exchanges/:mic/is-open", middleware.RequireRole("EMPLOYEE"), handlers.IsExchangeOpen(securitiesClient))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Run(":8083")
}
