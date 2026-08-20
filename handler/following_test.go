package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// invokeFollowingTunnel runs one request through GetFollowingTunnel and returns the response.
func invokeFollowingTunnel(t *testing.T, queryString string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/.ostatus/tunnel"+queryString, nil)
	recorder := httptest.NewRecorder()
	ctx := echo.New().NewContext(request, recorder)

	require.NoError(t, GetFollowingTunnel(ctx))
	return recorder
}

// TestGetFollowingTunnel verifies that a legacy tunnel link redirects to the "following-edit" settings page
func TestGetFollowingTunnel(t *testing.T) {

	// Legacy tunnel links redirect to the "following-edit" settings page
	response := invokeFollowingTunnel(t, "?uri=https://other.example/@person")

	require.Equal(t, http.StatusFound, response.Code)
	require.Equal(t, "/@me/settings/following-edit?url=https%3A%2F%2Fother.example%2F%40person", response.Header().Get("Location"))
}

// TestGetFollowingTunnel_EmptyURI verifies that a missing "uri" parameter still redirects
func TestGetFollowingTunnel_EmptyURI(t *testing.T) {

	// A missing "uri" parameter still redirects (the settings page shows its empty form)
	response := invokeFollowingTunnel(t, "")

	require.Equal(t, http.StatusFound, response.Code)
	require.Equal(t, "/@me/settings/following-edit?url=", response.Header().Get("Location"))
}

// TestGetFollowingTunnel_EscapesHTML verifies that a hostile "uri" cannot inject markup into the response
func TestGetFollowingTunnel_EscapesHTML(t *testing.T) {

	// RULE: hostile "uri" values are fully escaped, so they cannot inject markup or scripts
	response := invokeFollowingTunnel(t, `?uri=%22%3E%3Cscript%3Ealert(1)%3C/script%3E`)

	require.Equal(t, http.StatusFound, response.Code)
	require.NotContains(t, response.Header().Get("Location"), "<script>")
	require.NotContains(t, response.Body.String(), "<script>")
}

// TestGetFollowingTunnel_EscapesHeaderInjection verifies that CR/LF in the "uri" cannot split the Location header
func TestGetFollowingTunnel_EscapesHeaderInjection(t *testing.T) {

	// RULE: CR/LF in the "uri" value must not split the Location header
	response := invokeFollowingTunnel(t, "?uri=x%0D%0ASet-Cookie:%20pwned=1")

	location := response.Header().Get("Location")
	require.Equal(t, http.StatusFound, response.Code)
	require.False(t, strings.ContainsAny(location, "\r\n"))
	require.Empty(t, response.Header().Get("Set-Cookie"))
}
