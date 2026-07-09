package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseStream verifies the happy-path parsing of Stream URLs into a
// (streamToken, action) pair, including the "home"/"view" defaults and the
// discarding of trailing path/query segments.
func TestParseStream(t *testing.T) {

	locator := Locator{host: "https://example.com"}

	do := func(url string, expectedToken string, expectedAction string) {
		token, action, err := locator.ParseStream(url)
		require.Nil(t, err, "url: %s", url)
		require.Equal(t, expectedToken, token, "token for url: %s", url)
		require.Equal(t, expectedAction, action, "action for url: %s", url)
	}

	// Stream token, no action => "view"
	do("https://example.com/1234", "1234", "view")
	do("https://example.com/my-token", "my-token", "view")

	// Stream token with explicit action
	do("https://example.com/1234/edit", "1234", "edit")
	do("https://example.com/my-token/delete", "my-token", "delete")

	// Trailing slash after the token behaves like an empty action => "view"
	do("https://example.com/1234/", "1234", "view")

	// Only the first action segment is kept; deeper route data is discarded
	do("https://example.com/1234/edit/extra", "1234", "edit")
	do("https://example.com/1234/edit/extra/more", "1234", "edit")

	// Query strings are stripped from the action
	do("https://example.com/1234/edit?foo=bar", "1234", "edit")
	do("https://example.com/1234/edit?foo=bar&baz=qux", "1234", "edit")

	// A query string directly on the token (empty action) still defaults to "view",
	// and the query is discarded.
	do("https://example.com/1234?foo=bar", "1234?foo=bar", "view")

	// Bare host + slash resolves to the "home" stream (empty token default)
	do("https://example.com/", "home", "view")
}

// TestParseStream_HomeDefault isolates the empty-token behavior that mirrors the
// legacy ParsePath: a URL with no stream token resolves to "home"/"view".
func TestParseStream_HomeDefault(t *testing.T) {

	locator := Locator{host: "https://example.com"}

	token, action, err := locator.ParseStream("https://example.com/")
	require.Nil(t, err)
	require.Equal(t, "home", token)
	require.Equal(t, "view", action)
}

// TestParseStream_HostMismatch verifies that URLs which do not begin with the
// configured host (protocol + domain) are rejected. This is the guard that keeps
// a server from treating another server's URLs as its own Streams.
func TestParseStream_HostMismatch(t *testing.T) {

	locator := Locator{host: "https://example.com"}

	fail := func(url string) {
		token, action, err := locator.ParseStream(url)
		require.NotNil(t, err, "expected error for url: %s", url)
		require.Empty(t, token, "url: %s", url)
		require.Empty(t, action, "url: %s", url)
	}

	// Different host entirely
	fail("https://other.com/1234")

	// Same domain, wrong scheme
	fail("http://example.com/1234")

	// Loopback aliases are distinct hosts (localhost != 127.0.0.1)
	fail("https://localhost/1234")

	// A prefix that matches the host string but is not a host boundary
	// ("example.community" starts with "example.com"). The guard requires
	// the host to be followed by "/", so this must fail.
	fail("https://example.community/1234")

	// The bare host with no trailing slash does not satisfy host+"/" and fails
	fail("https://example.com")

	// Empty string
	fail("")

	// A WebFinger-style acct: value is not a URL for this host
	fail("acct:benpate@example.com")
}

// TestParseStream_RejectsActorPaths verifies that actor-style paths (which begin
// with "@" after the host) are rejected, since those are Users/Search actors, not
// Streams. Stream URLs are of the form host/token; actor URLs are host/@handle.
func TestParseStream_RejectsActorPaths(t *testing.T) {

	locator := Locator{host: "https://example.com"}

	fail := func(url string) {
		token, action, err := locator.ParseStream(url)
		require.NotNil(t, err, "expected error for url: %s", url)
		require.Empty(t, token, "url: %s", url)
		require.Empty(t, action, "url: %s", url)
	}

	fail("https://example.com/@username")
	fail("https://example.com/@username/pub")
	fail("https://example.com/@application")
	fail("https://example.com/@search")
	fail("https://example.com/@search_1234")
}

// TestParseStream_RejectsHiddenPaths verifies that paths whose token begins with
// a "." (dot-file / hidden-path style) are rejected.
func TestParseStream_RejectsHiddenPaths(t *testing.T) {

	locator := Locator{host: "https://example.com"}

	fail := func(url string) {
		token, action, err := locator.ParseStream(url)
		require.NotNil(t, err, "expected error for url: %s", url)
		require.Empty(t, token, "url: %s", url)
		require.Empty(t, action, "url: %s", url)
	}

	fail("https://example.com/.hidden")
	fail("https://example.com/.well-known/webfinger")
	fail("https://example.com/./edit")
}

// TestParseStream_RejectsReservedTokens verifies that reserved top-level tokens
// (which collide with built-in routes) are rejected as Stream tokens.
func TestParseStream_RejectsReservedTokens(t *testing.T) {

	locator := Locator{host: "https://example.com"}

	fail := func(url string) {
		token, action, err := locator.ParseStream(url)
		require.NotNil(t, err, "expected error for url: %s", url)
		require.Empty(t, token, "url: %s", url)
		require.Empty(t, action, "url: %s", url)
	}

	// Every reserved token must be rejected, with or without a trailing action.
	for _, reserved := range []string{"admin", "startup", "oauth", "signin", "signout", "register"} {
		fail("https://example.com/" + reserved)
		fail("https://example.com/" + reserved + "/edit")
	}

	// A non-reserved token that merely contains a reserved word is still valid.
	token, action, err := locator.ParseStream("https://example.com/administration")
	require.Nil(t, err)
	require.Equal(t, "administration", token)
	require.Equal(t, "view", action)
}

// TestIsReservedPath exhaustively checks the reserved-token predicate, including
// the non-reserved fall-through cases.
func TestIsReservedPath(t *testing.T) {

	// Every reserved token
	for _, reserved := range []string{"admin", "startup", "oauth", "signin", "signout", "register"} {
		require.True(t, isReservedPath(reserved), "expected %q to be reserved", reserved)
	}

	// Non-reserved tokens
	for _, allowed := range []string{"", "home", "administration", "signing", "register-me", "1234", "Admin"} {
		require.False(t, isReservedPath(allowed), "expected %q to NOT be reserved", allowed)
	}
}

// TestParseStream_HostWithPort verifies that ParseStream works when the configured
// host includes a port (as Factory.Host() produces for non-standard ports).
func TestParseStream_HostWithPort(t *testing.T) {

	locator := Locator{host: "http://localhost:8080"}

	token, action, err := locator.ParseStream("http://localhost:8080/1234/edit")
	require.Nil(t, err)
	require.Equal(t, "1234", token)
	require.Equal(t, "edit", action)

	// A URL for the same host without the port must NOT match
	token, action, err = locator.ParseStream("http://localhost/1234/edit")
	require.NotNil(t, err)
	require.Empty(t, token)
	require.Empty(t, action)
}
