package build

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// stubHostnameFactory satisfies the (62-method) Factory interface by embedding it
// (nil) and overriding only Hostname() -- which is all that Host()/URL()/Protocol()
// actually read. Any other Factory call would panic, which is fine: these tests
// never make one.
type stubHostnameFactory struct {
	Factory
	hostname string
}

// Hostname returns the bare domain name of this server. Implements the Builder interface.
func (f stubHostnameFactory) Hostname() string { return f.hostname }

// newTestCommon builds a Common whose factory reports the given hostname and whose
// request targets rawURL (so _request.Host carries the browser's host:port).
func newTestCommon(t *testing.T, rawURL string, factoryHostname string) Common {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	require.Nil(t, err)

	return Common{
		_factory: stubHostnameFactory{hostname: factoryHostname},
		_request: request,
	}
}

// TestCommon_requestPort proves the port is taken from the browser's request (the
// #5 fix), so absolute URLs stay correct behind a proxy/port-map. Standard ports
// (80/443), portless hosts, and a nil request all collapse to "".
func TestCommon_requestPort(t *testing.T) {

	tests := []struct {
		host string
		want string
	}{
		{"localhost:8080", ":8080"},   // dev / Docker 8080:80 -- the reported bug
		{"localhost", ""},             // no explicit port
		{"localhost:80", ""},          // standard HTTP port dropped
		{"example.com", ""},           // prod behind a proxy
		{"example.com:443", ""},       // standard HTTPS port dropped
		{"example.com:8443", ":8443"}, // non-standard public port preserved
		{"[::1]:8080", ":8080"},       // bracketed IPv6 literal with port
		{"", ""},                      // no Host header
	}

	for _, test := range tests {
		w := Common{_request: &http.Request{Host: test.host}}
		require.Equal(t, test.want, w.requestPort(), "host=%q", test.host)
	}

	// A nil request must not panic.
	require.Equal(t, "", Common{}.requestPort())
}

// TestCommon_RelativeURL proves form actions/htmx targets resolve to a root-relative
// reference (no host/port), so the browser posts to whatever origin it is actually on.
func TestCommon_RelativeURL(t *testing.T) {

	w := newTestCommon(t, "http://localhost:8080/@me/settings/general-form?tab=general", "localhost")

	require.Equal(t, "/@me/settings/general-form?tab=general", w.RelativeURL())
}

// TestCommon_Host_and_URL_carryRequestPort proves absolute URLs (permalinks, og:url,
// oembed) carry the port the browser used -- :8080 in dev, none in prod.
func TestCommon_Host_and_URL_carryRequestPort(t *testing.T) {

	// Browser on :8080 (e.g. Docker 8080:80): the port must be present.
	dev := newTestCommon(t, "http://localhost:8080/@me/settings", "localhost")
	require.Equal(t, "http://localhost:8080", dev.Host())
	require.Equal(t, "http://localhost:8080/@me/settings", dev.URL())

	// Production behind a proxy on the standard port: no port in the URL.
	prod := newTestCommon(t, "https://example.com/@me/settings", "example.com")
	require.Equal(t, "https://example.com", prod.Host())
	require.Equal(t, "https://example.com/@me/settings", prod.URL())
}

// TestReportedBug_FormActionOnNonStandardPort ties the fixes back to the original
// report: a form served on http://localhost:8080 must POST back to :8080. It does so
// by rendering a RELATIVE action (browser supplies the origin), never a portless
// absolute URL like the old http://localhost/... that silently hit :80.
func TestReportedBug_FormActionOnNonStandardPort(t *testing.T) {

	w := newTestCommon(t, "http://localhost:8080/@me/settings/general-form", "localhost")

	// The action is relative -- it does not hard-code the host or (wrong) port.
	action := w.RelativeURL()
	require.Equal(t, "/@me/settings/general-form", action)
	require.False(t, strings.Contains(action, "localhost"))

	// WrapForm emits that relative action verbatim as the hx-post target.
	formHTML := WrapForm(action, "", "application/x-www-form-urlencoded")
	require.Contains(t, formHTML, `hx-post="/@me/settings/general-form"`)
	require.NotContains(t, formHTML, "http://localhost")
}
