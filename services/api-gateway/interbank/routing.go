package interbank

import (
	"fmt"
	"os"
)

type BankInfo struct {
	RoutingNumber string
	BankName      string
	BankURL       string
	APIKey        string
}

func ExtractRoutingNumber(accountNumber string) string {
	if len(accountNumber) < 3 {
		return ""
	}
	return accountNumber[:3]
}

func IsOwnBank(routingNumber string) bool {
	return routingNumber == os.Getenv("OWN_ROUTING_NUMBER")
}

func ResolveBankByRoutingNumber(routingNumber string) (BankInfo, error) {
	own := os.Getenv("OWN_ROUTING_NUMBER")
	partner := os.Getenv("PARTNER_ROUTING_NUMBER")

	switch routingNumber {
	case own:
		return BankInfo{RoutingNumber: own}, nil
	case partner:
		return BankInfo{
			RoutingNumber: routingNumber,
			BankName:      os.Getenv("PARTNER_BANK_NAME"),
			BankURL:       os.Getenv("PARTNER_BANK_URL"),
			APIKey:        os.Getenv("PARTNER_API_KEY"),
		}, nil
	default:
		return BankInfo{}, fmt.Errorf("unknown routing number: %s", routingNumber)
	}
}
