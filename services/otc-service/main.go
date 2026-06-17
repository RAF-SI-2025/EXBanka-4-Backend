package main

import (
	"log"
	"net"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	otcdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/otc-service/db"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/otc-service/handlers"
	pb_auth "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/auth"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/otc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	authConn, err := grpc.NewClient(os.Getenv("AUTH_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to auth-service: %v", err)
	}
	defer func() { _ = authConn.Close() }()

	var amqpCh *amqp.Channel
	if amqpURL := os.Getenv("RABBITMQ_URL"); amqpURL != "" {
		var amqpConn *amqp.Connection
		for i := 0; i < 10; i++ {
			amqpConn, err = amqp.Dial(amqpURL)
			if err == nil {
				break
			}
			log.Printf("AMQP connect attempt %d failed: %v", i+1, err)
			time.Sleep(2 * time.Second)
		}
		if amqpConn != nil {
			defer func() { _ = amqpConn.Close() }()
			if ch, chErr := amqpConn.Channel(); chErr == nil {
				defer func() { _ = ch.Close() }()
				for _, q := range []string{"email.otc.counteroffer", "email.otc.statuschange", "email.otc.expiry"} {
					_, _ = ch.QueueDeclare(q, true, false, false, false, nil)
				}
				amqpCh = ch
			}
		}
	}

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
		AmqpChannel:  amqpCh,
		AuthClient:   pb_auth.NewAuthServiceClient(authConn),
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

	// Daily expiry warning emails at 09:00.
	go func() {
		for {
			now := time.Now()
			next09 := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
			if !now.Before(next09) {
				next09 = next09.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next09))
			srvImpl.SendExpiryWarnings()
		}
	}()

	log.Printf("otc-service gRPC server listening on %s", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
