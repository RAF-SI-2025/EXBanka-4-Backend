package utils

import (
	"strings"
	"testing"
)

func TestGenerateCardNumber_LuhnValid(t *testing.T) {
	for _, brand := range []string{"VISA", "MASTERCARD", "DINACARD", "AMERICAN_EXPRESS"} {
		n := GenerateCardNumber(brand, "00001")
		if !ValidateCardNumber(n) {
			t.Errorf("GenerateCardNumber(%q) = %q, fails Luhn check", brand, n)
		}
	}
}

func TestGenerateCardNumber_Length(t *testing.T) {
	cases := []struct {
		brand  string
		length int
	}{
		{"VISA", 16},
		{"MASTERCARD", 16},
		{"DINACARD", 16},
		{"AMERICAN_EXPRESS", 15},
	}
	for _, c := range cases {
		n := GenerateCardNumber(c.brand, "00001")
		if len(n) != c.length {
			t.Errorf("GenerateCardNumber(%q) length = %d, want %d (number: %q)", c.brand, len(n), c.length, n)
		}
	}
}

func TestGenerateCardNumber_MII(t *testing.T) {
	cases := []struct {
		brand  string
		prefix string
	}{
		{"VISA", "4"},
		{"MASTERCARD", "5"},
		{"DINACARD", "9"},
		{"AMERICAN_EXPRESS", "3"},
	}
	for _, c := range cases {
		n := GenerateCardNumber(c.brand, "00001")
		if !strings.HasPrefix(n, c.prefix) {
			t.Errorf("GenerateCardNumber(%q) = %q, want prefix %q", c.brand, n, c.prefix)
		}
	}
}

func TestGenerateCardNumber_BrandDetection(t *testing.T) {
	cases := []struct {
		brand   string
		iinCode string // must be brand-compatible
	}{
		{"VISA", "00001"},   // MII "4" + IIN "00001" → prefix "400001"
		{"AMERICAN_EXPRESS", "40001"}, // MII "3" + IIN "40001" → prefix "340001"
	}
	for _, c := range cases {
		n := GenerateCardNumber(c.brand, c.iinCode)
		got := DetectCardName(n)
		if got != c.brand {
			t.Errorf("DetectCardName(GenerateCardNumber(%q, %q)) = %q, want %q (number: %q)", c.brand, c.iinCode, got, c.brand, n)
		}
	}
}

func TestGenerateCardNumber_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		n := GenerateCardNumber("VISA", "00001")
		seen[n] = true
	}
	if len(seen) < 2 {
		t.Error("GenerateCardNumber produced identical numbers across 10 calls")
	}
}
