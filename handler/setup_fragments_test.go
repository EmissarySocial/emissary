package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// TestIsHTMXRequest confirms the header sniff: any HX-Request value counts, absence does not.
func TestIsHTMXRequest(t *testing.T) {

	ctx, _ := newSetupTestContext(http.MethodGet, "/domains/000000000000000000000001", "")
	require.False(t, isHTMXRequest(ctx))

	ctx, _ = newSetupTestContext(http.MethodGet, "/domains/000000000000000000000001", "true")
	require.True(t, isHTMXRequest(ctx))
}

// TestSetupFragments_RedirectDirectNavigation confirms that the three setup fragment
// endpoints send a non-htmx (direct navigation) request to their parent page instead of
// serving a scriptless fragment -- the BUG-109 native-submit path. Each handler gets a
// nil factory on purpose: the guard must run before ANY factory access, and this test
// panics if it ever moves later.
func TestSetupFragments_RedirectDirectNavigation(t *testing.T) {

	test := func(name string, handler echo.HandlerFunc, path string, wantLocation string) {
		t.Run(name, func(t *testing.T) {
			ctx, recorder := newSetupTestContext(http.MethodGet, path, "")

			require.NoError(t, handler(ctx))
			require.Equal(t, http.StatusSeeOther, recorder.Code)
			require.Equal(t, wantLocation, recorder.Header().Get("Location"))
		})
	}

	test("domain editor", SetupDomainGet(nil), "/domains/000000000000000000000001", "/domains")
	test("server section", SetupServerGet(nil), "/server/general", "/server")
	test("domain users", SetupDomainUsersGet(nil, nil), "/domains/000000000000000000000001/users", "/domains")
}

// newSetupTestContext builds an echo context for a setup-console request, with the
// HX-Request header set to hxRequest when it is non-empty.
func newSetupTestContext(method string, path string, hxRequest string) (echo.Context, *httptest.ResponseRecorder) {

	request := httptest.NewRequest(method, path, nil)

	if hxRequest != "" {
		request.Header.Set("HX-Request", hxRequest)
	}

	recorder := httptest.NewRecorder()
	return echo.New().NewContext(request, recorder), recorder
}
