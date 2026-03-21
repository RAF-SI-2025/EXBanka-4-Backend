package models

import "time"

type AuthorizedPerson struct {
	ID            int64
	FirstName     string
	LastName      string
	DateOfBirth   time.Time
	Gender        string
	Email         string
	PhoneNumber   string
	Address       string
	AccountNumber string
}
