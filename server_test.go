package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
