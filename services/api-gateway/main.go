package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	gwgrpc "github.com/exbanka/backend/services/api-gateway/grpc"
	"github.com/exbanka/backend/services/api-gateway/handlers"
	"github.com/exbanka/backend/services/api-gateway/middleware"
	_ "github.com/exbanka/backend/services/api-gateway/docs"
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
	employeeClient, empConn, err := gwgrpc.NewEmployeeClient("localhost:50051")
	if err != nil {
		log.Fatalf("failed to connect to employee-service: %v", err)
	}
	defer empConn.Close()

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
	r.POST("/login", handlers.Login(authClient))
	r.POST("/refresh", handlers.Refresh(authClient))
	r.POST("/auth/activate", handlers.Activate(authClient))
	r.POST("/auth/forgot-password", handlers.ForgotPassword(authClient, emailClient))
	r.POST("/auth/reset-password", handlers.ResetPassword(authClient))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Run(":8081")
}
