package interbank

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractRoutingNumber(t *testing.T) {
	assert.Equal(t, "", ExtractRoutingNumber(""))
	assert.Equal(t, "", ExtractRoutingNumber("12"))
	assert.Equal(t, "123", ExtractRoutingNumber("123"))
	assert.Equal(t, "888", ExtractRoutingNumber("8880012345"))
}

func TestIsOwnBank(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	assert.True(t, IsOwnBank("888"))
	assert.False(t, IsOwnBank("444"))
	assert.False(t, IsOwnBank(""))
}

func TestResolveBankByRoutingNumber_Own(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	info, err := ResolveBankByRoutingNumber("888")
	require.NoError(t, err)
	assert.Equal(t, "888", info.RoutingNumber)
	assert.Empty(t, info.BankURL)
	assert.Empty(t, info.APIKey)
}

func TestResolveBankByRoutingNumber_Partner(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")
	t.Setenv("PARTNER_BANK_NAME", "Banka 4")
	t.Setenv("PARTNER_BANK_URL", "https://banka-4.example.com")
	t.Setenv("PARTNER_API_KEY", "their-key")
	t.Setenv("OWN_INTERBANK_API_KEY", "our-key")

	info, err := ResolveBankByRoutingNumber("444")
	require.NoError(t, err)
	assert.Equal(t, "444", info.RoutingNumber)
	assert.Equal(t, "Banka 4", info.BankName)
	assert.Equal(t, "https://banka-4.example.com", info.BankURL)
	assert.Equal(t, "our-key", info.APIKey)
}

func TestResolveBankByRoutingNumber_Unknown(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "888")
	t.Setenv("PARTNER_ROUTING_NUMBER", "444")

	_, err := ResolveBankByRoutingNumber("999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "999")
}

func TestResolveBankByRoutingNumber_EmptyEnv(t *testing.T) {
	t.Setenv("OWN_ROUTING_NUMBER", "")
	t.Setenv("PARTNER_ROUTING_NUMBER", "")

	_, err := ResolveBankByRoutingNumber("888")
	require.Error(t, err)
}
