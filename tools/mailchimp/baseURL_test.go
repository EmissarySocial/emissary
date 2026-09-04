package mailchimp

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBaseURL_Valid confirms that a well-formed data center produces the documented
// API root
func TestBaseURL_Valid(t *testing.T) {

	testCases := map[string]string{
		"us6":  "https://us6.api.mailchimp.com/3.0",
		"us21": "https://us21.api.mailchimp.com/3.0",
		" us6": "https://us6.api.mailchimp.com/3.0",
	}

	for dataCenter, expected := range testCases {

		result, err := BaseURL(dataCenter)

		require.NoError(t, err)
		require.Equal(t, expected, result)
	}
}

// TestBaseURL_SSRF confirms that no crafted data center can steer an outbound
// request away from Mailchimp
func TestBaseURL_SSRF(t *testing.T) {

	// A regression here is server-side request forgery from Emissary's own network
	// position, which is usually inside a private network.

	attacks := map[string]string{
		"another host entirely": "evil.example.com",
		"host with path":        "evil.example.com/x#",
		"subdomain prefix":      "us6.evil.example.com",
		"userinfo separator":    "us6@evil.example.com",
		"port":                  "us6:8080",
		"path traversal":        "us6/../../admin",
		"leading slashes":       "//evil.example.com",
		"loopback address":      "127.0.0.1",
		"link-local metadata":   "169.254.169.254",
		"scheme injection":      "us6http://evil.example.com",
		"embedded newline":      "us6\nHost: evil.example.com",
		"embedded null":         "us6\x00",
		"embedded space":        "us6 evil.example.com",
		"uppercase":             "US6",
		"unicode homoglyph":     "us６",
		"hyphenated":            "us-6",
		"underscore":            "us_6",
		"empty string":          "",
		"whitespace only":       "   ",
		"too short":             "u",
		"too long":              strings.Repeat("a", 33),
	}

	for name, dataCenter := range attacks {

		t.Run(name, func(t *testing.T) {

			result, err := BaseURL(dataCenter)

			require.Error(t, err, "must reject: %q", dataCenter)
			require.Empty(t, result, "must not return a URL alongside an error")
		})
	}
}

// TestBaseURL_ScaryButHarmless documents values that read like an attack but are not
func TestBaseURL_ScaryButHarmless(t *testing.T) {

	// The mitigation is not a blocklist of frightening words: the value can only become
	// ONE label under api.mailchimp.com, so "localhost" yields localhost.api.mailchimp.com.
	// Do not "harden" the pattern against these -- it would reject a future data center.

	for _, dataCenter := range []string{"localhost", "internal", "admin", "metadata"} {

		result, err := BaseURL(dataCenter)

		require.NoError(t, err)
		require.Equal(t, "https://"+dataCenter+".api.mailchimp.com/3.0", result)
	}
}

// TestBaseURL_HostIsAlwaysMailchimp asserts that any successful result addresses a
// mailchimp.com host over HTTPS
func TestBaseURL_HostIsAlwaysMailchimp(t *testing.T) {

	// The table above catches known attack shapes; this catches the ones nobody listed.

	inputs := []string{
		"us6",
		"us21",
		"evil.example.com",
		"us6@evil.example.com",
		"us6/../admin",
		"//evil.example.com",
		"localhost",
		"",
	}

	for _, dataCenter := range inputs {

		result, err := BaseURL(dataCenter)

		if err != nil {
			continue
		}

		parsed, parseErr := url.Parse(result)

		require.NoError(t, parseErr, "a returned base URL must parse")
		require.Equal(t, "https", parsed.Scheme)
		require.True(t, strings.HasSuffix(parsed.Host, ".api.mailchimp.com"), "host was %q", parsed.Host)
		require.Empty(t, parsed.User, "a base URL must carry no userinfo")
		require.Equal(t, "/3.0", parsed.Path)
	}
}

// TestValidateDataCenter_Accepts confirms that legitimate values are not refused
func TestValidateDataCenter_Accepts(t *testing.T) {

	for _, dataCenter := range []string{"us6", "us21", "us1", "ab", strings.Repeat("a", 32)} {
		require.NoError(t, ValidateDataCenter(dataCenter), "must accept: %q", dataCenter)
	}
}

// TestValidateDataCenter_ErrorNamesWhereToFindIt confirms the empty-value message
// tells the User where to look
func TestValidateDataCenter_ErrorNamesWhereToFindIt(t *testing.T) {

	err := ValidateDataCenter("")

	require.Error(t, err)
	require.Contains(t, err.Error(), "admin.mailchimp.com")
}
