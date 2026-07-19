package handler

import (
	"strings"
	"testing"

	"github.com/benpate/html"
	"github.com/stretchr/testify/require"
)

// TestSafeIntentURL_NeutralizesDangerousSchemes proves that caller-supplied / remote-fetched URLs
// with a dangerous scheme (or off-site protocol-relative form) are dropped before they reach an
// intent page's href/src, guarding the actor-URL and actor-icon sinks in the follow/like/dislike
// intent handlers.
func TestSafeIntentURL_NeutralizesDangerousSchemes(t *testing.T) {

	dangerous := []string{
		"javascript:alert(document.domain)",
		"JavaScript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"vbscript:msgbox(1)",
		"file:///etc/passwd",
		"//evil.example/phish",  // protocol-relative == off-site
		"\tjavascript:alert(1)", // leading-whitespace scheme smuggling
	}

	for _, target := range dangerous {
		require.Equal(t, "", safeIntentURL(target), "input %q should be neutralized to an empty URL", target)
	}
}

// TestSafeIntentURL_PreservesSafeURLs proves that legitimate actor URLs/icons — an off-site http(s)
// address or a same-site relative path — pass through unchanged.
func TestSafeIntentURL_PreservesSafeURLs(t *testing.T) {

	safe := []string{
		"https://remote.example/@author",
		"http://remote.example/icon.png",
		"/@me/stream/123",
		"", // an empty URL is already harmless
	}

	for _, target := range safe {
		require.Equal(t, target, safeIntentURL(target), "input %q should be preserved", target)
	}
}

// TestWriteIntentObjectContent_SanitizesRemoteHTML is the regression test for the like/dislike
// intent XSS: the object being liked/disliked is fetched from a caller-supplied URL, so its
// summary/content are untrusted HTML. They must be sanitized (scripts + event handlers stripped)
// before being written via InnerHTML.
func TestWriteIntentObjectContent_SanitizesRemoteHTML(t *testing.T) {

	render := func(summary string, content string) string {
		b := html.New()
		write_intentObjectContent(b, summary, content)
		return b.String()
	}

	// A malicious summary must have its script / event-handler markup stripped.
	summaryOut := render(`<img src=x onerror=alert(document.domain)><script>alert(1)</script>hello`, "")
	require.NotContains(t, summaryOut, "<script", "script tag survived sanitization")
	require.NotContains(t, summaryOut, "onerror", "event handler survived sanitization")
	require.Contains(t, summaryOut, "hello", "benign text should be preserved")

	// When summary is empty, the (also untrusted) content is sanitized the same way.
	contentOut := render("", `<a href="javascript:alert(1)">click</a><b>ok</b>`)
	require.NotContains(t, strings.ToLower(contentOut), "javascript:", "javascript: href survived sanitization")
	require.Contains(t, contentOut, "<b>ok</b>", "benign markup should be preserved")

	// Summary is preferred over content, matching the original handler behavior.
	both := render("SUMMARY", "CONTENT")
	require.Contains(t, both, "SUMMARY")
	require.NotContains(t, both, "CONTENT")

	// Empty in, empty out (no stray element).
	require.Equal(t, "", render("", ""))
}
