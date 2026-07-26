package service

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestParsePath(t *testing.T) {

	id1, _ := primitive.ObjectIDFromHex("123456789012345678901234")

	{
		urlValue, userID, objectType, objectID, err := ParseProfileURL("example.com", "https://example.com/@123456789012345678901234")
		require.Nil(t, err)
		require.NotNil(t, urlValue)
		require.Equal(t, id1, userID)
		require.Empty(t, objectType)
		require.Empty(t, objectID)
	}

	{
		urlValue, userID, objectType, objectID, err := ParseProfileURL("example.com", "https://example.com/@123456789012345678901234/pub")
		require.Nil(t, err)
		require.NotNil(t, urlValue)
		require.Equal(t, id1, userID)
		require.Empty(t, objectType)
		require.Empty(t, objectID)
	}
	{
		urlValue, userID, objectType, objectID, err := ParseProfileURL("example.com", "https://example.com/@123456789012345678901234/pub/followers")
		require.Nil(t, err)
		require.NotNil(t, urlValue)
		require.Equal(t, id1, userID)
		require.Equal(t, "followers", objectType)
		require.Empty(t, objectID)
	}

	{
		id2, _ := primitive.ObjectIDFromHex("234567890123456789012345")

		urlValue, userID, objectType, objectID, err := ParseProfileURL("example.com", "https://example.com/@123456789012345678901234/pub/followers/234567890123456789012345")
		require.Nil(t, err)
		require.NotNil(t, urlValue)
		require.Equal(t, id1, userID)
		require.Equal(t, "followers", objectType)
		require.Equal(t, id2, objectID)
	}
}

func TestParsePathErrors(t *testing.T) {
	{
		_, _, _, _, err := ParseProfileURL("example.com", "not-a-url")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("example.com", "https://example.com")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("example.com", "https://example.com/not-a-username")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("example.com", "https://example.com/@not-an-objectid")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("example.com", "https://example.com/@123456789012345678901234/not-pub")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("example.com", "https://example.com/@123456789012345678901234/pub/followers/not-an-objectid")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("example.com", "https://example.com/@123456789012345678901234/pub/followers/234567890123456789012345/path-too-long")
		require.NotNil(t, err)
	}
}

// TestParseProfileURL_Hostname pins the ownership check: a profile URL is only local if it is on
// THIS domain.  Without it, only the path was inspected, so any server could hand us a look-alike
// path (`https://anywhere.example/@<localUserID>`) and have it resolve to one of our Users.
func TestParseProfileURL_Hostname(t *testing.T) {

	const hexID = "123456789012345678901234"
	id, err := primitive.ObjectIDFromHex(hexID)
	require.Nil(t, err)

	// isLocal asserts whether `value` is accepted as an actor URL belonging to `hostname`
	isLocal := func(hostname string, value string, expected bool) {
		t.Helper()
		_, userID, _, _, err := ParseProfileURL(hostname, value)

		if expected {
			require.Nil(t, err, "should accept %q on %q", value, hostname)
			require.Equal(t, id, userID)
			return
		}

		require.NotNil(t, err, "should reject %q on %q", value, hostname)
		require.Equal(t, http.StatusUnprocessableEntity, derp.ErrorCode(err), "wrong host is 422, not 400/500")
		require.True(t, userID.IsZero(), "must not leak a UserID from a foreign URL")
	}

	// Matching hostnames are local
	isLocal("example.com", "https://example.com/@"+hexID, true)
	isLocal("example.com", "https://EXAMPLE.COM/@"+hexID, true)      // hostnames are case-insensitive
	isLocal("example.com", "https://example.com:8443/@"+hexID, true) // a non-standard port is the same domain
	isLocal("example.com", "http://example.com/@"+hexID, true)       // scheme is not part of the check
	isLocal("example.com", "//example.com/@"+hexID, true)            // protocol-relative still names our host

	// Everything else is foreign -- including the shapes an attacker would reach for
	isLocal("example.com", "https://anywhere.example/@"+hexID, false)
	isLocal("example.com", "https://example.com.evil.test/@"+hexID, false) // suffix look-alike
	isLocal("example.com", "https://evil.test/example.com/@"+hexID, false) // path look-alike
	isLocal("example.com", "https://sub.example.com/@"+hexID, false)       // subdomains are separate domains
	isLocal("example.com", "https://user@evil.test/@"+hexID, false)        // userinfo look-alike
	isLocal("example.com", "/@"+hexID, false)                              // relative: no hostname at all
	isLocal("example.com", "https://mastodon.social/users/benpate", false) // the shape from the live error log

	// An empty hostname would disable the check entirely, so it is refused as a programming error
	_, _, _, _, err = ParseProfileURL("", "https://example.com/@"+hexID)
	require.NotNil(t, err)
	require.Equal(t, http.StatusInternalServerError, derp.ErrorCode(err))
}

func TestParseFollowersURI(t *testing.T) {

	host := "https://example.com"
	hexID := "123456789012345678901234"
	id, err := primitive.ObjectIDFromHex(hexID)
	require.Nil(t, err)

	do := func(uri string, wantType string, wantID primitive.ObjectID) {
		t.Helper()
		gotType, gotID := parseFollowersURI(host, uri)
		require.Equal(t, wantType, gotType, "actorType for %q", uri)
		require.Equal(t, wantID, gotID, "actorID for %q", uri)
	}

	// User followers collections
	do("https://example.com/@"+hexID+"/pub/followers", model.ActorTypeUser, id) // long form
	do("followers:"+hexID, model.ActorTypeUser, id)                             // shortcut scheme

	// Stream (bare hex, no "@") and SearchQuery ("@search_") followers collections — W6
	do("https://example.com/"+hexID+"/pub/followers", model.ActorTypeStream, id)
	do("https://example.com/@search_"+hexID+"/pub/followers", model.ActorTypeSearchQuery, id)

	// NOT a local followers collection → ("", NilObjectID)
	do("https://example.com/@"+hexID+"/pub/followers/", "", primitive.NilObjectID)            // trailing slash
	do("https://example.com/@"+hexID+"/invalid-other-path/", "", primitive.NilObjectID)       // wrong suffix
	do("https://example.com/@not-a-valid-userid/pub/followers", "", primitive.NilObjectID)    // non-hex user
	do("https://example.com/not-a-valid-id/pub/followers", "", primitive.NilObjectID)         // non-hex stream
	do("https://example.com/@search_not-hex/pub/followers", "", primitive.NilObjectID)        // non-hex search
	do("https://not-even-your-domain.bro/"+hexID+"/pub/followers", "", primitive.NilObjectID) // foreign host
	do("followers:not-hex", "", primitive.NilObjectID)                                        // bad shortcut token
}
