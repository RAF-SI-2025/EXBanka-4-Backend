package main

import (
	"log"
	"net"
	"os"

	portfoliodb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/portfolio-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/portfolio-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	"google.golang.org/grpc"
)

const grpcPort = ":50062"

func main() {
	db, err := portfoliodb.Connect(os.Getenv("PORTFOLIO_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to portfolio_db: %v", err)
	}
	defer db.Close()

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	pb.RegisterPortfolioServiceServer(srv, &handlers.PortfolioServer{DB: db})

	log.Printf("portfolio-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
