package queue

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

const OrderStatusQueueName = "email.orderstatus"

type OrderStatusMessage struct {
	Email        string  `json:"email"`
	FirstName    string  `json:"first_name"`
	OrderID      int64   `json:"order_id"`
	Ticker       string  `json:"ticker"`
	Direction    string  `json:"direction"`
	Status       string  `json:"status"`
	Quantity     int32   `json:"quantity"`
	FilledQty    int32   `json:"filled_qty"`
	PricePerUnit float64 `json:"price_per_unit"`
}

func PublishOrderStatus(ch *amqp.Channel, msg OrderStatusMessage) error {
	if ch == nil {
		return nil
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return ch.Publish("", OrderStatusQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}
