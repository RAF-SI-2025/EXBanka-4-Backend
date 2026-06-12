package main

import (
	"log"
	"net"
	"os"
	"time"

	otcdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/otc-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/otc-service/handlers"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"google.golang.org/grpc"
)

const grpcPort = ":50063"

func main() {
	otcDB, err := otcdb.Connect(os.Getenv("OTC_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to otc_db: %v", err)
	}
	defer func() { _ = otcDB.Close() }()

	employeeDB, err := otcdb.Connect(os.Getenv("EMPLOYEE_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to employee_db: %v", err)
	}
	defer func() { _ = employeeDB.Close() }()

	clientDB, err := otcdb.Connect(os.Getenv("CLIENT_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to client_db: %v", err)
	}
	defer func() { _ = clientDB.Close() }()

	accountDB, err := otcdb.Connect(os.Getenv("ACCOUNT_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to account_db: %v", err)
	}
	defer func() { _ = accountDB.Close() }()

	portfolioDB, err := otcdb.Connect(os.Getenv("PORTFOLIO_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to portfolio_db: %v", err)
	}
	defer func() { _ = portfolioDB.Close() }()

	securitiesDB, err := otcdb.Connect(os.Getenv("SECURITIES_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to securities_db: %v", err)
	}
	defer func() { _ = securitiesDB.Close() }()

	exchangeDB, err := otcdb.Connect(os.Getenv("EXCHANGE_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to exchange_db: %v", err)
	}
	defer func() { _ = exchangeDB.Close() }()

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	srvImpl := &handlers.OtcServer{
		DB:           otcDB,
		EmployeeDB:   employeeDB,
		ClientDB:     clientDB,
		AccountDB:    accountDB,
		PortfolioDB:  portfolioDB,
		SecuritiesDB: securitiesDB,
		ExchangeDB:   exchangeDB,
	}
	pb.RegisterOtcServiceServer(srv, srvImpl)

	// Recover any SAGA flows that were interrupted by a previous crash.
	go srvImpl.RecoverInFlightSagas()

	// Hourly contract expiration with tax loss recording.
	// Runs immediately on startup so stale contracts are cleaned up at boot.
	go func() {
		srvImpl.ExpireContracts()
		for range time.Tick(time.Hour) {
			srvImpl.ExpireContracts()
		}
	}()

	log.Printf("otc-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
