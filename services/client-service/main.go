package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	clientdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/client-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/client-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
)

func main() {
	database, err := clientdb.Connect("postgres://client_user:client_pass@localhost:5435/client_db?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	lis, err := net.Listen("tcp", ":50056")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterClientServiceServer(s, &handlers.ClientServer{DB: database})
	log.Println("client-service listening on :50056")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
