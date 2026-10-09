package build

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

/******************************************
 * Client Fingerprint
 *
 * These accessors are the only path by which a visitor's IP
 * and browser headers reach a Template, and the contact form
 * copies all of them into an email that leaves the server.
 * The tests below pin the three properties that makes safe:
 * the IP comes from the configured strategy rather than from
 * RemoteAddr, no accessor can reach a credential header, and
 * nothing a client sends can be unbounded in length.
 ******************************************/

// stubClientIPFactory satisfies the Factory interface by embedding it (nil) and overriding only
// ClientIP, standing in for the server's configured trusted-proxy strategy.  It answers with the
// last X-Forwarded-For entry, which is what a RIGHTMOST-TRUSTED-COUNT deployment does -- the point
// being that it is NOT RemoteAddr, so a test can tell the two apart.
type stubClientIPFactory struct {
	Factory
}

// ClientIP resolves the request's client IP. Implements the Factory interface.
func (f stubClientIPFactory) ClientIP(request *http.Request) string {

	if forwarded := request.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}

	return strings.TrimSuffix(request.RemoteAddr, ":1234")
}

// newClientCommon builds a Common carrying a request with the provided headers
func newClientCommon(headers map[string]string) Common {

	request := httptest.NewRequest(http.MethodPost, "/000000000000000000000000/submit", nil)
	request.RemoteAddr = "10.0.0.1:1234"

	for name, value := range headers {
		request.Header.Set(name, value)
	}

	return Common{
		_factory: stubClientIPFactory{},
		_request: request,
	}
}

// TestCommon_ClientIP proves the IP comes from the server's configured strategy and not from
// RemoteAddr.  Behind a reverse proxy RemoteAddr is the proxy itself, identical for every
// visitor, so a footer built on it would attribute every message on the site to one address.
func TestCommon_ClientIP(t *testing.T) {

	// With no forwarding header, the strategy falls back to the connecting address
	require.Equal(t, "10.0.0.1", newClientCommon(nil).ClientIP())

	// With one, the strategy's answer wins over RemoteAddr
	builder := newClientCommon(map[string]string{"X-Forwarded-For": "203.0.113.42, 198.51.100.7"})
	require.Equal(t, "198.51.100.7", builder.ClientIP())
}

// TestCommon_ClientIP_NilRequest proves the accessors tolerate a Builder assembled outside a
// handler.  Several steps build one that way, and a panic here would take down a request that
// merely rendered a page containing the accessor.
func TestCommon_ClientIP_NilRequest(t *testing.T) {

	builder := Common{_factory: stubClientIPFactory{}}

	require.Equal(t, "", builder.ClientIP())
	require.Equal(t, "", builder.ClientUserAgent())
	require.Equal(t, "", builder.ClientDescription())
	require.Equal(t, "", builder.ClientReferer())
}

// TestCommon_ClientHeaders proves each accessor reads the header it names.  A silent mix-up here
// would be invisible: every value is a plausible-looking string in a footer nobody cross-checks.
func TestCommon_ClientHeaders(t *testing.T) {

	builder := newClientCommon(map[string]string{
		"User-Agent":         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15",
		"Accept":             "text/html,application/xhtml+xml",
		"Accept-Language":    "en-US,en;q=0.9",
		"Accept-Encoding":    "gzip, deflate, br",
		"Sec-CH-UA":          `"Chromium";v="128", "Not;A=Brand";v="24"`,
		"Sec-CH-UA-Platform": `"macOS"`,
		"Sec-CH-UA-Mobile":   "?0",
		"DNT":                "1",
		"Sec-GPC":            "1",
		"Referer":            "https://example.com/contact",
	})

	require.Equal(t, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15", builder.ClientUserAgent())
	require.Equal(t, "text/html,application/xhtml+xml", builder.ClientAccept())
	require.Equal(t, "en-US,en;q=0.9", builder.ClientAcceptLanguage())
	require.Equal(t, "gzip, deflate, br", builder.ClientAcceptEncoding())
	require.Equal(t, `"Chromium";v="128", "Not;A=Brand";v="24"`, builder.ClientBrands())
	require.Equal(t, `"macOS"`, builder.ClientPlatform())
	require.Equal(t, "?0", builder.ClientMobile())
	require.Equal(t, "1", builder.ClientDoNotTrack())
	require.Equal(t, "1", builder.ClientPrivacyControl())
	require.Equal(t, "https://example.com/contact", builder.ClientReferer())

	// An absent header reads as absent, so the email omits its row rather than showing a blank
	empty := newClientCommon(nil)
	require.Equal(t, "", empty.ClientUserAgent())
	require.Equal(t, "", empty.ClientAcceptLanguage())
	require.Equal(t, "", empty.ClientBrands())
}

// TestCommon_ClientDescription proves the sniffed summary is produced for a real User-Agent and
// suppressed for a missing one.  sniff answers every string, so it maps "" onto
// "Unrecognized Device / Unknown" -- a confident-looking claim about a client that said nothing.
func TestCommon_ClientDescription(t *testing.T) {

	builder := newClientCommon(map[string]string{
		"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Mobile/15E148 Safari/604.1",
	})

	require.Equal(t, "iPhone / Safari", builder.ClientDescription())

	require.Equal(t, "", newClientCommon(nil).ClientDescription())
}

// TestCommon_ClientHeadersAreBounded proves a hostile client cannot dictate the size of the email
// we send.  Go's HTTP server caps the total header block at ~1MB but no single header, so an
// unbounded accessor would copy a megabyte of User-Agent into a message the owner has to open.
//
// The value is TRUNCATED rather than rejected, which is the opposite of the rule that governs the
// visitor's message (CONTACT-FORM D10): that rule protects content, where silent shortening is
// unrecoverable, while discarding a whole submission over a long header is the false positive
// FORM-SPAM-PREVENTION D3 forbids.
func TestCommon_ClientHeadersAreBounded(t *testing.T) {

	builder := newClientCommon(map[string]string{
		"User-Agent":      strings.Repeat("A", 10000),
		"Accept-Language": strings.Repeat("B", 10000),
	})

	userAgent := builder.ClientUserAgent()

	require.Equal(t, clientHeaderMaxLength+1, len([]rune(userAgent)), "the value must be cut to the bound, plus the ellipsis that marks the cut")
	require.True(t, strings.HasSuffix(userAgent, "…"), "a shortened value must not be readable as the whole one")
	require.Equal(t, clientHeaderMaxLength+1, len([]rune(builder.ClientAcceptLanguage())), "every accessor funnels through the same bound")

	// A value at exactly the bound is not marked, because it was not cut
	exact := newClientCommon(map[string]string{"User-Agent": strings.Repeat("C", clientHeaderMaxLength)})
	require.Equal(t, strings.Repeat("C", clientHeaderMaxLength), exact.ClientUserAgent())
}

// TestCommon_ClientHeadersCountRunes proves the bound counts runes, not bytes, so a multi-byte
// character is never cut in half into invalid UTF-8 that a mail client would render as garbage.
func TestCommon_ClientHeadersCountRunes(t *testing.T) {

	// Each rune here is three bytes, so a byte-counting bound would cut mid-character
	builder := newClientCommon(map[string]string{"Accept-Language": strings.Repeat("日", 1000)})

	result := builder.ClientAcceptLanguage()

	require.Equal(t, clientHeaderMaxLength+1, len([]rune(result)))
	require.True(t, strings.HasPrefix(result, strings.Repeat("日", clientHeaderMaxLength)))
}

// TestCommon_ClientHeadersCannotReachCredentials pins the reason this section is a CLOSED set of
// accessors.  Cookie and Authorization are request headers exactly like the ones above, and a
// send-email step renders whatever a template asks for into a message body whose recipient is
// configured per-page -- so a general ".RequestHeader" would turn one template line into session
// token exfiltration.  If a future edit adds one, this test is what should fail.
func TestCommon_ClientHeadersCannotReachCredentials(t *testing.T) {

	builder := newClientCommon(map[string]string{
		"Cookie":        "session=super-secret-token",
		"Authorization": "Bearer super-secret-token",
	})

	// Every exported accessor in this section, called for its value
	values := []string{
		builder.ClientIP(),
		builder.ClientDescription(),
		builder.ClientUserAgent(),
		builder.ClientAccept(),
		builder.ClientAcceptLanguage(),
		builder.ClientAcceptEncoding(),
		builder.ClientBrands(),
		builder.ClientPlatform(),
		builder.ClientMobile(),
		builder.ClientDoNotTrack(),
		builder.ClientPrivacyControl(),
		builder.ClientReferer(),
	}

	for _, value := range values {
		require.NotContains(t, value, "super-secret-token", "no client accessor may return a credential header")
	}
}

// TestTruncateRunes covers the helper's edges directly, since every accessor above depends on it
func TestTruncateRunes(t *testing.T) {

	require.Equal(t, "", truncateRunes("", 10))
	require.Equal(t, "short", truncateRunes("short", 10))
	require.Equal(t, "exactlyten", truncateRunes("exactlyten", 10))
	require.Equal(t, "exactlyten…", truncateRunes("exactlyten!", 10))

	// A zero bound keeps nothing but still marks that something was removed
	require.Equal(t, "…", truncateRunes("anything", 0))
}
