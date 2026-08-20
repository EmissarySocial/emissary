package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestOAuthClientSchema returns the rosetta schema that describes a TestOAuthClient
func TestOAuthClientSchema(t *testing.T) {

	client := NewOAuthClient()
	s := schema.New(OAuthClientSchema())

	table := []tableTestItem{
		{"clientId", "123456781234567812345678", nil},
		// note: "clientSecret" is intentionally NOT on the schema surface (set via direct field access only).
		{"clientUrl", "https://example.com/client", nil},
		{"name", "My App", nil},
		{"summary", "A test OAuth client", nil},
		{"website", "https://example.com", nil},
		{"redirectUris.0", "https://example.com/callback", nil},
		{"redirectUris.1", "https://example.com/callback2", nil},
		{"scopes.0", "read", nil},
		{"scopes.1", "write", nil},
		// note: "iconUrl" is a virtual/url field and is omitted from the round-trip table.
	}

	tableTest_Schema(t, &s, &client, table)
}

// TestOAuthClient_IsConfidential verifies that a client counts as confidential only when it stores a secret
func TestOAuthClient_IsConfidential(t *testing.T) {

	t.Run("client with a secret is confidential", func(t *testing.T) {
		client := OAuthClient{ClientSecret: "topsecret"}
		require.True(t, client.IsConfidential())
	})

	t.Run("client without a secret is public", func(t *testing.T) {
		client := OAuthClient{ClientSecret: ""}
		require.False(t, client.IsConfidential())
	})
}

// TestOAuthClient_ValidateSecret verifies secret checking for both confidential and public clients
func TestOAuthClient_ValidateSecret(t *testing.T) {

	// --- Confidential client (a secret is stored) ---
	confidential := OAuthClient{ClientSecret: "topsecret"}

	t.Run("correct secret is accepted", func(t *testing.T) {
		require.Nil(t, confidential.ValidateSecret("topsecret"))
	})

	t.Run("wrong secret is rejected", func(t *testing.T) {
		require.NotNil(t, confidential.ValidateSecret("guess"))
	})

	t.Run("empty secret is rejected", func(t *testing.T) {
		require.NotNil(t, confidential.ValidateSecret(""))
	})

	// --- Public client (no secret stored): the reported bypass ---
	// An empty supplied secret must NEVER satisfy an empty stored secret.
	public := OAuthClient{ClientSecret: ""}

	t.Run("public client: empty secret is rejected (the bug)", func(t *testing.T) {
		require.NotNil(t, public.ValidateSecret(""))
	})

	t.Run("public client: any secret is rejected", func(t *testing.T) {
		require.NotNil(t, public.ValidateSecret("anything"))
	})
}
