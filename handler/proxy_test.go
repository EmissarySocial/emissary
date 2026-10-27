package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// visitorProxyRequest builds an anonymous /.proxy POST, with no sign-in cookie, asking for the provided id
func visitorProxyRequest(id string) *steranko.Context {

	form := url.Values{"id": {id}}
	request := httptest.NewRequest(http.MethodPost, "/.proxy", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	factory := &service.Factory{}
	return factory.Steranko(unusedSession{}).Context(echo.New().NewContext(request, httptest.NewRecorder()))
}

// TestPostProxyURL_VisitorBadURL verifies that a visitor's bad URL is Forbidden, which Digital Dome counts
func TestPostProxyURL_VisitorBadURL(t *testing.T) {

	// A zero Factory and unusedSession prove the request is refused before anything is loaded
	for _, id := range []string{"", "not-a-url", "http://127.0.0.1/users/alice"} {
		err := PostProxyURL(visitorProxyRequest(id), &service.Factory{}, unusedSession{})

		// RULE: Digital Dome counts only 403s, so a validation error must not keep its own code
		require.Equal(t, http.StatusForbidden, derp.ErrorCode(err), "requesting %q", id)
	}
}

// applyVisitorHeaders runs the visitor-header option against a new request, and returns its headers
func applyVisitorHeaders(t *testing.T, clientIP string, userAgent string) http.Header {

	t.Helper()

	request, err := http.NewRequest(http.MethodGet, "https://mastodon.social/users/alice", nil)
	require.NoError(t, err)
	request.Header.Set("User-Agent", "Emissary (+https://example.social)")

	option := proxyVisitorHeaders(clientIP, userAgent)
	require.Nil(t, option.ModifyRequest(nil, request))

	return request.Header
}

// TestProxyVisitorHeaders_IPv4 verifies the forwarded headers for an IPv4 visitor
func TestProxyVisitorHeaders_IPv4(t *testing.T) {

	header := applyVisitorHeaders(t, "192.0.2.1", "Mozilla/5.0 (Macintosh)")

	require.Equal(t, `for="192.0.2.1"`, header.Get("Forwarded"))
	require.Equal(t, "192.0.2.1", header.Get("X-Forwarded-For"))
	require.Equal(t, "Mozilla/5.0 (Macintosh)", header.Get("X-Forwarded-User-Agent"))

	// RULE: Emissary's own User-Agent stays, because it names the signing actor
	require.Equal(t, "Emissary (+https://example.social)", header.Get("User-Agent"))
}

// TestProxyVisitorHeaders_IPv6 verifies that an IPv6 address is bracketed in the Forwarded header
func TestProxyVisitorHeaders_IPv6(t *testing.T) {

	header := applyVisitorHeaders(t, "2001:db8::1", "")

	require.Equal(t, `for="[2001:db8::1]"`, header.Get("Forwarded"))
	require.Equal(t, "2001:db8::1", header.Get("X-Forwarded-For"))
	require.Empty(t, header.Values("X-Forwarded-User-Agent"))
}

// TestProxyVisitorHeaders_NotAnAddress verifies that a value that is not an IP address is never forwarded
func TestProxyVisitorHeaders_NotAnAddress(t *testing.T) {

	for _, clientIP := range []string{"", "not-an-ip", `192.0.2.1", for="203.0.113.9`, "192.0.2.1\r\nX-Injected: 1"} {
		header := applyVisitorHeaders(t, clientIP, "Mozilla/5.0")
		require.Empty(t, header.Values("Forwarded"), clientIP)
		require.Empty(t, header.Values("X-Forwarded-For"), clientIP)
	}
}

// TestProxyVisitorHeaders_LongUserAgent verifies that the browser description is capped, on a rune boundary
func TestProxyVisitorHeaders_LongUserAgent(t *testing.T) {

	header := applyVisitorHeaders(t, "192.0.2.1", strings.Repeat("é", proxyUserAgentLimit))
	forwarded := header.Get("X-Forwarded-User-Agent")

	require.LessOrEqual(t, len(forwarded), proxyUserAgentLimit)
	require.Equal(t, strings.Repeat("é", proxyUserAgentLimit/2), forwarded)
}
