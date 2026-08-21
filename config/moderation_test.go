package config

import (
	"testing"
	"time"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

func TestModerationSchema(t *testing.T) {

	m := NewModeration()
	s := schema.New(ModerationSchema())

	table := []tableTestItem{
		{"provider", "coop", nil},
		{"url", "http://coop:3000", nil},
		{"coop.apiKey", "COOP_API_KEY", nil},
		{"coop.webhookPublicKey", "COOP_WEBHOOK_PUBLIC_KEY", nil},
	}

	tableTest_Schema(t, &s, &m, table)
}

func TestModerationIsNil(t *testing.T) {

	// Empty config is nil
	empty := NewModeration()
	require.True(t, empty.IsNil())

	// Configured with a provider is not nil
	configured := Moderation{Provider: "coop"}
	require.False(t, configured.IsNil())
}

func TestModerationTestConnection_NoProvider(t *testing.T) {
	// Unconfigured moderation is a no-op success (same as SMTP)
	m := NewModeration()
	require.Nil(t, m.TestConnection(2*time.Second))
}

func TestModerationTestConnection_ProviderWithoutURL(t *testing.T) {
	// Provider selected but no URL configured
	m := Moderation{Provider: "coop", Coop: Coop{APIKey: "test-key"}}
	err := m.TestConnection(2*time.Second)
	require.NotNil(t, err)
}

func TestModerationTestConnection_CoopWithoutAPIKey(t *testing.T) {
	// Coop selected but no API key
	m := Moderation{Provider: "coop", URL: "http://coop:3000"}
	err := m.TestConnection(2*time.Second)
	require.NotNil(t, err)
}

func TestModerationTestConnection_CoopUnreachable(t *testing.T) {
	// Coop configured but backend not running — should fail fast
	m := Moderation{Provider: "coop", URL: "http://127.0.0.1:1", Coop: Coop{APIKey: "test-key"}}
	err := m.TestConnection(2*time.Second)
	require.NotNil(t, err)
}

func TestModerationTestConnection_UnknownProvider(t *testing.T) {
	m := Moderation{Provider: "unknown", URL: "http://coop:3000"}
	err := m.TestConnection(2*time.Second)
	require.NotNil(t, err)
}
