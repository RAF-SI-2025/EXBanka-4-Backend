package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	amqp "github.com/rabbitmq/amqp091-go"
)

const priceAlertQueueName = "email.pricealert"

type priceAlertNotif struct {
	Email        string  `json:"email"`
	FirstName    string  `json:"first_name"`
	Ticker       string  `json:"ticker"`
	Condition    string  `json:"condition"`
	Threshold    float64 `json:"threshold"`
	CurrentPrice float64 `json:"current_price"`
}

func checkPriceAlerts(db *sql.DB, listingID int64, newPrice, changePercent float64,
	amqpCh *amqp.Channel, empClient pb_emp.EmployeeServiceClient, cliClient pb_client.ClientServiceClient) {

	ctx := context.Background()

	var ticker string
	_ = db.QueryRowContext(ctx, `SELECT ticker FROM listing WHERE id = $1`, listingID).Scan(&ticker)

	rows, err := db.QueryContext(ctx, `
		SELECT id, condition, threshold, user_id, user_type, notification_type
		FROM price_alerts
		WHERE listing_id = $1 AND is_active = true`,
		listingID,
	)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id, userID int64
		var condition, userType, notifType string
		var threshold float64
		if err := rows.Scan(&id, &condition, &threshold, &userID, &userType, &notifType); err != nil {
			continue
		}

		triggered := false
		switch condition {
		case "ABOVE":
			triggered = newPrice >= threshold
		case "BELOW":
			triggered = newPrice <= threshold
		case "CHANGE_PCT_UP":
			triggered = changePercent >= threshold
		case "CHANGE_PCT_DOWN":
			triggered = changePercent <= -threshold
		}
		if !triggered {
			continue
		}

		_, _ = db.ExecContext(ctx, `
			UPDATE price_alerts SET is_active = false, triggered_at = NOW()
			WHERE id = $1`, id)

		if amqpCh == nil || (notifType != "EMAIL" && notifType != "BOTH") {
			continue
		}

		var email, firstName string
		if userType == "CLIENT" && cliClient != nil {
			if resp, err := cliClient.GetClientById(ctx, &pb_client.GetClientByIdRequest{Id: userID}); err == nil && resp.Client != nil {
				email, firstName = resp.Client.Email, resp.Client.FirstName
			}
		} else if empClient != nil {
			if resp, err := empClient.GetEmployeeById(ctx, &pb_emp.GetEmployeeByIdRequest{Id: userID}); err == nil && resp.Employee != nil {
				email, firstName = resp.Employee.Email, resp.Employee.FirstName
			}
		}
		if email == "" {
			continue
		}

		body, err := json.Marshal(priceAlertNotif{
			Email:        email,
			FirstName:    firstName,
			Ticker:       ticker,
			Condition:    condition,
			Threshold:    threshold,
			CurrentPrice: newPrice,
		})
		if err != nil {
			log.Printf("price_alerts: marshal notif for alert %d: %v", id, err)
			continue
		}
		if err := amqpCh.Publish("", priceAlertQueueName, false, false, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		}); err != nil {
			log.Printf("price_alerts: publish notif for alert %d: %v", id, err)
		}
	}
}
