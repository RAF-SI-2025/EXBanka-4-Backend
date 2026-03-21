package utils

import "fmt"

// ValidateCardNumber returns true if the card number passes the Luhn check.
func ValidateCardNumber(cardNumber string) bool {
	sum := 0
	double := false
	for i := len(cardNumber) - 1; i >= 0; i-- {
		d := int(cardNumber[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

// GenerateCheckDigit takes the partial card number (all digits except the last)
// and returns the single check digit that makes the full number Luhn-valid.
func GenerateCheckDigit(partial string) string {
	// Append a 0 placeholder and run Luhn; the check digit is (10 - sum%10) % 10.
	sum := 0
	double := true // the placeholder 0 is at position 0 from right, so next (partial's last) is doubled
	for i := len(partial) - 1; i >= 0; i-- {
		d := int(partial[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return fmt.Sprintf("%d", (10-sum%10)%10)
}
