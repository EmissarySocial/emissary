package handler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetIntent_Continue_NeutralizesDangerousSchemes proves that a caller-supplied
// Activity Intent target with a dangerous scheme (e.g. "javascript:") never reaches
// the Continue link's href. Such values fall back to the user's home page instead.
func TestGetIntent_Continue_NeutralizesDangerousSchemes(t *testing.T) {

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
		result := getIntent_Continue(target)

		// The dangerous scheme must not survive into the rendered link.
		require.NotContains(t, strings.ToLower(result), "javascript:", "input %q leaked a javascript: href", target)
		require.NotContains(t, strings.ToLower(result), "vbscript:", "input %q leaked a vbscript: href", target)
		require.NotContains(t, strings.ToLower(result), "data:text/html", "input %q leaked a data: href", target)
		require.NotContains(t, result, "//evil.example", "input %q leaked an off-site host", target)

		// The Continue link falls back to the safe home page.
		require.Contains(t, result, `href="/@me"`, "input %q should fall back to /@me", target)
	}
}

// TestGetIntent_Continue_PreservesSafeTargets proves that legitimate targets — a
// same-site relative path or an off-site http(s) URL shown for user confirmation —
// are preserved in the Continue link.
func TestGetIntent_Continue_PreservesSafeTargets(t *testing.T) {

	require.Contains(t, getIntent_Continue("/@me/stream/123"), `href="/@me/stream/123"`)
	require.Contains(t, getIntent_Continue("https://remote.example/@author"), `href="https://remote.example/@author"`)
}

// TestGetIntent_Continue_CloseDirective proves the "(close)" sentinel still short-circuits
// to the window-closing script without a confirmation page.
func TestGetIntent_Continue_CloseDirective(t *testing.T) {
	require.Equal(t, "<script>window.close();</script>", getIntent_Continue("(close)"))
}
