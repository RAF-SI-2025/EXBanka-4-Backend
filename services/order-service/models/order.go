package models

import "time"

type Order struct {
	ID                int64
	UserID            int64
	UserType          string // CLIENT or EMPLOYEE
	AssetID           int64
	OrderType         string
	Quantity          int32
	ContractSize      int32
	PricePerUnit      float64
	LimitValue        *float64
	StopValue         *float64
	Direction         string
	Status            string
	ApprovedBy        *int64
	IsDone            bool
	LastModification  time.Time
	RemainingPortions int32
	AfterHours        bool
	IsAON             bool
	IsMargin          bool
	AccountID         int64
}
