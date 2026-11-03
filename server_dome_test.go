package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome"
	"github.com/benpate/digital-dome/dome4echo"
	"github.com/stretchr/testify/require"
)

// testClientIP is the address every request in these tests comes from.
const testClientIP = "203.0.113.7"

// testClientUserAgent is how a typical Mastodon client identifies itself.
const testClientUserAgent = "Tusky/27.0 Android/14 okhttp/4.12.0"

// TestDome_PassesEveryAPIRoute asserts that Dome lets a Mastodon client reach every API, OAuth, and
// discovery route.
func TestDome_PassesEveryAPIRoute(t *testing.T) {

	digitalDome := newTestDome()

	for _, route := range apiRoutes() {
		request := newAPIRequest(route.method, samplePath(route.path))
		require.Nil(t, digitalDome.VerifyRequest(request), "%s %s", route.method, route.path)
	}
}

// TestDome_KeepsAPIClientErrors asserts that a client error from any API, OAuth, or discovery route
// reaches the client unchanged, and never counts against the client's IP.
func TestDome_KeepsAPIClientErrors(t *testing.T) {

	// A soft-blocked pattern inside a route turns these into a scored 403 (BUG-267)
	clientErrors := []error{
		derp.BadRequest("test", "Bad request"),
		derp.Unauthorized("test", "Token expired"),
		derp.NotFound("test", "Not found"),
		derp.Validation("test", "Invalid value"),
	}

	digitalDome := newTestDome()

	for _, route := range apiRoutes() {

		request := newAPIRequest(route.method, samplePath(route.path))

		for _, clientError := range clientErrors {
			returned := digitalDome.HandleError(request, clientError)
			require.Equal(t, derp.ErrorCode(clientError), derp.ErrorCode(returned), "%s %s", route.method, route.path)
		}
	}

	// Hundreds of client errors later, the client still gets in
	require.Nil(t, digitalDome.VerifyRequest(newAPIRequest(http.MethodGet, "/api/v1/timelines/home")))
}

// TestDome_UnimplementedAPICallsAreNotScored asserts that a Mastodon client calling an endpoint
// Emissary does not implement gets a 404 for any method, and keeps its access to the server.
func TestDome_UnimplementedAPICallsAreNotScored(t *testing.T) {

	// Each refusal Dome scores counts against the IP, and six lock it out of the whole server
	digitalDome := newTestDome()
	e := makeTestRoutes()
	e.Pre(dome4echo.New(digitalDome))

	// Call an endpoint nobody built, six times with every method
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for range 6 {
			recorder := httptest.NewRecorder()
			e.ServeHTTP(recorder, newAPIRequest(method, "/api/v1/not-implemented-anywhere/x"))
			require.Equal(t, http.StatusNotFound, recorder.Code, method)
		}
	}

	// The client still gets in afterwards
	require.Nil(t, digitalDome.VerifyRequest(newAPIRequest(http.MethodGet, "/api/v1/timelines/home")))
}

/******************************************
 * Dome Test Helpers
 ******************************************/

// newTestDome returns Emissary's own Digital Dome, resolving every request to testClientIP.
func newTestDome() *dome.Dome {
	return server.NewDigitalDome(func(*http.Request) string { return testClientIP })
}

// apiRoute is one registered method and path that a Mastodon client can call.
type apiRoute struct {
	method string
	path   string
}

// apiRoutes returns every registered route under the Mastodon API, OAuth, and discovery paths.
func apiRoutes() []apiRoute {

	result := make([]apiRoute, 0)

	for _, route := range makeTestRoutes().Routes() {
		for _, prefix := range []string{"/api/", "/oauth/", "/.well-known/", "/nodeinfo/"} {
			if strings.HasPrefix(route.Path, prefix) {
				result = append(result, apiRoute{method: route.Method, path: route.Path})
				break
			}
		}
	}

	return result
}

// newAPIRequest returns a request from a typical Mastodon client.
func newAPIRequest(method string, path string) *http.Request {
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("User-Agent", testClientUserAgent)
	return request
}
