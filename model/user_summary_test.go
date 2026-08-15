package model

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUser_SummaryHTML renders the bio Markdown and linkifies hashtags when a TagURL is defined.
// The links are absolute (anchored to the User's ProfileURL) because this HTML is federated as the
// ActivityPub "summary".
func TestUser_SummaryHTML(t *testing.T) {
	user := NewUser()
	user.StatusMessage = "Travel blogger #travel and #Food2024"
	user.Hashtags = []string{"travel", "Food2024"}
	user.ProfileURL = "https://example.com/@123456781234567812345678"
	user.TagURL = "/search?q="

	result := user.SummaryHTML()

	require.Contains(t, result, `<a href="https://example.com/search?q=%23travel" target="_blank">#travel</a>`)
	require.Contains(t, result, `<a href="https://example.com/search?q=%23Food2024" target="_blank">#Food2024</a>`)
}

// TestUser_SummaryHTML_NoProfileURL confirms that a User with no ProfileURL degrades to a relative
// link rather than a broken absolute one.
func TestUser_SummaryHTML_NoProfileURL(t *testing.T) {
	user := NewUser()
	user.StatusMessage = "Travel blogger #travel"
	user.Hashtags = []string{"travel"}
	user.TagURL = "/search?q="

	result := user.SummaryHTML()

	require.Contains(t, result, `<a href="/search?q=%23travel" target="_blank">#travel</a>`)
}

// TestUser_SummaryHTML_NoTagURL renders Markdown but leaves hashtags as plain text when no TagURL is set.
func TestUser_SummaryHTML_NoTagURL(t *testing.T) {
	user := NewUser()
	user.StatusMessage = "Travel blogger #travel"
	user.Hashtags = []string{"travel"}

	result := user.SummaryHTML()

	require.NotContains(t, result, "<a ")
	require.Contains(t, result, "#travel")
}

// TestUser_SummaryHTML_RendersMarkdown confirms the bio Markdown is rendered to HTML.
func TestUser_SummaryHTML_RendersMarkdown(t *testing.T) {
	user := NewUser()
	user.StatusMessage = "**bold** bio"

	result := user.SummaryHTML()

	require.Contains(t, result, "<strong>bold</strong>")
}

// summaryEventHandler matches an on* event handler inside a real HTML tag.
// Escaped or plain text such as "[x](javascript:alert(1))" is inert and must not match.
var summaryEventHandler = regexp.MustCompile(`(?is)<[^>]*[\s/]on[a-z]+\s*=`)

// summaryScriptURL matches a javascript: scheme used as a URL inside a real HTML tag.
var summaryScriptURL = regexp.MustCompile(`(?is)<[^>]*=\s*["']?\s*javascript:`)

// TestUser_SummaryHTML_DropsDangerousHTML confirms that markdown rendering does not emit
// executable markup.  The assertions target markup the browser would ACT on: a leftover
// "javascript:" in plain text is inert, so matching the bare string would be a false alarm.
func TestUser_SummaryHTML_DropsDangerousHTML(t *testing.T) {

	inputs := []string{
		"<script>alert(1)</script> [x](javascript:alert(1))",
		"[x](javascript:alert(1))",
		"<img src=x onerror=alert(1)>",
		"<div onclick=\"alert(1)\">click</div>",
		"<a href=\"javascript:alert(1)\">click</a>",
	}

	for _, input := range inputs {
		user := NewUser()
		user.StatusMessage = input

		result := user.SummaryHTML()

		require.NotContains(t, strings.ToLower(result), "<script", "input: %q", input)
		require.NotRegexp(t, summaryEventHandler, result, "input: %q", input)
		require.NotRegexp(t, summaryScriptURL, result, "input: %q", input)
	}
}

// TestUser_SummaryHTML_SanitizesRatherThanEscapes pins the fact that a bio is rendered by
// the application's shared Markdown converter, which passes raw HTML through the sanitizer
// instead of escaping it.  Safe inline markup therefore renders, and unsafe markup is dropped.
func TestUser_SummaryHTML_SanitizesRatherThanEscapes(t *testing.T) {

	user := NewUser()
	user.StatusMessage = "<b>bold</b> <script>alert(1)</script>"

	result := user.SummaryHTML()

	require.Contains(t, result, "<b>bold</b>")
	require.NotContains(t, result, "<script")
}
