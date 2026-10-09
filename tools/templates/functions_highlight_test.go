package templates

import (
	"html/template"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// highlightFunc returns the "highlight" helper from a FuncMap built with no icon provider
func highlightFunc(t *testing.T) func(string, string) template.HTML {

	t.Helper()

	fn, ok := FuncMap(nil)["highlight"].(func(string, string) template.HTML)
	require.True(t, ok, "highlight helper must be registered with this signature")

	return fn
}

// TestHighlight_EscapesText is the XSS property for a helper that returns template.HTML.
//
// RULE: html/template will not escape a template.HTML value downstream, so the helper
// must escape its own inputs.  See the funcmap notes in AGENTS.md.
func TestHighlight_EscapesText(t *testing.T) {

	highlight := highlightFunc(t)

	result := string(highlight(`<script>alert(1)</script>`, ""))

	require.NotContains(t, result, "<script>")
	require.Contains(t, result, "&lt;script&gt;")
}

// TestHighlight_EscapesTextWhileWrappingMatch proves escaping survives the wrap path
func TestHighlight_EscapesTextWhileWrappingMatch(t *testing.T) {

	highlight := highlightFunc(t)

	result := string(highlight(`hello <img src=x onerror=alert(1)> world`, "world"))

	require.NotContains(t, result, "<img")
	require.Contains(t, result, `<b class="highlight">world</b>`)
}

// TestHighlight_EscapesSearchTerm pins the other untrusted input: the search term itself
func TestHighlight_EscapesSearchTerm(t *testing.T) {

	highlight := highlightFunc(t)

	result := string(highlight(`a "><script>x</script> b`, `"><script>x</script>`))

	require.NotContains(t, result, "<script>")
	require.Equal(t, 1, strings.Count(result, `<b class="highlight">`))
}

// TestHighlight_LeavesPlainTextAlone proves the helper still does its actual job
func TestHighlight_LeavesPlainTextAlone(t *testing.T) {

	highlight := highlightFunc(t)

	require.Equal(t, template.HTML("the quick fox"), highlight("the quick fox", ""))
	require.Equal(t, template.HTML(`the <b class="highlight">quick</b> fox`), highlight("the quick fox", "quick"))
}

// TestIconFilled_NilProviderDoesNotPanic pins the guard that "icon" already had
func TestIconFilled_NilProviderDoesNotPanic(t *testing.T) {

	fn, ok := FuncMap(nil)["iconFilled"].(func(string) template.HTML)
	require.True(t, ok)
	require.NotPanics(t, func() { fn("folder") })
	require.Equal(t, template.HTML(""), fn("folder"))
}
