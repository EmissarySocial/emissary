package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// TestErrorHandler_EchoHTTPError confirms that Echo's own routing errors
// (*echo.HTTPError) are surfaced with their real status code instead of being
// flattened to a generic 500. See BUG-003209 family: unsupported HTTP methods
// (PUT/PATCH/TRACE) previously returned 500 instead of 405.
func TestErrorHandler_EchoHTTPError(t *testing.T) {

	testCases := []struct {
		name       string
		err        error
		expectCode int
	}{
		{"MethodNotAllowed", echo.ErrMethodNotAllowed, http.StatusMethodNotAllowed},
		{"NotFound", echo.ErrNotFound, http.StatusNotFound},
		{"BadRequest", echo.ErrBadRequest, http.StatusBadRequest},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			e := echo.New()
			request := httptest.NewRequest(http.MethodPut, "/@qatester", nil)
			request.Host = "example.com" // Non-local host so we hit the plain (non-dump) response path
			recorder := httptest.NewRecorder()
			ctx := e.NewContext(request, recorder)

			errorHandler(testCase.err, ctx)

			require.Equal(t, testCase.expectCode, recorder.Code)
		})
	}
}

// TestErrorHandler_SignedRequestIsNotRedirected pins the second half of BUG-20. Refusing an invalid
// signature only helps if the refusal REACHES the peer: a 303 to /signin is meaningless to a machine
// and reads as success, which would re-hide the failure the 401 exists to surface.
func TestErrorHandler_SignedRequestIsNotRedirected(t *testing.T) {

	testCases := []struct {
		name       string
		signature  string
		accept     string
		expectCode int
	}{
		// The case the redirect used to swallow: signed, but with no ActivityPub Accept header, so
		// handleActivityPubError declines it. A peer with a misconfigured Accept is exactly the
		// population BUG-20 is meant to diagnose.
		{"SignedWithoutAcceptHeader", `keyId="https://remote.example/@alice#main-key",signature="AAAA"`, "", http.StatusUnauthorized},

		// Signed AND asking for ActivityPub: already handled as JSON, and must stay that way.
		{"SignedActivityPub", `keyId="https://remote.example/@alice#main-key",signature="AAAA"`, "application/activity+json", http.StatusUnauthorized},

		// A browser sends no Signature, so it still gets the sign-in redirect. Must not regress.
		{"UnsignedBrowser", "", "text/html", http.StatusSeeOther},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			e := echo.New()
			request := httptest.NewRequest(http.MethodGet, "/@bob/pub/objects/123", nil)
			request.Host = "example.com" // Non-local host so we hit the plain (non-dump) response path

			if testCase.signature != "" {
				request.Header.Set("Signature", testCase.signature)
			}

			if testCase.accept != "" {
				request.Header.Set("Accept", testCase.accept)
			}

			recorder := httptest.NewRecorder()
			ctx := e.NewContext(request, recorder)

			errorHandler(derp.Unauthorized("test", "Invalid HTTP Signature"), ctx)

			require.Equal(t, testCase.expectCode, recorder.Code)
		})
	}
}
