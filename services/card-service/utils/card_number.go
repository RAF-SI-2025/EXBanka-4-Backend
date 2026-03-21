package utils

import (
	"fmt"
	"math/rand"
)

// miiByBrand returns the Major Industry Identifier digit for the given card brand.
var miiByBrand = map[string]string{
	"VISA":             "4",
	"MASTERCARD":       "5",
	"DINACARD":         "9",
	"AMERICAN_EXPRESS": "3",
}

// GenerateCardNumber generates a valid card number for the given brand and IIN code.
// iinCode must be exactly 5 digits.
// Returns 16 digits for Visa/Mastercard/DinaCard, 15 digits for AmEx.
// The Luhn check digit is computed and appended automatically.
// Note: uniqueness against the database is the caller's responsibility.
func GenerateCardNumber(cardName string, iinCode string) string {
	mii, ok := miiByBrand[cardName]
	if !ok {
		mii = "4" // default to Visa MII
	}

	accountSegmentLen := 9
	if cardName == "AMERICAN_EXPRESS" {
		accountSegmentLen = 8
	}

	accountSegment := fmt.Sprintf("%0*d", accountSegmentLen, rand.Intn(pow10(accountSegmentLen)))
	partial := mii + iinCode + accountSegment
	checkDigit := GenerateCheckDigit(partial)
	return partial + checkDigit
}

func pow10(n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= 10
	}
	return result
}
