package execution

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

// CalculatePrice returns the price per unit and whether the order can execute now.
// ask and bid are current market prices for the listing.
// limitValue and stopValue are 0 if not set (caller should pass dereferenced pointer values).
func CalculatePrice(orderType, direction string, ask, bid, limitValue, stopValue float64) (pricePerUnit float64, canExecute bool) {
	switch orderType {
	case "MARKET":
		if direction == "BUY" {
			return ask, true
		}
		return bid, true

	case "LIMIT":
		if direction == "BUY" {
			if ask <= limitValue {
				return math.Min(limitValue, ask), true
			}
			return 0, false
		}
		// SELL
		if bid >= limitValue {
			return math.Max(limitValue, bid), true
		}
		return 0, false

	case "STOP":
		if direction == "BUY" {
			if ask > stopValue {
				return ask, true
			}
			return 0, false
		}
		// SELL
		if bid < stopValue {
			return bid, true
		}
		return 0, false

	case "STOP_LIMIT":
		if direction == "BUY" {
			if ask >= stopValue {
				return ask, true
			}
			return 0, false
		}
		// SELL
		if bid < stopValue {
			return bid, true
		}
		return 0, false
	}

	return 0, false
}

// CalculateCommission returns the commission for an order given its type and total price.
func CalculateCommission(orderType string, totalPrice float64) float64 {
	switch orderType {
	case "MARKET", "STOP":
		return math.Min(0.14*totalPrice, 7.0)
	case "LIMIT", "STOP_LIMIT":
		return math.Min(0.24*totalPrice, 12.0)
	}
	return 0
}

// ApproximatePrice returns the estimated total price before order confirmation.
func ApproximatePrice(contractSize int32, pricePerUnit float64, quantity int32) float64 {
	return float64(contractSize) * pricePerUnit * float64(quantity)
}

// FillInterval returns how long to wait before the next partial fill.
// volume is the daily traded volume, remainingQuantity is what is left on the order.
// If afterHours is true, an extra 30 minutes is added.
func FillInterval(volume int64, remainingQuantity int32, afterHours bool) time.Duration {
	if volume <= 0 || remainingQuantity <= 0 {
		return 5 * time.Second
	}

	fillsPerDay := volume / int64(remainingQuantity)
	if fillsPerDay <= 0 {
		fillsPerDay = 1
	}

	// timeInterval = Random(0, 24*60 / fillsPerDay) seconds
	maxSeconds := int64(24*60) / fillsPerDay
	if maxSeconds <= 0 {
		maxSeconds = 1
	}

	seconds := rand.Int64N(maxSeconds + 1)
	d := time.Duration(seconds) * time.Second

	if afterHours {
		d += 30 * time.Minute
	}

	return d
}

// IsAfterHours reports whether now is within 4 hours before closeTime in the given timezone.
// closeTime is expected in "HH:MM" format, timezone is an IANA timezone name (e.g. "America/New_York").
func IsAfterHours(closeTime string, timezone string, now time.Time) bool {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return false
	}

	local := now.In(loc)

	var closeH, closeM int
	if _, err := parseHHMM(closeTime, &closeH, &closeM); err != nil {
		return false
	}

	close := time.Date(local.Year(), local.Month(), local.Day(), closeH, closeM, 0, 0, loc)
	fourHoursBefore := close.Add(-4 * time.Hour)

	return local.After(fourHoursBefore) && local.Before(close)
}

// ValidateMargin returns true if the margin requirement is met.
// Either loanAmount or accountBalance must exceed initialMarginCost.
func ValidateMargin(initialMarginCost, loanAmount, accountBalance float64) bool {
	return loanAmount > initialMarginCost || accountBalance > initialMarginCost
}

func parseHHMM(s string, h, m *int) (int, error) {
	n, err := fmt.Sscanf(s, "%d:%d", h, m)
	return n, err
}
