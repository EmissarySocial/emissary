package activitypub

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsSameOrigin verifies that two URLs are compared by host, and that malformed input is rejected
func TestIsSameOrigin(t *testing.T) {

	testCases := []struct {
		name     string
		first    string
		second   string
		expected bool
	}{
		// Same-origin cases
		{"identical", "https://x.social/@bob", "https://x.social/@bob", true},
		{"same host, different paths", "https://x.social/@bob", "https://x.social/notes/1", true},
		{"host case-insensitive", "https://X.Social/@bob", "https://x.social/notes/1", true},
		{"scheme case-insensitive", "HTTPS://x.social/a", "https://x.social/b", true},
		{"matching port", "https://x.social:8443/a", "https://x.social:8443/b", true},
		{"http scheme", "http://x.social/a", "http://x.social/b", true},
		{"ipv6 host", "https://[2001:db8::1]/a", "https://[2001:db8::1]/b", true},

		// Different-origin cases -- the security-relevant ones
		{"victim url as object", "https://attacker.example/@evil", "https://victim.example/notes/1", false},
		{"scheme mismatch", "http://x.social/a", "https://x.social/a", false},
		{"port mismatch", "https://x.social/a", "https://x.social:8443/a", false},
		{"subdomain mismatch", "https://x.social/a", "https://evil.x.social/a", false},

		// Garbage never binds
		{"empty first", "", "https://x.social/a", false},
		{"empty second", "https://x.social/a", "", false},
		{"both empty", "", "", false},
		{"first unparseable", "://not a url", "https://x.social/a", false},
		{"non-http scheme", "javascript:alert(1)", "javascript:alert(1)", false},
		{"mailto scheme", "mailto:bob@x.social", "mailto:bob@x.social", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, IsSameOrigin(testCase.first, testCase.second), testCase.name)
		})
	}
}
