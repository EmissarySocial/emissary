package config

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestOwnerSchema(t *testing.T) {

	o := NewOwner()
	s := schema.New(OwnerSchema())

	table := []tableTestItem{
		{"displayName", "DISPLAY_NAME", nil},
		{"username", "USERNAME", nil},
		{"emailAddress", "EMAIL_ADDRESS", nil},
		{"phoneNumber", "PHONE_NUMBER", nil},
		{"mailingAddress", "MAILING_ADDRESS", nil},
	}

	tableTest_Schema(t, &s, &o, table)
}
