package main

import (
	"log"

	"github.com/gin-gonic/gin"
	gwgrpc "github.com/exbanka/backend/services/api-gateway/grpc"
	"github.com/exbanka/backend/services/api-gateway/handlers"
)

func main() {
	employeeClient, conn, err := gwgrpc.NewEmployeeClient("localhost:50051")
	if err != nil {
		log.Fatalf("failed to connect to employee-service: %v", err)
	}
	defer conn.Close()

	r := gin.Default()
	r.GET("/employees", handlers.GetEmployees(employeeClient))
	r.GET("/employees/search", handlers.SearchEmployees(employeeClient))
	r.Run(":8081")
}
