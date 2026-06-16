package main

import (
	"log"
	"net"
	"os"

	funddb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/fund-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/fund-service/handlers"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/fund-service/scheduler"
	pb_account "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/account"
	pb_exchange "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/exchange"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	pb_order "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_securities "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const grpcPort = ":50064"

func main() {
	fundDB, err := funddb.Connect(os.Getenv("FUND_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to fund_db: %v", err)
	}
	defer func() { _ = fundDB.Close() }()

	accountDB, err := funddb.Connect(os.Getenv("ACCOUNT_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to account_db: %v", err)
	}
	defer func() { _ = accountDB.Close() }()

	employeeDB, err := funddb.Connect(os.Getenv("EMPLOYEE_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to employee_db: %v", err)
	}
	defer func() { _ = employeeDB.Close() }()

	accountConn, err := grpc.NewClient(os.Getenv("ACCOUNT_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to account-service: %v", err)
	}
	defer func() { _ = accountConn.Close() }()
	accountClient := pb_account.NewAccountServiceClient(accountConn)

	orderConn, err := grpc.NewClient(os.Getenv("ORDER_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to order-service: %v", err)
	}
	defer func() { _ = orderConn.Close() }()
	orderClient := pb_order.NewOrderServiceClient(orderConn)

	exchangeDB, err := funddb.Connect(os.Getenv("EXCHANGE_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to exchange_db: %v", err)
	}
	defer func() { _ = exchangeDB.Close() }()

	exchangeConn, err := grpc.NewClient(os.Getenv("EXCHANGE_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to exchange-service: %v", err)
	}
	defer func() { _ = exchangeConn.Close() }()
	exchangeClient := pb_exchange.NewExchangeServiceClient(exchangeConn)

	securitiesConn, err := grpc.NewClient(os.Getenv("SECURITIES_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to securities-service: %v", err)
	}
	defer func() { _ = securitiesConn.Close() }()
	securitiesClient := pb_securities.NewSecuritiesServiceClient(securitiesConn)

	portfolioConn, err := grpc.NewClient(os.Getenv("PORTFOLIO_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to portfolio-service: %v", err)
	}
	defer func() { _ = portfolioConn.Close() }()
	portfolioClient := pb_portfolio.NewPortfolioServiceClient(portfolioConn)

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	pb.RegisterFundServiceServer(srv, &handlers.FundServer{
		DB:               fundDB,
		AccountDB:        accountDB,
		EmployeeDB:       employeeDB,
		ExchangeDB:       exchangeDB,
		AccountClient:    accountClient,
		OrderClient:      orderClient,
		ExchangeClient:   exchangeClient,
		SecuritiesClient: securitiesClient,
		PortfolioClient:  portfolioClient,
	})

	sched := &scheduler.PerformanceScheduler{DB: fundDB, SecuritiesClient: securitiesClient, ExchangeClient: exchangeClient}
	sched.Start()

	log.Printf("fund-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
