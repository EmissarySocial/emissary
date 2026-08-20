package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// invokeCrossOriginProtection runs one request through the middleware and
// returns the result: nil when the request was passed to the next handler,
// or the rejection error.
func invokeCrossOriginProtection(t *testing.T, method string, headers map[string]string) error {
	t.Helper()

	request := httptest.NewRequest(method, "https://emissary.example/@me/settings", nil)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	request.Host = "emissary.example"

	e := echo.New()
	ctx := e.NewContext(request, httptest.NewRecorder())

	passed := func(_ echo.Context) error {
		return nil
	}

	return CrossOriginProtection()(passed)(ctx)
}

// TestCrossOriginProtection_SafeMethods verifies that GET, HEAD, and OPTIONS are always allowed
func TestCrossOriginProtection_SafeMethods(t *testing.T) {

	// Safe methods are always allowed, even when explicitly cross-site
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		err := invokeCrossOriginProtection(t, method, map[string]string{"Sec-Fetch-Site": "cross-site"})
		require.NoError(t, err, "safe method %s must be allowed", method)
	}
}

// TestCrossOriginProtection_ServerToServer verifies that non-browser requests are allowed through
func TestCrossOriginProtection_ServerToServer(t *testing.T) {

	// Requests without Sec-Fetch-Site or Origin headers (ActivityPub, webhooks, curl)
	// are allowed: they are not browser requests, so CSRF does not apply
	err := invokeCrossOriginProtection(t, http.MethodPost, nil)
	require.NoError(t, err)
}

// TestCrossOriginProtection_SameOrigin verifies that same-origin and user-initiated requests are allowed
func TestCrossOriginProtection_SameOrigin(t *testing.T) {

	// Same-origin browser requests are allowed
	err := invokeCrossOriginProtection(t, http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-origin"})
	require.NoError(t, err)

	// User-initiated requests (Sec-Fetch-Site: none) are allowed
	err = invokeCrossOriginProtection(t, http.MethodPost, map[string]string{"Sec-Fetch-Site": "none"})
	require.NoError(t, err)

	// Older browsers: no Sec-Fetch-Site, but an Origin that matches the Host
	err = invokeCrossOriginProtection(t, http.MethodPost, map[string]string{"Origin": "https://emissary.example"})
	require.NoError(t, err)
}

// TestCrossOriginProtection_CrossSite verifies that cross-site and same-site POSTs are rejected
func TestCrossOriginProtection_CrossSite(t *testing.T) {

	// Cross-site POSTs are rejected via Sec-Fetch-Site
	err := invokeCrossOriginProtection(t, http.MethodPost, map[string]string{"Sec-Fetch-Site": "cross-site"})
	require.Error(t, err)
	require.True(t, derp.IsForbidden(err))

	// Subdomains count as "same-site" but still a different origin — rejected
	err = invokeCrossOriginProtection(t, http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-site"})
	require.Error(t, err)
	require.True(t, derp.IsForbidden(err))

	// Older browsers: no Sec-Fetch-Site, and an Origin that does not match the Host
	err = invokeCrossOriginProtection(t, http.MethodPost, map[string]string{"Origin": "https://attacker.example"})
	require.Error(t, err)
	require.True(t, derp.IsForbidden(err))
}

// TestCrossOriginProtection_CrossSiteDelete verifies that every unsafe method is guarded, not just POST
func TestCrossOriginProtection_CrossSiteDelete(t *testing.T) {

	// All non-safe methods are covered, not just POST
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		err := invokeCrossOriginProtection(t, method, map[string]string{"Sec-Fetch-Site": "cross-site"})
		require.Error(t, err, "non-safe method %s must be rejected", method)
		require.True(t, derp.IsForbidden(err))
	}
}
