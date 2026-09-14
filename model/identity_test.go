package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
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

// TestIdentity_SetIdentifier_ActivityPubClearsHandle pins that binding an actor discards the old handle,
// so a stale handle can never sit beside a different actor
func TestIdentity_SetIdentifier_ActivityPubClearsHandle(t *testing.T) {

	identity := NewIdentity()
	identity.WebfingerUsername = "@alice@evil.example"

	require.True(t, identity.SetIdentifier(IdentifierTypeActivityPub, "https://good.example/@bob"))
	require.Equal(t, "https://good.example/@bob", identity.ActivityPubActor)
	require.Equal(t, "", identity.WebfingerUsername)

	// The mirror image: setting a handle discards the actor
	require.True(t, identity.SetIdentifier(IdentifierTypeWebfinger, "@bob@good.example"))
	require.Equal(t, "", identity.ActivityPubActor)
	require.Equal(t, "@bob@good.example", identity.WebfingerUsername)

	// An unknown type changes nothing
	require.False(t, identity.SetIdentifier("BOGUS", "value"))
	require.Equal(t, "@bob@good.example", identity.WebfingerUsername)
}

// TestIdentity_IsEmpty pins that an Identity holding only an actor is not empty
func TestIdentity_IsEmpty(t *testing.T) {

	table := []struct {
		email  string
		handle string
		actor  string
		empty  bool
	}{
		{"", "", "", true},
		{"sarah@example.com", "", "", false},
		{"", "@sarah@example.com", "", false},
		{"", "", "https://example.com/@sarah", false},
		{"sarah@example.com", "@sarah@example.com", "https://example.com/@sarah", false},
	}

	for _, item := range table {

		identity := NewIdentity()
		identity.EmailAddress = item.email
		identity.WebfingerUsername = item.handle
		identity.ActivityPubActor = item.actor

		require.Equal(t, item.empty, identity.IsEmpty(), "email %q handle %q actor %q", item.email, item.handle, item.actor)
	}
}
