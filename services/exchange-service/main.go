package main

import (
	"log"

	exdb "github.com/exbanka/backend/services/exchange-service/db"
)

func main() {
	database, err := exdb.Connect("postgres://exchange_user:exchange_pass@localhost:5435/exchange_db?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("exchange-service started")
	select {}
}
