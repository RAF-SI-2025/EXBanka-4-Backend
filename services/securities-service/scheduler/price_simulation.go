package scheduler

import (
	"database/sql"
	"log"
	"math/rand"
	"time"

	pb_client "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/client"
	pb_emp "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	amqp "github.com/rabbitmq/amqp091-go"
)

// StartPriceSimulation launches a background goroutine that ticks every minute.
// When test mode is enabled it applies a small random price fluctuation (±1%) to
// every listing so the UI reflects a live-looking market during development.
// Price alert notifications are also sent when simulated prices cross alert thresholds.
func StartPriceSimulation(db *sql.DB, amqpCh *amqp.Channel, empClient pb_emp.EmployeeServiceClient, cliClient pb_client.ClientServiceClient) {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			enabled, err := isTestModeEnabled(db)
			if err != nil {
				log.Printf("price_simulation: test mode check: %v", err)
				continue
			}
			if !enabled {
				continue
			}
			if err := simulatePrices(db, amqpCh, empClient, cliClient); err != nil {
				log.Printf("price_simulation: %v", err)
			}
		}
	}()
	log.Println("price_simulation: started (fires every 1 minute when test mode is on)")
}

func isTestModeEnabled(db *sql.DB) (bool, error) {
	var enabled bool
	err := db.QueryRow(`SELECT test_mode_enabled FROM settings WHERE id = TRUE`).Scan(&enabled)
	return enabled, err
}

func simulatePrices(db *sql.DB, amqpCh *amqp.Channel, empClient pb_emp.EmployeeServiceClient, cliClient pb_client.ClientServiceClient) error {
	rows, err := db.Query(`SELECT id, price, ask, bid, change FROM listing`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	type row struct {
		id     int64
		price  float64
		ask    float64
		bid    float64
		change float64
	}
	var listings []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.price, &r.ask, &r.bid, &r.change); err != nil {
			return err
		}
		listings = append(listings, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, l := range listings {
		if l.price <= 0 {
			continue
		}

		// Random factor in [-0.01, +0.01] (symmetric ±1%)
		factor := 1 + (rand.Float64()*0.02 - 0.01)
		newPrice := l.price * factor

		// Preserve ask/bid spread ratio; fall back to a fixed 0.1% spread if price was zero.
		var newAsk, newBid float64
		if l.ask > 0 && l.bid > 0 {
			newAsk = newPrice * (l.ask / l.price)
			newBid = newPrice * (l.bid / l.price)
		} else {
			newAsk = newPrice * 1.001
			newBid = newPrice * 0.999
		}

		// change tracks cumulative day change from previous close (prevClose = price - change).
		prevClose := l.price - l.change
		newChange := newPrice - prevClose

		_, err := db.Exec(`
			UPDATE listing SET price=$2, ask=$3, bid=$4, change=$5, last_refresh=$6
			WHERE id=$1`,
			l.id, newPrice, newAsk, newBid, newChange, time.Now())
		if err != nil {
			log.Printf("price_simulation: update listing %d: %v", l.id, err)
			continue
		}

		var changePercent float64
		if l.price > 0 {
			changePercent = (newPrice - l.price) / l.price * 100
		}
		go checkPriceAlerts(db, l.id, newPrice, changePercent, amqpCh, empClient, cliClient)
	}

	log.Printf("price_simulation: updated %d listings", len(listings))
	return nil
}
