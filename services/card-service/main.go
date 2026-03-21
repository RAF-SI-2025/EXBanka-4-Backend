package main

import (
	"log"
	"net"

	carddb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/card-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/card-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/card"
	"google.golang.org/grpc"
)

const (
	cardDBDSN    = "postgres://card_user:card_pass@localhost:5440/card_db?sslmode=disable"
	accountDBDSN = "postgres://account_user:account_pass@localhost:5436/account_db?sslmode=disable"
	grpcPort     = ":50059"
)

func main() {
	cardDB, err := carddb.Connect(cardDBDSN)
	if err != nil {
		log.Fatalf("failed to connect to card_db: %v", err)
	}
	defer cardDB.Close()

	accountDB, err := carddb.Connect(accountDBDSN)
	if err != nil {
		log.Fatalf("failed to connect to account_db: %v", err)
	}
	defer accountDB.Close()

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	pb.RegisterCardServiceServer(srv, &handlers.CardServer{
		DB:        cardDB,
		AccountDB: accountDB,
	})

	log.Printf("card-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
