package utils

import (
	"testing"
)

func TestValidateCardNumber_KnownValid(t *testing.T) {
	valid := []string{
		"4532015112830366", // Visa
		"5425233430109903", // Mastercard
		"378282246310005",  // AmEx (15 digits)
		"6011111111111117", // Luhn-valid (any brand)
	}
	for _, n := range valid {
		if !ValidateCardNumber(n) {
			t.Errorf("ValidateCardNumber(%q) = false, want true", n)
		}
	}
}

func TestValidateCardNumber_Tampered(t *testing.T) {
	// Flip last digit of each known-valid number — should fail
	tampered := []string{
		"4532015112830367",
		"5425233430109904",
		"378282246310006",
	}
	for _, n := range tampered {
		if ValidateCardNumber(n) {
			t.Errorf("ValidateCardNumber(%q) = true, want false (tampered)", n)
		}
	}
}

func TestGenerateCheckDigit_ProducesValidNumber(t *testing.T) {
	partials := []string{
		"453201511283036", // Visa partial (15 digits)
		"542523343010990", // Mastercard partial (15 digits)
		"37828224631000",  // AmEx partial (14 digits)
	}
	for _, p := range partials {
		full := p + GenerateCheckDigit(p)
		if !ValidateCardNumber(full) {
			t.Errorf("partial %q + check digit = %q, but ValidateCardNumber = false", p, full)
		}
	}
}
