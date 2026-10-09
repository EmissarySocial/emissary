package mailchimp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// testHex is filler shaped like the hex half of a credential, assembled from two
// halves so that secret scanners do not report this file as a live key
const testHex = "0123456789abcdef" + "0123456789abcdef"

// TestValidateAPIKey_FormatIsOpaque confirms that every credential shape is accepted
func TestValidateAPIKey_FormatIsOpaque(t *testing.T) {

	// Mailchimp's key format is theirs to change and an OAuth token does not share it.
	// Each value below would fail a format check; whether one WORKS is Mailchimp's call.

	credentials := map[string]string{
		"today's api key format": testHex + "-us6",
		"no data center suffix":  testHex,
		"an oauth-style value":   strings.Repeat("aZ9", 11),
		"uppercase hex":          strings.ToUpper(testHex) + "-US6",
		"longer than today":      testHex + testHex + "-us6",
		"shorter than today":     "abc123",
		"unexpected punctuation": testHex + ".us6",
		"a jwt-shaped value":     "eyJhbGciOiJIUzI1NiJ9." + "eyJzdWIiOiIxIn0.abc",
	}

	for name, apiKey := range credentials {

		t.Run(name, func(t *testing.T) {
			require.NoError(t, ValidateAPIKey(apiKey), "format must be opaque: %q", apiKey)
		})
	}
}

// TestValidateAPIKey_TransportSafety confirms a credential must be storable and
// sendable in an Authorization header
func TestValidateAPIKey_TransportSafety(t *testing.T) {

	rejected := map[string]string{
		"empty":            "",
		"whitespace only":  "   ",
		"interior newline": testHex + "\n-us6",
		"header injection": testHex + "-us6\r\nX-Evil: 1",
		"embedded null":    testHex + "\x00",
		"interior space":   testHex + " -us6",
		"interior tab":     testHex + "\t" + "-us6",
		"non-ascii":        testHex + "-us６",
		"absurdly long":    strings.Repeat("a", 257),
	}

	for name, apiKey := range rejected {

		t.Run(name, func(t *testing.T) {
			require.Error(t, ValidateAPIKey(apiKey), "must reject: %q", apiKey)
		})
	}
}

// TestValidateAPIKey_TrimsWhitespace confirms a pasted credential with surrounding
// whitespace is accepted
func TestValidateAPIKey_TrimsWhitespace(t *testing.T) {
	require.NoError(t, ValidateAPIKey("  "+testHex+"-us6\n"))
}

// TestValidateAPIKey_DoesNotLeakTheKey confirms a rejection never quotes what it
// rejected
func TestValidateAPIKey_DoesNotLeakTheKey(t *testing.T) {

	secret := testHex + "deadbeef"

	err := ValidateAPIKey(secret + "\n" + "-us6")

	require.Error(t, err)
	require.NotContains(t, err.Error(), secret)
}
