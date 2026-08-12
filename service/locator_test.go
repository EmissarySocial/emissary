package service

import (
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/stretchr/testify/require"
)

func TestLocator(t *testing.T) {

	do := func(value string, objType string, objToken string) {
		resultType, resultToken := locateObjectFromURL("https://example.com", value)
		require.Equal(t, objType, resultType, "locating %q", value)
		require.Equal(t, objToken, resultToken, "locating %q", value)
	}

	// Identify URLs
	do("https://example.com", "Application", "")              // Special case for service account
	do("https://example.com/", "Application", "")             // Special case for service account with trailing slash
	do("https://example.com/@application", "Application", "") // Service account

	do("https://example.com/1234", "Stream", "1234")         // Stream by ID
	do("https://example.com/token/", "Stream", "token")      // Stream by token (with trailing slash)
	do("https://example.com/token/route", "Stream", "token") // Stream by token (with trailing route)

	do("https://example.com/@search", "SearchDomain", "")    // Global search actor
	do("https://example.com/@search_1234", "Search", "1234") // Search by ID

	do("https://example.com/@1234", "User", "1234")                      // User by ID
	do("https://example.com/@username", "User", "username")              // User by username
	do("https://example.com/@username/other-routes", "User", "username") // User by username (with trailing route)

	// Identify Usernames
	do("acct:benpate@example.com", "User", "benpate")  // Username with acct: prefix
	do("benpate@example.com", "User", "benpate")       // Username without acct: prefix
	do("@benpate@example.com", "User", "benpate")      // Username with leading @
	do("acct:@benpate@example.com", "User", "benpate") // Username with acct: and leading @

	do("acct:search_12345678@example.com", "Search", "12345678")  // Search with acct: prefix
	do("search_12345678@example.com", "Search", "12345678")       // Search without acct: prefix
	do("@search_12345678@example.com", "Search", "12345678")      // Search with leading @
	do("acct:@search_12345678@example.com", "Search", "12345678") // Search with acct: and leading @

	do("search@example.com", "SearchDomain", "")  // Global search actor
	do("@search@example.com", "SearchDomain", "") // Global search actor with leading @

	do("application@example.com", "Application", "")  // Service account
	do("@application@example.com", "Application", "") // Service account with leading @

	// Naked usernames are assumed to be local.  This leniency is deliberate, and the foreign-host
	// check must not break it.
	do("benpate", "User", "benpate")
	do("@benpate", "User", "benpate")
	do("application", "Application", "")
	do("search_12345678", "Search", "12345678")
}

// TestLocator_ForeignHost is the regression guard for BUG-18: a resource that names another host must
// resolve to nothing.  Before the fix, "bob@other.example" fell through to the naked-username branch
// and was queried as a local User literally named "bob@other.example".
func TestLocator_ForeignHost(t *testing.T) {

	foreign := func(value string) {
		resultType, resultToken := locateObjectFromURL("https://example.com", value)
		require.Equal(t, "", resultType, "locating %q", value)
		require.Equal(t, "", resultToken, "locating %q", value)
	}

	// Accounts on another host
	foreign("acct:bob@other.example")
	foreign("bob@other.example")
	foreign("@bob@other.example")
	foreign("acct:@bob@other.example")
	foreign("acct:application@other.example")
	foreign("acct:search@other.example")
	foreign("acct:search_1234@other.example")

	// URLs on another host
	foreign("https://other.example")
	foreign("https://other.example/@bob")
	foreign("https://other.example/@application")
	foreign("https://other.example/token")

	// Hostnames that merely resemble ours
	foreign("acct:bob@example.com.other.example")
	foreign("acct:bob@sub.example.com")
	foreign("acct:bob@notexample.com")
	foreign("https://example.com.other.example/@bob")

	// A trailing host always wins over the naked-username reading, so an account cannot smuggle a
	// foreign host past the check by wearing a local one first.
	foreign("acct:bob@example.com@other.example")
}

// TestLocator_EmptyResource pins the empty and degenerate cases.  An empty resource must resolve to
// nothing rather than to a User whose token is the empty string, which would be a database query with
// no possible answer.  (The handler rejects a missing `resource` before it gets this far; this is the
// second line of defense, and covers the other callers of the locator.)
func TestLocator_EmptyResource(t *testing.T) {

	empty := func(value string) {
		resultType, resultToken := locateObjectFromURL("https://example.com", value)
		require.Equal(t, "", resultType, "locating %q", value)
		require.Equal(t, "", resultToken, "locating %q", value)
	}

	empty("")
	empty("acct:")
	empty("@")
	empty("acct:@")
	empty("@@example.com")
	empty("acct:@@example.com")

	// "@example.com" is NOT in this list.  A single leading "@" with no second "@" is the naked
	// username form, so this reads as a User named "example.com" -- the same answer the locator gave
	// before BUG-18.  It is a harmless database miss (usernames cannot contain "."), and reading it
	// as a host instead would have to break naked usernames to do it.
	resultType, resultToken := locateObjectFromURL("https://example.com", "@example.com")
	require.Equal(t, "User", resultType)
	require.Equal(t, "example.com", resultToken)
}

// TestLocator_HostComparison proves the host check folds case and ignores the port, per RFC 3986.  A
// server reached on a non-standard port still serves the same objects.
func TestLocator_HostComparison(t *testing.T) {

	do := func(value string, objType string, objToken string) {
		resultType, resultToken := locateObjectFromURL("https://example.com", value)
		require.Equal(t, objType, resultType, "locating %q", value)
		require.Equal(t, objToken, resultToken, "locating %q", value)
	}

	// Case-insensitive
	do("acct:benpate@EXAMPLE.COM", "User", "benpate")
	do("acct:benpate@Example.Com", "User", "benpate")
	do("https://EXAMPLE.COM/@benpate", "User", "benpate")

	// Port-insensitive
	do("acct:benpate@example.com:8443", "User", "benpate")
	do("https://example.com:8443/@benpate", "User", "benpate")

	// Protocol-insensitive.  The value describes an object on this host either way.
	do("http://example.com/@benpate", "User", "benpate")

	// A host configured WITH a port still matches values written without one
	resultType, resultToken := locateObjectFromURL("https://example.com:8443", "acct:benpate@example.com")
	require.Equal(t, "User", resultType)
	require.Equal(t, "benpate", resultToken)
}

// TestLocator_URLExtras confirms that query strings and fragments are discarded rather than glued onto
// the token.
func TestLocator_URLExtras(t *testing.T) {

	do := func(value string, objType string, objToken string) {
		resultType, resultToken := locateObjectFromURL("https://example.com", value)
		require.Equal(t, objType, resultType, "locating %q", value)
		require.Equal(t, objToken, resultToken, "locating %q", value)
	}

	do("https://example.com/token?rel=self", "Stream", "token")
	do("https://example.com/@benpate?rel=self", "User", "benpate")
	do("https://example.com/@benpate#main-key", "User", "benpate")
	do("https://example.com/?rel=self", "Application", "")
	do("https://example.com/%40benpate", "User", "benpate") // Percent-encoded "@"
}

// TestLocator_GetWebFingerResult_Foreign proves the service layer refuses a foreign resource with a
// 404, and does so WITHOUT touching the database.  Every dependent service on this Locator is nil, so
// any attempt to query one would panic rather than return.
func TestLocator_GetWebFingerResult_Foreign(t *testing.T) {

	locator := Locator{host: "https://example.com"}

	for _, resource := range []string{"acct:bob@other.example", "https://other.example/@bob", "", "@"} {

		result, err := locator.GetWebFingerResult(nil, resource)

		require.Error(t, err, "locating %q", resource)
		require.Equal(t, 404, derp.ErrorCode(err), "locating %q", resource)
		require.Equal(t, digit.Resource{}, result, "locating %q", resource)
	}
}

// FuzzLocateObjectFromURL asserts the two properties that must hold for every input a stranger can
// send to `/.well-known/webfinger`: the parser never panics, and no value can be made to resolve
// locally by appending a foreign host to it.
func FuzzLocateObjectFromURL(f *testing.F) {

	const host = "https://example.com"

	for _, seed := range []string{
		"", "@", "acct:", "acct:benpate@example.com", "@benpate@example.com", "benpate",
		"https://example.com/@benpate", "https://other.example/@benpate", "acct:bob@other.example",
		"https://example.com:8443/@benpate", "acct:benpate@EXAMPLE.COM", "search_1234",
		"://", "acct:%40benpate@example.com", "\x00", "\xff\xfe",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {

		resultType, _ := locateObjectFromURL(host, value)

		require.Contains(t, []string{"", "Application", "SearchDomain", "Search", "Stream", "User"}, resultType)

		// Any account-shaped value that ends in a foreign host belongs to that host, never to us.
		//
		// Two shapes are excluded. URL-shaped values, because their host is the authority and
		// everything after it is path data that may legitimately contain an "@". And values that are
		// empty once `acct:` is removed, because appending to those produces "@other.example" -- a
		// leading "@" followed by no host at all, which is the naked-username form and not an
		// account on another server.
		if trimmed := strings.TrimPrefix(value, "acct:"); (trimmed != "") && !strings.Contains(value, "://") {
			foreign := value + "@other.example"
			foreignType, foreignToken := locateObjectFromURL(host, foreign)
			require.Equal(t, "", foreignType, "locating %q", foreign)
			require.Equal(t, "", foreignToken, "locating %q", foreign)
		}
	})
}
