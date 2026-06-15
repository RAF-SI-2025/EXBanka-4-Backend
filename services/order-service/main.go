package main

import (
	"log"
	"net"
	"os"
	"time"

	orderdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/execution"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/order-service/handlers"
	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_exchange "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/exchange"
	pb_fund "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/fund"
	pb_loan "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/loan"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/order"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb_sec "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const grpcPort = ":50061"

func main() {
	orderDB, err := orderdb.Connect(os.Getenv("ORDER_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to order_db: %v", err)
	}
	defer func() { _ = orderDB.Close() }()

	accountDB, err := orderdb.Connect(os.Getenv("ACCOUNT_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to account_db: %v", err)
	}
	defer func() { _ = accountDB.Close() }()

	securitiesDB, err := orderdb.Connect(os.Getenv("SECURITIES_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to securities_db: %v", err)
	}
	defer func() { _ = securitiesDB.Close() }()

	exchangeDB, err := orderdb.Connect(os.Getenv("EXCHANGE_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to exchange_db: %v", err)
	}
	defer func() { _ = exchangeDB.Close() }()

	employeeDB, err := orderdb.Connect(os.Getenv("EMPLOYEE_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to employee_db: %v", err)
	}
	defer func() { _ = employeeDB.Close() }()

	secConn, err := grpc.NewClient(os.Getenv("SECURITIES_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to securities-service: %v", err)
	}
	defer func() { _ = secConn.Close() }()

	loanConn, err := grpc.NewClient(os.Getenv("LOAN_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to loan-service: %v", err)
	}
	defer func() { _ = loanConn.Close() }()

	empConn, err := grpc.NewClient(os.Getenv("EMPLOYEE_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to employee-service: %v", err)
	}
	defer func() { _ = empConn.Close() }()

	exchangeConn, err := grpc.NewClient(os.Getenv("EXCHANGE_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to exchange-service: %v", err)
	}
	defer func() { _ = exchangeConn.Close() }()

	portfolioConn, err := grpc.NewClient(os.Getenv("PORTFOLIO_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to portfolio-service: %v", err)
	}
	defer func() { _ = portfolioConn.Close() }()

	fundConn, err := grpc.NewClient(os.Getenv("FUND_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to fund-service: %v", err)
	}
	defer func() { _ = fundConn.Close() }()

	clientConn, err := grpc.NewClient(os.Getenv("CLIENT_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to client-service: %v", err)
	}
	defer func() { _ = clientConn.Close() }()

	securitiesClient := pb_sec.NewSecuritiesServiceClient(secConn)
	loanClient := pb_loan.NewLoanServiceClient(loanConn)
	employeeClient := pb_emp.NewEmployeeServiceClient(empConn)
	exchangeClient := pb_exchange.NewExchangeServiceClient(exchangeConn)
	portfolioClient := pb_portfolio.NewPortfolioServiceClient(portfolioConn)
	fundClient := pb_fund.NewFundServiceClient(fundConn)
	clientClient := pb_client.NewClientServiceClient(clientConn)

	// RabbitMQ — optional; notifications disabled if URL not set
	var amqpChannel *amqp.Channel
	if rabbitmqURL := os.Getenv("RABBITMQ_URL"); rabbitmqURL != "" {
		var rabbitConn *amqp.Connection
		for i := 1; i <= 10; i++ {
			rabbitConn, err = amqp.Dial(rabbitmqURL)
			if err == nil {
				break
			}
			log.Printf("order-service: RabbitMQ not ready (attempt %d/10): %v", i, err)
			time.Sleep(2 * time.Second)
		}
		if rabbitConn != nil {
			defer func() { _ = rabbitConn.Close() }()
			amqpChannel, err = rabbitConn.Channel()
			if err != nil {
				log.Printf("order-service: failed to open amqp channel: %v", err)
			} else {
				defer func() { _ = amqpChannel.Close() }()
				if _, err := amqpChannel.QueueDeclare("email.orderstatus", true, false, false, false, nil); err != nil {
					log.Printf("order-service: queue declare: %v", err)
				}
			}
		}
	}

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
		PortfolioClient:  portfolioClient,
		ClientClient:     clientClient,
		AmqpChannel:      amqpChannel,
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
		ExchangeClient:   exchangeClient,
		FundClient:       fundClient,
		AmqpChannel:      amqpChannel,
		ClientClient:     clientClient,
	}
	scheduler.Start()

	recurringScheduler := &execution.RecurringScheduler{
		DB:               orderDB,
		AccountDB:        accountDB,
		SecuritiesDB:     securitiesDB,
		EmployeeDB:       employeeDB,
		SecuritiesClient: securitiesClient,
	}
	recurringScheduler.Start()

	log.Printf("order-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
