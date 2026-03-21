package utils

import "strconv"

// DetectCardName returns the card brand for the given card number.
// Returns "VISA", "MASTERCARD", "DINACARD", "AMERICAN_EXPRESS", or "" if unrecognized.
func DetectCardName(cardNumber string) string {
	n := len(cardNumber)

	// American Express: 15 digits, prefix 34 or 37
	if n == 15 && len(cardNumber) >= 2 {
		prefix2 := cardNumber[:2]
		if prefix2 == "34" || prefix2 == "37" {
			return "AMERICAN_EXPRESS"
		}
	}

	if n != 16 {
		return ""
	}

	// DinaCard: prefix 9891
	if len(cardNumber) >= 4 && cardNumber[:4] == "9891" {
		return "DINACARD"
	}

	// Mastercard: prefix 51–55 or 2221–2720
	if len(cardNumber) >= 2 {
		prefix2 := cardNumber[:2]
		if prefix2 >= "51" && prefix2 <= "55" {
			return "MASTERCARD"
		}
	}
	if len(cardNumber) >= 4 {
		prefix4, err := strconv.Atoi(cardNumber[:4])
		if err == nil && prefix4 >= 2221 && prefix4 <= 2720 {
			return "MASTERCARD"
		}
	}

	// Visa: prefix 4
	if cardNumber[0] == '4' {
		return "VISA"
	}

	return ""
}
