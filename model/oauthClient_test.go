package model

import (
	"encoding/json"
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

// TestOAuthClient_Toot verifies the Mastodon Application carries the redirect URIs and scopes that clients read back
func TestOAuthClient_Toot(t *testing.T) {

	client := NewOAuthClient()
	client.Name = "IceCubesApp"
	client.RedirectURIs = []string{"icecubesapp://", "https://example.com/callback"}
	client.Scopes = []string{"read", "write"}

	application := client.Toot()
	require.Equal(t, []string{"icecubesapp://", "https://example.com/callback"}, application.RedirectURIs)
	require.Equal(t, "icecubesapp://\nhttps://example.com/callback", application.RedirectURI)
	require.Equal(t, []string{"read", "write"}, application.Scopes)
}

// TestOAuthClient_MastodonAppJSON confirms the flag is stored under "mastodonApp" and kept off the schema
func TestOAuthClient_MastodonAppJSON(t *testing.T) {

	client := NewOAuthClient()
	client.MastodonApp = true

	encoded, err := json.Marshal(client)
	require.Nil(t, err)
	require.Contains(t, string(encoded), `"mastodonApp":true`)

	_, onSchema := client.GetPointer("mastodonApp")
	require.False(t, onSchema)
}
