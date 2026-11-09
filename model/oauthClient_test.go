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

// TestOAuthClient_AllowsRedirectURI verifies that a client may redirect to what it lists, and that a URL
// client listing nothing may redirect to its own origin and nowhere else
func TestOAuthClient_AllowsRedirectURI(t *testing.T) {

	registered := NewOAuthClient()
	registered.RedirectURIs = append(registered.RedirectURIs, "https://app.example.com/callback", "urn:ietf:wg:oauth:2.0:oob")

	listed := NewOAuthClient()
	listed.ClientURL = "https://app.example.com/client.json"
	listed.RedirectURIs = append(listed.RedirectURIs, "https://app.example.com/callback")

	unlisted := NewOAuthClient()
	unlisted.ClientURL = "https://fed.brid.gy/"

	ported := NewOAuthClient()
	ported.ClientURL = "http://localhost:8080/client.json"

	allows := func(client OAuthClient, redirectURI string, expected bool, reason string) {
		t.Helper()
		require.Equal(t, expected, client.AllowsRedirectURI(redirectURI), "%s: %q", reason, redirectURI)
	}

	// A registered client matches its list exactly, and nothing else
	allows(registered, "https://app.example.com/callback", true, "listed")
	allows(registered, "urn:ietf:wg:oauth:2.0:oob", true, "listed out-of-band")
	allows(registered, "https://app.example.com/other", false, "same origin, but a registered client")
	allows(registered, "", false, "empty")

	// A URL client that publishes a list is held to it
	allows(listed, "https://app.example.com/callback", true, "listed")
	allows(listed, "https://app.example.com/other", false, "same origin, but not listed")

	// A URL client that lists nothing may use its own origin (IndieAuth §4.2.2)
	allows(unlisted, "https://fed.brid.gy/oauth/indieauth/finish", true, "Bridgy Fed's callback")
	allows(unlisted, "https://FED.brid.gy/finish?state=1", true, "host in another case, with a query")
	allows(unlisted, "HTTPS://fed.brid.gy/finish", true, "scheme in another case")
	allows(unlisted, "https://fed.brid.gy:443/finish", true, "the default port written out")
	allows(unlisted, "http://fed.brid.gy/finish", false, "another scheme")
	allows(unlisted, "https://fed.brid.gy:8443/finish", false, "another port")
	allows(unlisted, "https://evil.example/finish", false, "another host")
	allows(unlisted, "https://fed.brid.gy.evil.example/finish", false, "a host that starts the same")
	allows(unlisted, "https://evil.example@fed.brid.gy/finish", false, "credentials")
	allows(unlisted, "https://fed.brid.gy/finish#token", false, "a fragment")
	allows(unlisted, "/finish", false, "relative")
	allows(unlisted, "javascript:alert(1)", false, "script")
	allows(unlisted, "", false, "empty")

	// A port in the client_id must match, and so must its absence
	allows(ported, "http://localhost:8080/callback", true, "same port")
	allows(ported, "http://localhost/callback", false, "default port")
}
