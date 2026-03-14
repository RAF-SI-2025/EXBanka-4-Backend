package main

import (
	"log"

	acdb "github.com/exbanka/backend/services/account-service/db"
)

func main() {
	database, err := acdb.Connect("postgres://account_user:account_pass@localhost:5436/account_db?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("account-service started")
	select {}
}
