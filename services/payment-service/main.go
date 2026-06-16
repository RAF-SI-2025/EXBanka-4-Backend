package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pmdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/payment-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/payment-service/handlers"
	pb_auth "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	pb_email "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/payment"
)

func main() {
	db, err := pmdb.Connect(os.Getenv("PAYMENT_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to payment_db: %v", err)
	}
	defer func() { _ = db.Close() }()

	accountDB, err := pmdb.Connect(os.Getenv("ACCOUNT_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to account_db: %v", err)
	}
	defer func() { _ = accountDB.Close() }()

	exchangeDB, err := pmdb.Connect(os.Getenv("EXCHANGE_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to exchange_db: %v", err)
	}
	defer func() { _ = exchangeDB.Close() }()

	clientDB, err := pmdb.Connect(os.Getenv("CLIENT_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to client_db: %v", err)
	}
	defer func() { _ = clientDB.Close() }()

	emailConn, err := grpc.NewClient(os.Getenv("EMAIL_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to email-service: %v", err)
	}
	defer func() { _ = emailConn.Close() }()

	authConn, err := grpc.NewClient(os.Getenv("AUTH_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to auth-service: %v", err)
	}
	defer func() { _ = authConn.Close() }()

	lis, err := net.Listen("tcp", ":50055")
	if err != nil {
		log.Fatalf("failed to listen on :50055: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterPaymentServiceServer(s, &handlers.PaymentServer{
		DB:          db,
		AccountDB:   accountDB,
		ExchangeDB:  exchangeDB,
		ClientDB:    clientDB,
		EmailClient: pb_email.NewEmailServiceClient(emailConn),
		AuthClient:  pb_auth.NewAuthServiceClient(authConn),
	})

	log.Println("payment-service gRPC server listening on :50055")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
