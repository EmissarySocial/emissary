package customcss

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsLocalURL confirms which addresses a browser would load from the page's own website
func TestIsLocalURL(t *testing.T) {

	local := func(value string) {
		t.Helper()
		t.Run(value, func(t *testing.T) {
			require.True(t, isLocalURL(value), "reading %q", value)
		})
	}

	remote := func(value string) {
		t.Helper()
		t.Run(value, func(t *testing.T) {
			require.False(t, isLocalURL(value), "reading %q", value)
		})
	}

	local("")
	local("/a.png")
	local("a.png")
	local("../a.png")
	local("./a.png")
	local("?v=1")
	local("#blur")
	local("/a:b.png")
	local("a/b:c.png")
	local("1abc:x")
	local(":x")
	local("data:image/png;base64,AAAA")
	local("DATA:image/png;base64,AAAA")
	local("  /a.png  ")
	local("/%2F%2Fexample.com/a.png")
	local(" //example.com/a.png")

	remote("//example.com/a.png")
	remote("https://example.com/a.png")
	remote("HTTP://example.com/a.png")
	remote("https:example.com/a.png")
	remote("http:/example.com/a.png")
	remote("ftp://example.com/a.png")
	remote("javascript:alert(1)")
	remote("blob:https://example.com/x")
	remote("a+b-c.d:x")
	remote(`/\example.com/a.png`)
	remote(`\\example.com/a.png`)
	remote("/\t/example.com/a.png")
	remote("/\n/example.com/a.png")
	remote(" //example.com/a.png")
	remote("\x01//example.com/a.png")
	remote("ht\ttps://example.com/a.png")
}

// TestURLScheme confirms where a scheme ends, and that a relative path has none
func TestURLScheme(t *testing.T) {

	scheme, hasScheme := urlScheme("https://example.com")
	require.True(t, hasScheme)
	require.Equal(t, "https", scheme)

	scheme, hasScheme = urlScheme("web+ap:x")
	require.True(t, hasScheme)
	require.Equal(t, "web+ap", scheme)

	scheme, hasScheme = urlScheme("h2c.v1-x:x")
	require.True(t, hasScheme)
	require.Equal(t, "h2c.v1-x", scheme)

	for _, value := range []string{"", "/a", "a/b:c", "1a:b", ":a", "a b:c", "abc"} {
		_, hasScheme := urlScheme(value)
		require.False(t, hasScheme, "reading %q", value)
	}
}

// TestIsServerRoute confirms which local addresses reach a dot-prefixed route that is not a file
func TestIsServerRoute(t *testing.T) {

	file := func(value string) {
		t.Helper()
		t.Run(value, func(t *testing.T) {
			require.False(t, isServerRoute(value), "reading %q", value)
		})
	}

	route := func(value string) {
		t.Helper()
		t.Run(value, func(t *testing.T) {
			require.True(t, isServerRoute(value), "reading %q", value)
		})
	}

	file("")
	file("/a.png")
	file("a.png")
	file("./a.png")
	file("../a.png")
	file("#blur")
	file("?v=.x")
	file("/a.png#.x")
	file("/@me/attachments/1?width=300")
	file("/.themes/global/resources/a.png")
	file("/.templates/x/resources/a.png")
	file("/.widgets/x/resources/a.png")
	file("/.domain/attachments/1")
	file("/.searchTag/1/attachments/2")
	file("data:image/png;base64,AAAA")

	route("/.intents/Create?account=a@evil.example")
	route(".intents/Create")
	route("./.intents/Create")
	route("../.intents/Create")
	route("/x/../.intents/Create")
	route("/.themes/../.intents/Create")
	route("/%2Eintents/Create")
	route("/%2eintents/Create")
	route(`\.intents\Create`)
	route("/.in\ttents/Create")
	route("  /.intents/Create")
	route("/.proxy")
	route("/.well-known/webfinger")
	route(".themes/global/resources/a.png")
	route("/x/.themes/a.png")
	route("/.themesX/a.png")
	route("/%zz/a.png")
}
