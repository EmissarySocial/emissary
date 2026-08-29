package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestFollowerSchema returns the rosetta schema that describes a TestFollower
func TestFollowerSchema(t *testing.T) {

	follower := NewFollower()
	s := schema.New(FollowerSchema())

	table := []tableTestItem{
		{"followerId", "123456781234567812345678", nil},
		{"parentId", "876543218765432187654321", nil},
		{"type", FollowerTypeUser, nil},
		{"method", FollowerMethodActivityPub, nil},
		{"format", MimeTypeActivityPub, nil},
		{"stateId", FollowerStateActive, nil},
		{"actor.name", "ACTOR NAME", nil},
		{"data.first", "DATA FIRST", nil},
		{"expireDate", "1234", int64(1234)},
	}

	tableTest_Schema(t, &s, &follower, table)
}

// testEmailFollower returns a Follower that unsubscribes by email, with a known secret
func testEmailFollower() Follower {

	follower := NewFollower()
	follower.Method = FollowerMethodEmail
	follower.ParentType = FollowerTypeUser
	follower.Data.SetString("secret", "abc123")

	return follower
}

// TestFollowerUnsubscribeLinkWithBrackets verifies that the RFC 2369 form wraps the plain link
func TestFollowerUnsubscribeLinkWithBrackets(t *testing.T) {

	follower := testEmailFollower()

	plain := follower.UnsubscribeLink("https://example.com")
	require.NotEmpty(t, plain)

	require.Equal(t, "<"+plain+">", follower.UnsubscribeLinkWithBrackets("https://example.com"))
}

// TestFollowerUnsubscribeLinkWithBrackets_Empty verifies that a Follower with no unsubscribe link
// yields an empty string rather than a bare "<>".  ServerEmail omits headers that render empty, so
// an empty result is what keeps a malformed List-Unsubscribe out of the message.
func TestFollowerUnsubscribeLinkWithBrackets_Empty(t *testing.T) {

	follower := NewFollower()
	follower.Method = FollowerMethodActivityPub

	require.Empty(t, follower.UnsubscribeLink("https://example.com"))
	require.Empty(t, follower.UnsubscribeLinkWithBrackets("https://example.com"))
}
