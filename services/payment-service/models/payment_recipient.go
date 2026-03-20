package models

type PaymentRecipient struct {
	ID            int64
	ClientID      int64
	Name          string
	AccountNumber string
}
