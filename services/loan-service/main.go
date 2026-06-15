package main

import (
	"log"
	"net"
	"os"

	loandb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/loan-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/loan-service/handlers"
	pb_auth "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_email "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const grpcPort = ":50058"

func main() {
	loanDB, err := loandb.Connect(os.Getenv("LOAN_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to loan_db: %v", err)
	}
	defer func() { _ = loanDB.Close() }()

	accountDB, err := loandb.Connect(os.Getenv("ACCOUNT_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to account_db: %v", err)
	}
	defer func() { _ = accountDB.Close() }()

	exchangeDB, err := loandb.Connect(os.Getenv("EXCHANGE_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to exchange_db: %v", err)
	}
	defer func() { _ = exchangeDB.Close() }()

	emailConn, err := grpc.NewClient(os.Getenv("EMAIL_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to email-service: %v", err)
	}
	defer func() { _ = emailConn.Close() }()
	emailClient := pb_email.NewEmailServiceClient(emailConn)

	clientConn, err := grpc.NewClient(os.Getenv("CLIENT_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to client-service: %v", err)
	}
	defer func() { _ = clientConn.Close() }()
	clientClient := pb_client.NewClientServiceClient(clientConn)

	authConn, err := grpc.NewClient(os.Getenv("AUTH_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to auth-service: %v", err)
	}
	defer func() { _ = authConn.Close() }()
	authClient := pb_auth.NewAuthServiceClient(authConn)

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	loanServer := &handlers.LoanServer{
		DB:           loanDB,
		AccountDB:    accountDB,
		ExchangeDB:   exchangeDB,
		EmailClient:  emailClient,
		ClientClient: clientClient,
		AuthClient:   authClient,
	}
	pb.RegisterLoanServiceServer(srv, loanServer)

	// Start cron jobs
	loanServer.StartCronJobs()

	log.Printf("loan-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
