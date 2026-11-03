package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// TestErrorHandler_EchoHTTPError confirms that Echo's own routing errors (*echo.HTTPError) are
// surfaced with their real status code instead of being flattened to a generic 500.
func TestErrorHandler_EchoHTTPError(t *testing.T) {

	// The BUG-003209 family: unsupported methods (PUT/PATCH/TRACE) answered 500, not 405.
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

// TestErrorHandler_SignedRequestIsNotRedirected pins the second half of BUG-20: a refusal only
// helps if it REACHES the peer, and a 303 to /signin reads as success to a machine.
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

// TestRegisterWithHEAD_MirrorsEachGET pins that every GET route added inside registerWithHEAD gains
// a HEAD route with the same path, handler, and middleware, and that other methods do not.
func TestRegisterWithHEAD_MirrorsEachGET(t *testing.T) {

	// Register one GET route with its own middleware, and one POST route
	e := echo.New()
	middlewareRuns := 0

	countMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			middlewareRuns++
			return next(ctx)
		}
	}

	registerWithHEAD(e, func() {
		e.GET("/api/v1/things/:id", func(ctx echo.Context) error { return ctx.String(http.StatusOK, "thing") }, countMiddleware)
		e.POST("/api/v1/things", func(ctx echo.Context) error { return ctx.NoContent(http.StatusCreated) })
	})

	// Only the GET route gains a HEAD
	methods := make(map[string]bool)

	for _, route := range e.Routes() {
		methods[route.Method+" "+route.Path] = true
	}

	require.True(t, methods["HEAD /api/v1/things/:id"])
	require.False(t, methods["HEAD /api/v1/things"], "only GET routes gain a HEAD")

	// The HEAD route answers through the GET route's own handler and middleware
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/api/v1/things/42", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, middlewareRuns)
}

// TestRegisterWithHEAD_RestoresTheHook pins that registerWithHEAD passes each route to a hook that was
// already installed, and puts that hook back when it finishes.
func TestRegisterWithHEAD_RestoresTheHook(t *testing.T) {

	e := echo.New()
	seen := make([]string, 0)

	e.OnAddRouteHandler = func(_ string, route echo.Route, _ echo.HandlerFunc, _ []echo.MiddlewareFunc) {
		seen = append(seen, route.Method+" "+route.Path)
	}

	registerWithHEAD(e, func() {
		e.GET("/api/v1/things", func(ctx echo.Context) error { return ctx.NoContent(http.StatusOK) })
	})

	require.NotNil(t, e.OnAddRouteHandler)
	require.Equal(t, []string{"GET /api/v1/things", "HEAD /api/v1/things"}, seen)
}
