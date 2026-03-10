package main

import (
	"log"

	"github.com/gin-gonic/gin"
	gwgrpc "github.com/exbanka/backend/services/api-gateway/grpc"
	"github.com/exbanka/backend/services/api-gateway/handlers"
	"github.com/exbanka/backend/services/api-gateway/middleware"
)

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

	r := gin.Default()
	r.GET("/employees", middleware.RequireRole("ADMIN"), handlers.GetEmployees(employeeClient))
	r.GET("/employees/search", middleware.RequireRole("ADMIN"), handlers.SearchEmployees(employeeClient))
	r.POST("/employees", middleware.RequireRole("ADMIN"), handlers.CreateEmployee(employeeClient))
	r.POST("/login", handlers.Login(authClient))
	r.POST("/refresh", handlers.Refresh(authClient))
	r.Run(":8081")
}
