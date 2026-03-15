package models

import "time"

type Account struct {
	ID               int64
	AccountNumber    string
	AccountName      string
	OwnerID          int64
	EmployeeID       int64
	CurrencyID       int64
	AccountType      string
	Status           string
	Balance          float64
	AvailableBalance float64
	CreatedDate      time.Time
	ExpirationDate   *time.Time
	DailyLimit       *float64
	MonthlyLimit     *float64
	DailySpent       float64
	MonthlySpent     float64
	MaintenanceFee   float64
	CompanyID        *int64
}
