package main

import (
	"log"
	"net"
	"os"

	orderdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/execution"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/handlers"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_loan "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const grpcPort = ":50061"

func main() {
	orderDB, err := orderdb.Connect(os.Getenv("ORDER_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to order_db: %v", err)
	}
	defer orderDB.Close()

	accountDB, err := orderdb.Connect(os.Getenv("ACCOUNT_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to account_db: %v", err)
	}
	defer accountDB.Close()

	securitiesDB, err := orderdb.Connect(os.Getenv("SECURITIES_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to securities_db: %v", err)
	}
	defer securitiesDB.Close()

	exchangeDB, err := orderdb.Connect(os.Getenv("EXCHANGE_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to exchange_db: %v", err)
	}
	defer exchangeDB.Close()

	employeeDB, err := orderdb.Connect(os.Getenv("EMPLOYEE_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to employee_db: %v", err)
	}
	defer employeeDB.Close()

	secConn, err := grpc.NewClient(os.Getenv("SECURITIES_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to securities-service: %v", err)
	}
	defer secConn.Close()

	loanConn, err := grpc.NewClient(os.Getenv("LOAN_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to loan-service: %v", err)
	}
	defer loanConn.Close()

	empConn, err := grpc.NewClient(os.Getenv("EMPLOYEE_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to employee-service: %v", err)
	}
	defer empConn.Close()

	portfolioConn, err := grpc.NewClient(os.Getenv("PORTFOLIO_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to portfolio-service: %v", err)
	}
	defer portfolioConn.Close()

	securitiesClient := pb_sec.NewSecuritiesServiceClient(secConn)
	loanClient := pb_loan.NewLoanServiceClient(loanConn)
	employeeClient := pb_emp.NewEmployeeServiceClient(empConn)
	portfolioClient := pb_portfolio.NewPortfolioServiceClient(portfolioConn)

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	orderServer := &handlers.OrderServer{
		DB:               orderDB,
		AccountDB:        accountDB,
		SecuritiesDB:     securitiesDB,
		ExchangeDB:       exchangeDB,
		EmployeeDB:       employeeDB,
		SecuritiesClient: securitiesClient,
		LoanClient:       loanClient,
		EmployeeClient:   employeeClient,
	}
	pb.RegisterOrderServiceServer(srv, orderServer)

	scheduler := &execution.Scheduler{
		DB:               orderDB,
		AccountDB:        accountDB,
		SecuritiesDB:     securitiesDB,
		ExchangeDB:       exchangeDB,
		EmployeeDB:       employeeDB,
		SecuritiesClient: securitiesClient,
		LoanClient:       loanClient,
		EmployeeClient:   employeeClient,
		PortfolioClient:  portfolioClient,
	}
	scheduler.Start()

	log.Printf("order-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
