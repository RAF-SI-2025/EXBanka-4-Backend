package utils

import "testing"

func TestDetectCardName(t *testing.T) {
	cases := []struct {
		number string
		want   string
	}{
		{"4111111111111111", "VISA"},
		{"4532015112830366", "VISA"},
		{"5105105105105100", "MASTERCARD"}, // prefix 51
		{"5425233430109903", "MASTERCARD"}, // prefix 54
		{"2221000000000009", "MASTERCARD"}, // prefix 2221 (lower bound)
		{"2720000000000005", "MASTERCARD"}, // prefix 2720 (upper bound)
		{"9891000000000001", "DINACARD"},
		{"371449635398431", "AMERICAN_EXPRESS"},  // prefix 37, 15 digits
		{"341234567890123", "AMERICAN_EXPRESS"},  // prefix 34, 15 digits
		{"9999999999999999", ""},                 // unrecognized
		{"1234567890123456", ""},                 // unrecognized prefix
	}
	for _, c := range cases {
		got := DetectCardName(c.number)
		if got != c.want {
			t.Errorf("DetectCardName(%q) = %q, want %q", c.number, got, c.want)
		}
	}
}
