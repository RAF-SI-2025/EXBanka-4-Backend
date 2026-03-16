package models

import "time"

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
