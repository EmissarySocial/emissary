package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// webfingerRequest builds a steranko.Context for one WebFinger request.
func webfingerRequest(queryString string) *steranko.Context {

	request := httptest.NewRequest(http.MethodGet, "/.well-known/webfinger"+queryString, nil)
	recorder := httptest.NewRecorder()

	return &steranko.Context{Context: echo.New().NewContext(request, recorder)}
}

// TestGetWebfinger_MissingResource is the regression guard for BUG-18: RFC 7033 Section 4.2 requires a
// 400 when the `resource` parameter is missing.  Before the fix, the empty string was resolved as a
// local account and a bare `GET /.well-known/webfinger` answered with the Application actor's JRD.
//
// The zero Factory and unusedSession are the assertion that no lookup happens: past the guard, the
// empty Locator answers 404 and the session panics on any query, so only the guard answers 400.
func TestGetWebfinger_MissingResource(t *testing.T) {

	// Parameter absent entirely, and parameter present but empty -- distinct paths in some routers.
	for _, queryString := range []string{"", "?", "?resource=", "?rel=self", "?resource=&rel=self"} {

		err := GetWebfinger(webfingerRequest(queryString), &service.Factory{}, unusedSession{})

		require.Error(t, err, "requesting %q", queryString)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(err), "requesting %q", queryString)
	}
}
