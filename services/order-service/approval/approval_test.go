package approval

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- NeedsApproval ---

func TestNeedsApproval_FlagTrue(t *testing.T) {
	assert.True(t, NeedsApproval(true, 0, 10000, 500))
}

func TestNeedsApproval_UsedLimitExceeded(t *testing.T) {
	assert.True(t, NeedsApproval(false, 10000, 10000, 1))
}

func TestNeedsApproval_UsedLimitOverLimit(t *testing.T) {
	assert.True(t, NeedsApproval(false, 11000, 10000, 1))
}

func TestNeedsApproval_OrderExceedsRemaining(t *testing.T) {
	// usedLimit=8000, limit=10000, remaining=2000, orderTotal=3000 → exceeds
	assert.True(t, NeedsApproval(false, 8000, 10000, 3000))
}

func TestNeedsApproval_AllClear(t *testing.T) {
	// usedLimit=0, limit=10000, orderTotal=500 → no approval needed
	assert.False(t, NeedsApproval(false, 0, 10000, 500))
}

func TestNeedsApproval_ExactlyAtRemainingLimit(t *testing.T) {
	// orderTotal == remaining limit → does not exceed → no approval
	assert.False(t, NeedsApproval(false, 5000, 10000, 5000))
}

// --- DetermineInitialStatus ---

func TestDetermineInitialStatus_Client(t *testing.T) {
	assert.Equal(t, "APPROVED", DetermineInitialStatus("CLIENT", false, false))
	assert.Equal(t, "APPROVED", DetermineInitialStatus("CLIENT", true, true))
}

func TestDetermineInitialStatus_EmployeeNotActuary(t *testing.T) {
	// Supervisor — no actuary_info row
	assert.Equal(t, "APPROVED", DetermineInitialStatus("EMPLOYEE", false, false))
	assert.Equal(t, "APPROVED", DetermineInitialStatus("EMPLOYEE", false, true))
}

func TestDetermineInitialStatus_AgentNoApproval(t *testing.T) {
	assert.Equal(t, "APPROVED", DetermineInitialStatus("EMPLOYEE", true, false))
}

func TestDetermineInitialStatus_AgentNeedsApproval(t *testing.T) {
	assert.Equal(t, "PENDING", DetermineInitialStatus("EMPLOYEE", true, true))
}

// --- IsSettlementExpired ---

func TestIsSettlementExpired_PastDate(t *testing.T) {
	assert.True(t, IsSettlementExpired("2020-01-01"))
}

func TestIsSettlementExpired_FutureDate(t *testing.T) {
	future := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	assert.False(t, IsSettlementExpired(future))
}

func TestIsSettlementExpired_EmptyString(t *testing.T) {
	assert.False(t, IsSettlementExpired(""))
}

func TestIsSettlementExpired_InvalidFormat(t *testing.T) {
	assert.False(t, IsSettlementExpired("not-a-date"))
}

func TestIsSettlementExpired_Today(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	// Today truncated to midnight is not before today truncated — should be false
	assert.False(t, IsSettlementExpired(today))
}
