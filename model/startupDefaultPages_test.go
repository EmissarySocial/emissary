package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStartupDefaultPages_IsEmpty verifies that any one landing page makes the value non-empty
func TestStartupDefaultPages_IsEmpty(t *testing.T) {

	require.True(t, StartupDefaultPages{}.IsEmpty())
	require.False(t, StartupDefaultPages{Anonymous: "/home"}.IsEmpty())
	require.False(t, StartupDefaultPages{Authenticated: "/@me"}.IsEmpty())
	require.False(t, StartupDefaultPages{Owner: "/admin"}.IsEmpty())
}

// TestStartupDefaultPages_Apply verifies that every landing page reaches its own Domain field
func TestStartupDefaultPages_Apply(t *testing.T) {

	writableDomain := NewWritableDomain()
	pages := StartupDefaultPages{Anonymous: "/home", Authenticated: "/@me/newsfeed", Owner: "/admin"}

	require.True(t, pages.Apply(&writableDomain))
	require.Equal(t, "/home", writableDomain.DefaultAnonymous)
	require.Equal(t, "/@me/newsfeed", writableDomain.DefaultAuthenticated)
	require.Equal(t, "/admin", writableDomain.DefaultOwner)

	// A second pass changes nothing, so the caller can skip its save
	require.False(t, pages.Apply(&writableDomain))
}

// TestStartupDefaultPages_Apply_KeepsUnnamedPages verifies that an empty landing page leaves the Domain's own value alone
func TestStartupDefaultPages_Apply_KeepsUnnamedPages(t *testing.T) {

	writableDomain := NewWritableDomain()
	writableDomain.DefaultAnonymous = "/welcome"
	writableDomain.DefaultOwner = "/admin/users"

	require.True(t, StartupDefaultPages{Authenticated: "/@me"}.Apply(&writableDomain))
	require.Equal(t, "/welcome", writableDomain.DefaultAnonymous)
	require.Equal(t, "/@me", writableDomain.DefaultAuthenticated)
	require.Equal(t, "/admin/users", writableDomain.DefaultOwner)

	require.False(t, StartupDefaultPages{}.Apply(&writableDomain))
}

// TestStartupDefaultPages_Apply_RefusesOtherWebsites verifies that a landing page on another website is never stored
func TestStartupDefaultPages_Apply_RefusesOtherWebsites(t *testing.T) {

	writableDomain := NewWritableDomain()
	writableDomain.DefaultAnonymous = "/welcome"

	pages := StartupDefaultPages{
		Anonymous:     "https://evil.example/",
		Authenticated: "//evil.example/",
		Owner:         "/\\evil.example/",
	}

	require.False(t, pages.Apply(&writableDomain))
	require.Equal(t, "/welcome", writableDomain.DefaultAnonymous)
	require.Empty(t, writableDomain.DefaultAuthenticated)
	require.Empty(t, writableDomain.DefaultOwner)
}

// TestStartupDefaultPages_Apply_NilDomain verifies that a nil Domain is refused without a panic
func TestStartupDefaultPages_Apply_NilDomain(t *testing.T) {
	require.False(t, StartupDefaultPages{Anonymous: "/home"}.Apply(nil))
}

// TestIsStartupDefaultPage covers each class of path a Theme might name
func TestIsStartupDefaultPage(t *testing.T) {

	check := func(value string, expected bool) {
		t.Helper()
		require.Equal(t, expected, isStartupDefaultPage(value), "checking %q", value)
	}

	// Paths on this website
	check("/", true)
	check("/home", true)
	check("/@me/newsfeed", true)
	check("/search?q=music", true)
	check("/"+strings.Repeat("x", 127), true)

	// Nothing to store
	check("", false)

	// Not rooted on this website
	check("home", false)
	check("https://evil.example/", false)
	check("javascript:alert(1)", false)
	check("//evil.example/", false)
	check("/\\evil.example/", false)

	// Not storable, or not safe in a redirect header
	check("/"+strings.Repeat("x", 128), false)
	check("/home\r\nSet-Cookie: x=1", false)
	check("/home\x00", false)
	check("/home\x7f", false)
}

// FuzzIsStartupDefaultPage verifies that every accepted value is a short, rooted, single-line path
func FuzzIsStartupDefaultPage(f *testing.F) {

	for _, seed := range []string{"", "/", "/home", "//evil.example", "/\\evil.example", "https://evil.example", "/a\r\nb", "\x00", "/\xff\xfe"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {

		if !isStartupDefaultPage(value) {
			return
		}

		require.True(t, strings.HasPrefix(value, "/"))
		require.False(t, strings.HasPrefix(value, "//"))
		require.False(t, strings.HasPrefix(value, "/\\"))
		require.LessOrEqual(t, len(value), 128)
		require.NotContains(t, value, "\n")
		require.NotContains(t, value, "\r")
	})
}
