package templates

import (
	"html/template"
	"testing"

	"github.com/stretchr/testify/require"
)

// markdownFunc returns the named Markdown helper from a FuncMap built with no icon provider
func markdownFunc(t *testing.T, name string) func(any) template.HTML {

	t.Helper()

	fn, ok := FuncMap(nil)[name].(func(any) template.HTML)
	require.True(t, ok, name+" helper must be registered with this signature")

	return fn
}

// TestMarkdownHardWraps_BreaksLines shows the one difference between the two Markdown helpers
func TestMarkdownHardWraps_BreaksLines(t *testing.T) {

	source := "one\ntwo"

	require.NotContains(t, string(markdownFunc(t, "markdown")(source)), "<br")
	require.Contains(t, string(markdownFunc(t, "markdownHardWraps")(source)), "<br")
}

// TestMarkdownHardWraps_Sanitizes is the XSS property for a helper that returns template.HTML
func TestMarkdownHardWraps_Sanitizes(t *testing.T) {

	result := string(markdownFunc(t, "markdownHardWraps")("one\ntwo <script>alert(1)</script>"))

	require.NotContains(t, result, "<script")
}
