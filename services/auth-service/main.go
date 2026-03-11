package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authdb "github.com/exbanka/backend/services/auth-service/db"
	"github.com/exbanka/backend/services/auth-service/handlers"
	pb_auth "github.com/exbanka/backend/shared/pb/auth"
	pb_emp "github.com/exbanka/backend/shared/pb/employee"
)

func main() {
	database, err := authdb.Connect("postgres://auth_user:auth_pass@localhost:5434/auth_db?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to auth-db: %v", err)
	}
	defer database.Close()

	empConn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to employee-service: %v", err)
	}
	defer empConn.Close()

	employeeClient := pb_emp.NewEmployeeServiceClient(empConn)

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb_auth.RegisterAuthServiceServer(s, &handlers.AuthServer{
		DB:             database,
		EmployeeClient: employeeClient,
	})

	log.Println("auth-service listening on :50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
