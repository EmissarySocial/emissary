package config

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

// TestOwnerSchema verifies that every Owner property round-trips through the schema
func TestOwnerSchema(t *testing.T) {

	o := NewOwner()
	s := schema.New(OwnerSchema())

	table := []tableTestItem{
		{"displayName", "DISPLAY_NAME", nil},
		{"username", "USERNAME", nil},
		{"emailAddress", "EMAIL@ADDRESS.COM", nil},
		{"phoneNumber", "PHONE_NUMBER", nil},
		{"mailingAddress", "MAILING_ADDRESS", nil},
	}

	tableTest_Schema(t, &s, &o, table)
}
