package mastodon

import (
	"testing"

	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
)

// TestAPIHashtag confirms which hashtag URLs are rewritten to "<origin>/tags/<name>" and which are left alone
func TestAPIHashtag(t *testing.T) {

	// check converts one #Cats URL and compares the result
	check := func(name string, tagURL string, want string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			got := apiHashtag(object.StatusTag{Name: "Cats", URL: tagURL})
			require.Equal(t, object.StatusTag{Name: "Cats", URL: want}, got, "converting %q", tagURL)
		})
	}

	// Rewritten to the origin's /tags/ URL
	check("search URL", "https://example.com/search?q=%23Cats", "https://example.com/tags/Cats")
	check("name is not the last segment", "https://example.com/tags/Cats/feed", "https://example.com/tags/Cats")
	check("no tags segment", "https://example.com/hashtags/Cats", "https://example.com/tags/Cats")
	check("tags inside a longer segment", "https://example.com/mytags/Cats", "https://example.com/tags/Cats")
	check("scheme is kept", "http://example.com/Cats", "http://example.com/tags/Cats")
	check("port is kept", "https://example.com:8443/Cats", "https://example.com:8443/tags/Cats")
	check("userinfo is dropped", "https://user:secret@example.com/Cats", "https://example.com/tags/Cats")

	// Already Mastodon-shaped
	check("tags URL", "https://mastodon.social/tags/Cats", "https://mastodon.social/tags/Cats")
	check("name matches ignoring case", "https://mastodon.social/tags/cats", "https://mastodon.social/tags/cats")
	check("tags segment deeper in the path", "https://example.com/tags/x/Cats", "https://example.com/tags/x/Cats")

	// No origin to rewrite against
	check("relative tags URL", "/tags/Cats", "/tags/Cats")
	check("relative search URL", "/search?q=Cats", "/search?q=Cats")
	check("empty URL", "", "")

	// Cannot be parsed
	check("invalid escape", "https://example.com/%zz", "https://example.com/%zz")
	check("control character", "https://example.com/\x7f", "https://example.com/\x7f")
}

// TestAPIHashtags confirms every tag is converted in order without changing the input, and an empty list stays empty
func TestAPIHashtags(t *testing.T) {

	tags := []object.StatusTag{
		{Name: "Cats", URL: "https://example.com/search?q=Cats"},
		{Name: "Dogs", URL: "https://mastodon.social/tags/Dogs"},
	}

	require.Equal(t, []object.StatusTag{
		{Name: "Cats", URL: "https://example.com/tags/Cats"},
		{Name: "Dogs", URL: "https://mastodon.social/tags/Dogs"},
	}, apiHashtags(tags))

	require.Equal(t, "https://example.com/search?q=Cats", tags[0].URL, "the input slice must not change")
	require.Equal(t, []object.StatusTag{}, apiHashtags(nil))
}
