package replace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLinkify wraps a single hashtag in an anchor to the base URL.
func TestLinkify(t *testing.T) {
	result := Linkify("Testing #travel here", "/search?q=", []string{"travel"})
	require.Equal(t, `Testing <a href="/search?q=%23travel" target="_blank">#travel</a> here`, result)
}

// TestLinkify_Multiple links every hashtag in the list.
func TestLinkify_Multiple(t *testing.T) {
	result := Linkify("#travel and #Food2024", "/search?q=", []string{"travel", "Food2024"})
	require.Equal(t, `<a href="/search?q=%23travel" target="_blank">#travel</a> and <a href="/search?q=%23Food2024" target="_blank">#Food2024</a>`, result)
}

// TestLinkify_SkipsEmptyTag ignores an empty tag name.
func TestLinkify_SkipsEmptyTag(t *testing.T) {
	result := Linkify("no #tag change", "/search?q=", []string{""})
	require.Equal(t, "no #tag change", result)
}

// TestLinkify_Idempotent confirms that a second pass does not re-wrap the existing link.
func TestLinkify_Idempotent(t *testing.T) {
	once := Linkify("Testing #travel here", "/search?q=", []string{"travel"})
	twice := Linkify(once, "/search?q=", []string{"travel"})
	require.Equal(t, once, twice)
}

// TestLinkify_EscapesHostileTag confirms that a database-sourced tag name cannot break out of the
// href attribute or inject markup into the anchor text.
func TestLinkify_EscapesHostileTag(t *testing.T) {
	hostile := `evil"><img src=x>`
	result := Linkify(`before #`+hostile+` after`, "/search?q=", []string{hostile})

	require.NotContains(t, result, "<img")
	require.NotContains(t, result, `"><`)
	require.Contains(t, result, `%3Cimg`)
	require.Contains(t, result, `&lt;img`)
}
