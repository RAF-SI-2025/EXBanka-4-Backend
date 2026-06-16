package main

import (
	"database/sql"
	_ "embed"
	"log"
	"net"
	"os"
	"time"
	_ "time/tzdata"

	secdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/securities-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/securities-service/handlers"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/securities-service/scheduler"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/securities-service/seeder"
	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	pb_ex "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/exchange"
	pb_portfolio "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/portfolio"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/securities"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

//go:embed assets/exchange_1.csv
var exchangeCSV []byte

//go:embed assets/future_data.csv
var futureDataCSV []byte

const grpcPort = ":50060"

func main() {
	securitiesDB, err := secdb.Connect(os.Getenv("SECURITIES_DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to securities_db: %v", err)
	}
	defer func() {
		if err := securitiesDB.Close(); err != nil {
			log.Printf("securities_db close: %v", err)
		}
	}()

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	var portfolioClient pb_portfolio.PortfolioServiceClient
	var exchangeClient pb_ex.ExchangeServiceClient
	var accountDB *sql.DB

	if addr := os.Getenv("PORTFOLIO_SERVICE_ADDR"); addr != "" {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("securities-service: portfolio client: %v", err)
		} else {
			portfolioClient = pb_portfolio.NewPortfolioServiceClient(conn)
		}
	}

	if addr := os.Getenv("EXCHANGE_SERVICE_ADDR"); addr != "" {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("securities-service: exchange client: %v", err)
		} else {
			exchangeClient = pb_ex.NewExchangeServiceClient(conn)
		}
	}

	if dbURL := os.Getenv("ACCOUNT_DB_URL"); dbURL != "" {
		adb, err := secdb.Connect(dbURL)
		if err != nil {
			log.Printf("securities-service: account_db: %v", err)
		} else {
			accountDB = adb
		}
	}

	// RabbitMQ — optional; price alert email notifications disabled if URL not set.
	var amqpCh *amqp.Channel
	if url := os.Getenv("RABBITMQ_URL"); url != "" {
		var rabbitConn *amqp.Connection
		for i := 0; i < 10; i++ {
			rabbitConn, err = amqp.Dial(url)
			if err == nil {
				break
			}
			time.Sleep(2 * time.Second)
		}
		if rabbitConn != nil {
			defer func() { _ = rabbitConn.Close() }()
			if ch, err := rabbitConn.Channel(); err == nil {
				defer func() { _ = ch.Close() }()
				_, _ = ch.QueueDeclare("email.pricealert", true, false, false, false, nil)
				amqpCh = ch
			}
		}
	}

	var empClient pb_emp.EmployeeServiceClient
	if addr := os.Getenv("EMPLOYEE_SERVICE_ADDR"); addr != "" {
		if conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials())); err == nil {
			empClient = pb_emp.NewEmployeeServiceClient(conn)
		}
	}
	var cliClient pb_client.ClientServiceClient
	if addr := os.Getenv("CLIENT_SERVICE_ADDR"); addr != "" {
		if conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials())); err == nil {
			cliClient = pb_client.NewClientServiceClient(conn)
		}
	}

	srv := grpc.NewServer()
	pb.RegisterSecuritiesServiceServer(srv, &handlers.SecuritiesServer{
		DB:    securitiesDB,
		Redis: redisClient,
	})

	// Seed data on startup (runs in background, does not block gRPC server).
	go seeder.Seed(securitiesDB, os.Getenv("ALPACA_API_KEY"), os.Getenv("ALPACA_API_SECRET_KEY"), os.Getenv("ALPHAVANTAGE_API_KEY"), exchangeCSV, futureDataCSV)

	// Refresh prices (interval overridable via PRICE_REFRESH_INTERVAL env for testing).
	refreshInterval := 15 * time.Minute
	if v := os.Getenv("PRICE_REFRESH_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			refreshInterval = d
		}
	}
	scheduler.StartPriceRefresh(securitiesDB, os.Getenv("ALPHAVANTAGE_API_KEY"), refreshInterval, redisClient, amqpCh, empClient, cliClient)

	// Snapshot daily prices + reset actuary limits at 23:59 every day.
	scheduler.ScheduleEOD(securitiesDB, os.Getenv("EMPLOYEE_SERVICE_ADDR"))

	// Quarterly dividend payouts.
	scheduler.StartDividendScheduler(securitiesDB, accountDB, portfolioClient, exchangeClient)

	// Simulate market price fluctuations every minute when test mode is enabled.
	scheduler.StartPriceSimulation(securitiesDB, amqpCh, empClient, cliClient)

	log.Printf("securities-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
