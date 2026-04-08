package approval

import "time"

// NeedsApproval reports whether an agent order requires supervisor approval.
// orderTotal is contractSize × pricePerUnit × quantity.
func NeedsApproval(needApprovalFlag bool, usedLimit, limitAmount, orderTotal float64) bool {
	if needApprovalFlag {
		return true
	}
	if usedLimit >= limitAmount {
		return true
	}
	if orderTotal > (limitAmount - usedLimit) {
		return true
	}
	return false
}

// DetermineInitialStatus returns "APPROVED" or "PENDING" for a newly created order.
// isActuary is true when the employee has a row in actuary_info (i.e. is an agent).
// Supervisors and clients are always auto-approved.
func DetermineInitialStatus(userType string, isActuary bool, needApproval bool) string {
	if userType == "CLIENT" {
		return "APPROVED"
	}
	// EMPLOYEE
	if !isActuary {
		return "APPROVED" // supervisor — no approval needed
	}
	if !needApproval {
		return "APPROVED" // agent with auto-approve
	}
	return "PENDING"
}

// IsSettlementExpired reports whether a settlement date string ("YYYY-MM-DD") is in the past.
// Returns false for empty strings (listing has no settlement date).
func IsSettlementExpired(settlementDate string) bool {
	if settlementDate == "" {
		return false
	}
	t, err := time.Parse("2006-01-02", settlementDate)
	if err != nil {
		return false
	}
	return t.Before(time.Now().Truncate(24 * time.Hour))
}
