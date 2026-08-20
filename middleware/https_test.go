package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// invokeHttpsRedirect runs one request through the middleware and returns the
// response recorder plus a flag indicating whether the request was passed to
// the next handler (true) or short-circuited by the middleware (false).
func invokeHttpsRedirect(t *testing.T, scheme string, host string) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "http://"+host+"/path", nil)
	request.Host = host

	// echo.Scheme() reports "https" from X-Forwarded-Proto (httptest does not
	// populate request.TLS), which also mirrors Emissary's real deployment
	// behind a TLS-terminating reverse proxy.
	if scheme == "https" {
		request.Header.Set(echo.HeaderXForwardedProto, "https")
	}

	e := echo.New()
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)

	passed := false
	next := func(_ echo.Context) error {
		passed = true
		return nil
	}

	err := HttpsRedirect(next)(ctx)
	require.NoError(t, err)

	return recorder, passed
}

// TestHttpsRedirect_InsecurePublic verifies that an insecure public request is redirected instead of handled
func TestHttpsRedirect_InsecurePublic(t *testing.T) {

	// An insecure public request is permanently redirected to its HTTPS URL,
	// and does not reach the handler
	recorder, passed := invokeHttpsRedirect(t, "http", "emissary.example")
	require.NotNil(t, recorder)

	require.False(t, passed, "insecure public request must be short-circuited")
	require.Equal(t, http.StatusPermanentRedirect, recorder.Code)
	require.Equal(t, "https://emissary.example/path", recorder.Header().Get("Location"))
	require.Empty(t, recorder.Header().Get("Strict-Transport-Security"))
}

// TestHttpsRedirect_SecurePublic verifies that a secure public request reaches the handler and gets an HSTS header
func TestHttpsRedirect_SecurePublic(t *testing.T) {

	// A secure public request passes through to the handler with the HSTS header set
	recorder, passed := invokeHttpsRedirect(t, "https", "emissary.example")
	require.NotNil(t, recorder)

	require.True(t, passed, "secure public request must reach the handler")
	require.Equal(t, "max-age=63072000", recorder.Header().Get("Strict-Transport-Security"))
}

// TestHttpsRedirect_LocalHosts verifies that local and private hosts are exempt from the redirect and HSTS
func TestHttpsRedirect_LocalHosts(t *testing.T) {

	// Local, loopback, and private hosts are exempt from both the redirect and
	// HSTS, on either scheme — this keeps development environments working
	localHosts := []string{
		"localhost",
		"localhost:8080",
		"emissary.local",
		"myserver.internal",
		"127.0.0.1",
		"192.168.1.10",
		"10.0.0.5",
		"[::1]",
	}

	for _, scheme := range []string{"http", "https"} {
		for _, host := range localHosts {
			recorder, passed := invokeHttpsRedirect(t, scheme, host)
			require.NotNil(t, recorder)

			require.True(t, passed, "%s://%s must pass through untouched", scheme, host)
			require.NotEqual(t, http.StatusPermanentRedirect, recorder.Code, "%s://%s must not be redirected", scheme, host)
			require.Empty(t, recorder.Header().Get("Strict-Transport-Security"), "%s://%s must not receive HSTS", scheme, host)
		}
	}
}
