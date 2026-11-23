package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAdvertiseIndieAuth verifies that the discovery links are added only to a domain with a primary user
func TestAdvertiseIndieAuth(t *testing.T) {

	const host = "https://example.com"

	// A primary user: one Link header naming all three URLs
	header := http.Header{}
	advertiseIndieAuth(header, host, true)
	require.Equal(t, []string{indieAuthLinks(host)}, header.Values("Link"))

	// No primary user: nothing
	header = http.Header{}
	advertiseIndieAuth(header, host, false)
	require.Empty(t, header.Values("Link"))

	// A Link header already on the response is kept
	header = http.Header{}
	header.Set("Link", `<https://example.com/style.css>; rel="preload"`)
	advertiseIndieAuth(header, host, true)
	require.Equal(t, []string{`<https://example.com/style.css>; rel="preload"`, indieAuthLinks(host)}, header.Values("Link"))
}

// TestIndieAuthLinks verifies the header value, and that it names the URLs the metadata document publishes
func TestIndieAuthLinks(t *testing.T) {

	require.Equal(t,
		`<https://example.com/.well-known/oauth-authorization-server>; rel="indieauth-metadata", `+
			`<https://example.com/oauth/authorize>; rel="authorization_endpoint", `+
			`<https://example.com/oauth/token>; rel="token_endpoint"`,
		indieAuthLinks("https://example.com"))

	for _, host := range []string{"https://example.com", "http://localhost", "http://localhost:8080"} {

		links := parseLinkHeader(t, indieAuthLinks(host))
		metadata := oauthMetadata(host)

		// RULE: Bridgy Fed refuses a login whose `me` advertises different endpoints, so these must match the metadata
		require.Equal(t, metadata["authorization_endpoint"], links["authorization_endpoint"], host)
		require.Equal(t, metadata["token_endpoint"], links["token_endpoint"], host)

		// IndieAuth §4.1.1: the issuer is a prefix of the metadata URL
		require.Equal(t, host+"/.well-known/oauth-authorization-server", links["indieauth-metadata"], host)
		require.True(t, strings.HasPrefix(links["indieauth-metadata"], metadata["issuer"].(string)+"/"), host)
	}
}

// parseLinkHeader maps each rel in a Link header value to its URL
func parseLinkHeader(t *testing.T, value string) map[string]string {

	t.Helper()
	result := map[string]string{}

	for _, link := range strings.Split(value, ", ") {

		target, params, found := strings.Cut(link, ">; ")
		require.True(t, found, "malformed link %q", link)
		require.True(t, strings.HasPrefix(target, "<"), "malformed link %q", link)

		rel, found := strings.CutPrefix(params, "rel=")
		require.True(t, found, "link %q names no rel", link)

		result[strings.Trim(rel, `"`)] = strings.TrimPrefix(target, "<")
	}

	return result
}
