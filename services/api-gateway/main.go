package main

import (
	"log"
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
// @host            localhost:8081
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
func main() {
	clientClient, clientConn, err := gwgrpc.NewClientClient("localhost:50056")
	if err != nil {
		log.Fatalf("failed to connect to client-service: %v", err)
	}
	defer clientConn.Close()

	employeeClient, empConn, err := gwgrpc.NewEmployeeClient("localhost:50051")
	if err != nil {
		log.Fatalf("failed to connect to employee-service: %v", err)
	}
	defer empConn.Close()

	paymentClient, pmConn, err := gwgrpc.NewPaymentClient("localhost:50055")
	if err != nil {
		log.Fatalf("failed to connect to payment-service: %v", err)
	}
	defer pmConn.Close()

	accountClient, accConn, err := gwgrpc.NewAccountClient("localhost:50054")
	if err != nil {
		log.Fatalf("failed to connect to account-service: %v", err)
	}
	defer accConn.Close()

	authClient, authConn, err := gwgrpc.NewAuthClient("localhost:50052")
	if err != nil {
		log.Fatalf("failed to connect to auth-service: %v", err)
	}
	defer authConn.Close()

	emailClient, emailConn, err := gwgrpc.NewEmailClient("localhost:50053")
	if err != nil {
		log.Fatalf("failed to connect to email-service: %v", err)
	}
	defer emailConn.Close()

	loanClient, loanConn, err := gwgrpc.NewLoanClient("localhost:50058")
	if err != nil {
		log.Fatalf("failed to connect to loan-service: %v", err)
	}
	defer loanConn.Close()

	_, cardConn, err := gwgrpc.NewCardClient("localhost:50059")
	if err != nil {
		log.Fatalf("failed to connect to card-service: %v", err)
	}
	defer cardConn.Close()

	exchangeClient, exchangeConn, err := gwgrpc.NewExchangeClient("localhost:50057")
	if err != nil {
		log.Fatalf("failed to connect to exchange-service: %v", err)
	}
	defer exchangeConn.Close()

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
	r.POST("/api/payments/create", handlers.CreatePayment(paymentClient))
	r.GET("/api/payments", handlers.GetPayments(paymentClient))
	r.GET("/api/payments/:paymentId", handlers.GetPaymentById(paymentClient))
	r.POST("/api/transfers", handlers.CreateTransfer(paymentClient))
	r.GET("/api/transfers", handlers.GetTransfers(paymentClient))
	r.POST("/api/recipients", handlers.CreatePaymentRecipient(paymentClient))
	r.GET("/api/recipients", handlers.GetPaymentRecipients(paymentClient))
	r.PUT("/api/recipients/:id", handlers.UpdatePaymentRecipient(paymentClient))
	r.DELETE("/api/recipients/:id", handlers.DeletePaymentRecipient(paymentClient))
	r.GET("/api/accounts", middleware.RequireRole("EMPLOYEE"), handlers.GetAllAccounts(accountClient))
	r.GET("/api/admin/accounts/:accountId", middleware.RequireRole("EMPLOYEE"), handlers.GetAccountAdmin(accountClient))
	r.GET("/api/accounts/my", handlers.GetMyAccounts(accountClient))
	r.GET("/api/accounts/:accountId", handlers.GetAccount(accountClient))
	r.PUT("/api/accounts/:accountId/name", handlers.RenameAccount(accountClient))
	r.PUT("/api/accounts/:accountId/limits", middleware.RequireRole("EMPLOYEE"), handlers.UpdateAccountLimits(accountClient))
	r.POST("/api/accounts/create", middleware.RequireRole("EMPLOYEE"), handlers.CreateAccount(accountClient))
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
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Run(":8083")
}
