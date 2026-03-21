package models

import "time"

type Card struct {
	ID                 int64
	CardNumber         string
	CardType           string
	CardName           string
	CreatedAt          time.Time
	ExpiryDate         time.Time
	AccountNumber      string
	CVV                string
	CardLimit          *float64
	Status             string
	AuthorizedPersonID *int64
}
