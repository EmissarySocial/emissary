package model

import (
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

// TestUser_SummaryHTML_DropsDangerousHTML confirms that markdown rendering does not emit executable markup.
func TestUser_SummaryHTML_DropsDangerousHTML(t *testing.T) {
	user := NewUser()
	user.StatusMessage = "<script>alert(1)</script> [x](javascript:alert(1))"

	result := user.SummaryHTML()

	require.NotContains(t, result, "<script>")
	require.NotContains(t, result, "javascript:")
}
