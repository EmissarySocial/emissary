package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// headExempt lists the GET routes that deliberately refuse HEAD through refuseHead, each with its reason.
var headExempt = map[string]string{
	"/@:userId/export/emissary-stream/:streamId/attachments/:attachmentId/original": "ServeOriginal copies the whole file",
	"/.geocode/network":                         "calls a paid IP-geolocation provider",
	"/.intents/discover":                        "fetches a remote actor the caller names",
	"/.unsplash/photos/:photo":                  "spends Unsplash API quota",
	"/.unsplash/collections/:collection/random": "spends Unsplash API quota",
	"/oauth/clients/:provider/callback":         "exchanges a one-time code, and AllowCSR handles GET alone",
}

// TestRoutes_EveryGETAnswersHEAD asserts that every GET route has a HEAD registration of its own.
func TestRoutes_EveryGETAnswersHEAD(t *testing.T) {

	// A GET-only route does not simply refuse HEAD: its HEAD falls back into whichever param route
	// shares its depth, and is answered by a handler that never serves its GET (BUG-143).
	e := makeTestRoutes()
	headPaths := make(map[string]bool)

	for _, route := range e.Routes() {
		if route.Method == http.MethodHead {
			headPaths[route.Path] = true
		}
	}

	for _, path := range getPaths(e) {
		require.True(t, headPaths[path], "GET route has no HEAD registration: %s", path)
	}
}

// TestRoutes_HEADResolvesToItsOwnRoute asserts that a HEAD request reaches the same route pattern
// that the equivalent GET does, and that the exempt routes refuse it.
func TestRoutes_HEADResolvesToItsOwnRoute(t *testing.T) {

	e := makeTestRoutes()
	refusing := refusingHeadPaths(e)

	for _, path := range getPaths(e) {

		requestPath := samplePath(path)
		require.Equal(t, path, matchedRoute(e, http.MethodHead, requestPath), "HEAD %s", requestPath)

		// RULE: A route refuses HEAD exactly when headExempt lists it, so neither can drift from the other
		_, exempt := headExempt[path]
		require.Equal(t, exempt, refusing[path], "refuseHead registration must match headExempt: %s", path)
	}
}

// TestRoutes_ExemptRoutesRefuseHEAD asserts that each exempt route answers HEAD with a 405 and an
// Allow header, rather than serving it.
func TestRoutes_ExemptRoutesRefuseHEAD(t *testing.T) {

	e := makeTestRoutes()

	for path := range headExempt {

		// The refusing handler touches no Factory, so it is safe to execute here
		requestPath := samplePath(path)
		recorder := httptest.NewRecorder()
		ctx := e.NewContext(httptest.NewRequest(http.MethodHead, requestPath, nil), recorder)
		e.Router().Find(http.MethodHead, requestPath, ctx)

		require.Equal(t, path, ctx.Path(), path)
		require.ErrorIs(t, ctx.Handler()(ctx), echo.ErrMethodNotAllowed, path)
		require.Equal(t, "OPTIONS, GET", recorder.Header().Get(echo.HeaderAllow), path)
	}
}

/******************************************
 * HEAD Test Helpers
 ******************************************/

// getPaths returns the path of every registered GET route.
func getPaths(e *echo.Echo) []string {

	result := make([]string, 0)

	for _, route := range e.Routes() {
		if route.Method == http.MethodGet {
			result = append(result, route.Path)
		}
	}

	return result
}

// samplePath turns a route pattern into a concrete request path, filling each parameter with "x".
func samplePath(pattern string) string {

	segments := strings.Split(pattern, "/")

	for index, segment := range segments {

		// Keep any static prefix, like the "@" in "/@:userId" or "search_" in "/@search_:searchId"
		if colon := strings.Index(segment, ":"); colon >= 0 {
			segments[index] = segment[:colon] + "x"
		}

		segments[index] = strings.ReplaceAll(segments[index], "*", "x")
	}

	return strings.Join(segments, "/")
}

// refusingHeadPaths returns the path of every HEAD route registered with refuseHead.
func refusingHeadPaths(e *echo.Echo) map[string]bool {

	// Echo wraps each handler in its own closure, so match the handler NAME it records, not a pointer
	result := make(map[string]bool)

	for _, route := range e.Routes() {
		if route.Method == http.MethodHead && strings.HasSuffix(route.Name, ".refuseHead") {
			result[route.Path] = true
		}
	}

	return result
}

// matchedRoute returns the pattern of the route that answers the provided method and path, or ""
// if the router answers it with a 404 or 405.
func matchedRoute(e *echo.Echo, method string, path string) string {

	ctx := e.NewContext(httptest.NewRequest(method, path, nil), httptest.NewRecorder())
	e.Router().Find(method, path, ctx)

	// Identify the matched handler by pointer: these handlers close over a nil Factory
	matched := reflect.ValueOf(ctx.Handler()).Pointer()

	if matched == reflect.ValueOf(echo.NotFoundHandler).Pointer() {
		return ""
	}

	if matched == reflect.ValueOf(echo.MethodNotAllowedHandler).Pointer() {
		return ""
	}

	return ctx.Path()
}
