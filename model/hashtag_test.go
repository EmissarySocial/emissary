package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHashtagURLPrefix_Relative confirms that a relative TagURL is anchored to the hostname.
func TestHashtagURLPrefix_Relative(t *testing.T) {
	require.Equal(t, "https://example.com/search?q=", HashtagURLPrefix("https://example.com", "/search?q="))
}

// TestHashtagURLPrefix_Absolute confirms that a TagURL that already names a host is used as-is.
func TestHashtagURLPrefix_Absolute(t *testing.T) {
	require.Equal(t, "https://elsewhere.com/tags?q=", HashtagURLPrefix("https://example.com", "https://elsewhere.com/tags?q="))
}

// TestHashtagURLPrefix_Empty confirms that an empty TagURL still means "do not link"
func TestHashtagURLPrefix_Empty(t *testing.T) {
	require.Equal(t, "", HashtagURLPrefix("https://example.com", ""))
}

// TestHashtagURLPrefix_NoHostname confirms that a missing hostname degrades to a relative URL
// instead of a broken one.
func TestHashtagURLPrefix_NoHostname(t *testing.T) {
	require.Equal(t, "/search?q=", HashtagURLPrefix("", "/search?q="))
}

// TestHashtagURL confirms that a complete hashtag link is absolute.
func TestHashtagURL(t *testing.T) {
	require.Equal(t, "https://example.com/search?q=%23travel", HashtagURL("https://example.com", "/search?q=", "travel"))
}

// TestHashtagURL_Empty confirms that no TagURL means no link at all (not a bare "%23tag").
func TestHashtagURL_Empty(t *testing.T) {
	require.Equal(t, "", HashtagURL("https://example.com", "", "travel"))
}

// TestHashtagURL_Escapes confirms that tag names are escaped the same way replace.Linkify escapes
// them, so a document's metadata links match the anchors inside its content.
func TestHashtagURL_Escapes(t *testing.T) {
	require.Equal(t, "https://example.com/search?q=%23caf%C3%A9", HashtagURL("https://example.com", "/search?q=", "café"))
	require.Equal(t, `https://example.com/search?q=%23%22%3E%3Cscript%3E`, HashtagURL("https://example.com", "/search?q=", `"><script>`))
}
