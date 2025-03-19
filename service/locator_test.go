package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocator(t *testing.T) {

	do := func(value string, objType string, objToken string) {
		resultType, resultToken := locateObjectFromURL("https://example.com", value)
		require.Equal(t, objType, resultType)
		require.Equal(t, objToken, resultToken)
	}

	// Identify URLs
	do("https://example.com", "Service", "")          // Special case for service account
	do("https://example.com/", "Service", "")         // Special case for service account with trailing slash
	do("https://example.com/@service", "Service", "") // Service account

	do("https://example.com/1234", "Stream", "1234")         // Stream by ID
	do("https://example.com/token/", "Stream", "token")      // Stream by token (with trailing slash)
	do("https://example.com/token/route", "Stream", "token") // Stream by token (with trailing route)

	do("https://example.com/@search-1234", "SearchQuery", "1234") // SearchQuery by ID

	do("https://example.com/@1234", "User", "1234")                      // User by ID
	do("https://example.com/@username", "User", "username")              // User by username
	do("https://example.com/@username/other-routes", "User", "username") // User by username (with trailing route)

	// Identify Usernames
	do("acct:benpate@example.com", "User", "benpate")  // Username with acct: prefix
	do("benpate@example.com", "User", "benpate")       // Username without acct: prefix
	do("@benpate@example.com", "User", "benpate")      // Username with leading @
	do("acct:@benpate@example.com", "User", "benpate") // Username with acct: and leading @

	do("acct:search-12345678@example.com", "SearchQuery", "12345678")  // SearchQuery with acct: prefix
	do("search-12345678@example.com", "SearchQuery", "12345678")       // SearchQuery without acct: prefix
	do("@search-12345678@example.com", "SearchQuery", "12345678")      // SearchQuery with leading @
	do("acct:@search-12345678@example.com", "SearchQuery", "12345678") // SearchQuery with acct: and leading @

	do("service@example.com", "Service", "")  // Service account
	do("@service@example.com", "Service", "") // Service account with leading @
}
