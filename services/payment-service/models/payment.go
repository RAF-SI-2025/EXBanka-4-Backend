package models

import "time"

type PaymentRecipient struct {
	ID            int64
	ClientID      int64
	Name          string
	AccountNumber string
}

type Payment struct {
	ID              int64
	OrderNumber     string
	FromAccount     string
	ToAccount       string
	InitialAmount   float64
	FinalAmount     float64
	Fee             float64
	RecipientID     *int64
	PaymentCode     string
	ReferenceNumber string
	Purpose         string
	Timestamp       time.Time
	Status          string
}
