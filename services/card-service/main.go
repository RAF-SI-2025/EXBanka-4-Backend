package main

import (
	"log"
	"net"
	"os"

	carddb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/card-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/card-service/handlers"
	pb_auth "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/card"
	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_email "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const grpcPort = ":50059"

func main() {
	cardDB, err := carddb.Connect(os.Getenv("CARD_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to card_db: %v", err)
	}
	defer func() {
		if err := cardDB.Close(); err != nil {
			log.Printf("card_db close: %v", err)
		}
	}()

	accountDB, err := carddb.Connect(os.Getenv("ACCOUNT_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to account_db: %v", err)
	}
	defer func() {
		if err := accountDB.Close(); err != nil {
			log.Printf("account_db close: %v", err)
		}
	}()

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

	clientConn, err := grpc.NewClient(os.Getenv("CLIENT_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to client-service: %v", err)
	}
	defer func() { _ = clientConn.Close() }()

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	pb.RegisterCardServiceServer(srv, &handlers.CardServer{
		DB:           cardDB,
		AccountDB:    accountDB,
		EmailClient:  pb_email.NewEmailServiceClient(emailConn),
		AuthClient:   pb_auth.NewAuthServiceClient(authConn),
		ClientClient: pb_client.NewClientServiceClient(clientConn),
	})

	log.Printf("card-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
