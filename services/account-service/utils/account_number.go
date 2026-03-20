package utils

import (
	"fmt"
	"math/rand"
)

// GenerateAccountNumber generates a unique 18-digit account number.
// Structure: bankCode (3) + branchCode (4) + randomNumber (9) + accountTypeCode (2)
// Validation: sum of all digits % 11 == 0
func GenerateAccountNumber(bankCode, branchCode, accountTypeCode string) string {
	for {
		randomNumber := fmt.Sprintf("%09d", rand.Intn(1_000_000_000))
		number := bankCode + branchCode + randomNumber + accountTypeCode

		if len(number) == 18 && digitSum(number)%11 == 0 {
			return number
		}
	}
}

func digitSum(s string) int {
	sum := 0
	for _, ch := range s {
		sum += int(ch - '0')
	}
	return sum
}
