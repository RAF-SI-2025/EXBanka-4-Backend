package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/auth-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/auth-service/handlers"
	pb_auth "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_email "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
)

func main() {
	database, err := authdb.Connect("postgres://auth_user:auth_pass@localhost:5434/auth_db?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to auth-db: %v", err)
	}
	defer database.Close()

	clientConn, err := grpc.NewClient("localhost:50056", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to client-service: %v", err)
	}
	defer clientConn.Close()

	clientClient := pb_client.NewClientServiceClient(clientConn)

	empConn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to employee-service: %v", err)
	}
	defer empConn.Close()

	employeeClient := pb_emp.NewEmployeeServiceClient(empConn)

	emailConn, err := grpc.NewClient("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to email-service: %v", err)
	}
	defer emailConn.Close()

	emailClient := pb_email.NewEmailServiceClient(emailConn)

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb_auth.RegisterAuthServiceServer(s, &handlers.AuthServer{
		DB:             database,
		EmployeeClient: employeeClient,
		EmailClient:    emailClient,
		ClientClient:   clientClient,
	})

	log.Println("auth-service listening on :50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
