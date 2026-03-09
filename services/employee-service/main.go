package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	empdb "github.com/exbanka/backend/services/employee-service/db"
	"github.com/exbanka/backend/services/employee-service/handlers"
	pb "github.com/exbanka/backend/shared/pb/employee"
)

func main() {
	database, err := empdb.Connect("postgres://employee_user:employee_pass@localhost:5433/employee_db?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterEmployeeServiceServer(s, &handlers.EmployeeServer{DB: database})
	log.Println("employee-service listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
