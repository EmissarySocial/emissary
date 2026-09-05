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

// TestFollowerUnsubscribeLink_IgnoresMethod verifies that the link is built for every Follower,
// not just email ones.
//
// The URL is public and can be typed by hand, so an empty return would have been the appearance
// of an authorization check rather than one.  The real gate is service.Follower.LoadBySecret,
// which loads EMAIL records only -- pinned by TestFollowerLoadBySecret_RejectsOtherMethods.
func TestFollowerUnsubscribeLink_IgnoresMethod(t *testing.T) {

	follower := NewFollower()
	follower.Method = FollowerMethodActivityPub

	link := follower.UnsubscribeLink("https://example.com")

	require.NotEmpty(t, link)
	require.Equal(t, "<"+link+">", follower.UnsubscribeLinkWithBrackets("https://example.com"))
}

// TestFollowerSchema_AcceptsAnEmailFollower guards the validation that silently broke every
// email subscription
func TestFollowerSchema_AcceptsAnEmailFollower(t *testing.T) {

	// An EMAIL Follower keeps its address in Actor.ProfileURL, which is how LoadByActor finds
	// one.  A bare address has no scheme and no host, so while that field carried a `url`
	// format, Follower.Save rejected signup, confirmation, and the Mailchimp webhook alike.

	follower := NewFollower()
	follower.ParentType = FollowerTypeUser
	follower.StateID = FollowerStateActive
	follower.Method = FollowerMethodEmail
	follower.Format = MimeTypeHTML
	follower.Actor.ProfileURL = "sarah@connor.mil"
	follower.Actor.EmailAddress = "sarah@connor.mil"
	follower.Actor.Name = "Sarah Connor"

	s := schema.New(FollowerSchema())
	_, err := s.Validate(&follower)

	require.NoError(t, err)
}
