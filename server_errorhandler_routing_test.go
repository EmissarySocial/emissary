package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/derp/plugins"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// TestIsRoutingError verifies that only Echo's own routing failures qualify, however they are wrapped.
func TestIsRoutingError(t *testing.T) {

	require.True(t, isRoutingError(echo.ErrNotFound))
	require.True(t, isRoutingError(echo.ErrMethodNotAllowed))

	// dome4echo rewraps Echo's error with a derp code, and keeps the original in the chain
	require.True(t, isRoutingError(derp.Wrap(echo.ErrMethodNotAllowed, "dome4echo.withStatusCode", "Method Not Allowed", derp.WithCode(http.StatusMethodNotAllowed))))

	// A handler's own "not found" is an application answer, and stays reportable
	require.False(t, isRoutingError(derp.NotFound("test", "User not found")))
	require.False(t, isRoutingError(echo.NewHTTPError(http.StatusNotFound)))
	require.False(t, isRoutingError(nil))
}

// TestErrorHandler_DoesNotReportRoutingErrors verifies that a routing failure still reaches the
// client with its status code, but is not reported to the ErrorLog.
func TestErrorHandler_DoesNotReportRoutingErrors(t *testing.T) {

	reporter := &capturingReporter{}
	derp.SetPlugins(reporter)
	defer derp.SetPlugins(plugins.JSON{}) // derp's own init() default

	for _, routingError := range []*echo.HTTPError{echo.ErrNotFound, echo.ErrMethodNotAllowed} {

		request := httptest.NewRequest(http.MethodPost, "https://example.com/", nil)
		recorder := httptest.NewRecorder()

		errorHandler(routingError, echo.New().NewContext(request, recorder))

		require.Equal(t, routingError.Code, recorder.Code)
	}

	require.Empty(t, reporter.reported)
}

// TestErrorHandler_ReportsApplicationNotFound verifies that a 404 raised by a handler is still reported.
func TestErrorHandler_ReportsApplicationNotFound(t *testing.T) {

	reporter := &capturingReporter{}
	derp.SetPlugins(reporter)
	defer derp.SetPlugins(plugins.JSON{}) // derp's own init() default

	request := httptest.NewRequest(http.MethodGet, "https://example.com/@missing", nil)
	errorHandler(derp.NotFound("test", "User not found"), echo.New().NewContext(request, httptest.NewRecorder()))

	require.Len(t, reporter.reported, 1)
}
