package config

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

func TestCoopSchema(t *testing.T) {

	c := NewCoop()
	s := schema.New(CoopSchema())

	table := []tableTestItem{
		{"apiKey", "COOP_API_KEY", nil},
		{"webhookPublicKey", "COOP_WEBHOOK_PUBLIC_KEY", nil},
	}

	tableTest_Schema(t, &s, &c, table)
}

func TestCoopIsNil(t *testing.T) {

	// Empty config is nil
	empty := NewCoop()
	require.True(t, empty.IsNil())

	// Configured with an API key is not nil
	configured := Coop{APIKey: "test-key"}
	require.False(t, configured.IsNil())
}
