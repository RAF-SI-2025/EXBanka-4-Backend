package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	exdb "github.com/RAF-SI-2025/EXBanka-4-Backend/services/exchange-service/db"
)

func main() {
	database, err := exdb.Connect("postgres://exchange_user:exchange_pass@localhost:5438/exchange_db?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("exchange-service started")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
}
