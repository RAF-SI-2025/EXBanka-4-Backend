package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	pmdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/payment-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/payment-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
)

func main() {
	db, err := pmdb.Connect("postgres://payment_user:payment_pass@localhost:5437/payment_db?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to payment_db: %v", err)
	}
	defer db.Close()

	accountDB, err := pmdb.Connect("postgres://account_user:account_pass@localhost:5436/account_db?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to account_db: %v", err)
	}
	defer accountDB.Close()

	exchangeDB, err := pmdb.Connect("postgres://exchange_user:exchange_pass@localhost:5438/exchange_db?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to exchange_db: %v", err)
	}
	defer exchangeDB.Close()

	clientDB, err := pmdb.Connect("postgres://client_user:client_pass@localhost:5435/client_db?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to client_db: %v", err)
	}
	defer clientDB.Close()

	lis, err := net.Listen("tcp", ":50055")
	if err != nil {
		log.Fatalf("failed to listen on :50055: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterPaymentServiceServer(s, &handlers.PaymentServer{
		DB:         db,
		AccountDB:  accountDB,
		ExchangeDB: exchangeDB,
		ClientDB:   clientDB,
	})

	log.Println("payment-service gRPC server listening on :50055")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
