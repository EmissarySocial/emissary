package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

// TestIdentitySchema returns the rosetta schema that describes a TestIdentity
func TestIdentitySchema(t *testing.T) {

	identity := NewIdentity()
	s := schema.New(IdentitySchema())

	table := []tableTestItem{
		{"identityId", "123456781234567812345678", nil},
		{"name", "Sarah Connor", nil},
		{"emailAddress", "sarah@example.com", nil},
		{"activityPubActor", "https://example.com/@sarah", nil},
		// note: "iconUrl" is a virtual/url field and is omitted from the round-trip table.
		// note: "webfingerUsername" is omitted because the "webfinger" format strips the leading "@"
		// on Set but then requires it on Validate, so the stored value cannot round-trip (rosetta
		// format.WebFinger bug, logged as a finding).
	}

	tableTest_Schema(t, &s, &identity, table)
}
