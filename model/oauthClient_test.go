package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

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
