package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
// The nil Factory is the assertion that no lookup happens: reaching the Locator would panic, so a
// clean 400 proves the guard fires first.
func TestGetWebfinger_MissingResource(t *testing.T) {

	// Parameter absent entirely, and parameter present but empty -- distinct paths in some routers.
	for _, queryString := range []string{"", "?", "?resource=", "?rel=self", "?resource=&rel=self"} {

		err := GetWebfinger(webfingerRequest(queryString), nil, nil)

		require.Error(t, err, "requesting %q", queryString)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(err), "requesting %q", queryString)
	}
}
