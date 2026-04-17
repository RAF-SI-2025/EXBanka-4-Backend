package models

import "time"

type PortfolioEntry struct {
	ID           int64
	UserID       int64
	UserType     string
	ListingID    int64
	Amount       int32
	BuyPrice     float64
	LastModified time.Time
	IsPublic     bool
	PublicAmount int32
	AccountID    int64
}
