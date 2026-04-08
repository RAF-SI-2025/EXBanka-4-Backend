package models

import "time"

type OrderPortion struct {
	ID       int64
	OrderID  int64
	Quantity int32
	Price    float64
	FilledAt time.Time
}
