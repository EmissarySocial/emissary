package build

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

/******************************************
 * Navigation
 *
 * These tests pin the one decision that separates an HTTP
 * redirect from an "Hx-Redirect" header. Both Steps used to
 * emit exactly one of the two unconditionally, which made an
 * off-site "redirect-to" silently do nothing under htmx (the
 * XHR followed the redirect, and CORS blocked it) and made a
 * non-htmx "forward-to" land on a blank 200.
 ******************************************/

// testCaller is a partialRequester that reports whichever kind of request a test needs
type testCaller bool

// IsPartialRequest returns TRUE when this caller stands in for an htmx request
func (caller testCaller) IsPartialRequest() bool {
	return bool(caller)
}

const (
	htmxCaller    = testCaller(true)
	browserCaller = testCaller(false)
)

// applied runs a PipelineBehavior and returns the PipelineResult it produced
func applied(behavior PipelineBehavior) PipelineResult {

	result := NewPipelineResult()
	behavior(&result)
	return result
}

func TestIsOffOrigin(t *testing.T) {

	// Relative targets stay on this site, so a fragment swap is always possible
	require.False(t, isOffOrigin("/stream/123"))
	require.False(t, isOffOrigin("/@me/settings?a=1&b=2"))
	require.False(t, isOffOrigin("relative/path"))
	require.False(t, isOffOrigin(""))

	// Absolute http(s) targets name a host of their own
	require.True(t, isOffOrigin("https://instagram.com"))
	require.True(t, isOffOrigin("http://localhost:8080/somewhere"))
	require.True(t, isOffOrigin("https://example.com/oauth?client_id=1&state=abc"))
}

// TestNavigateContent_SameOrigin confirms that the ordinary case is unchanged for both
// kinds of caller: an HTTP redirect, which htmx follows inside its XHR and swaps in place.
func TestNavigateContent_SameOrigin(t *testing.T) {

	for _, caller := range []testCaller{htmxCaller, browserCaller} {

		result := applied(navigateContent(caller, http.StatusTemporaryRedirect, "/@me/newsfeed/feed"))

		require.Equal(t, "/@me/newsfeed/feed", result.Headers["Location"])
		require.Equal(t, http.StatusTemporaryRedirect, result.GetStatusCode())
		require.Empty(t, result.Headers["Hx-Redirect"])
		require.True(t, result.Halt)
		require.True(t, result.FullPage)
	}
}

// TestNavigateContent_OffOriginToHTMX confirms the fix for the Instagram case: htmx cannot
// swap a cross-origin response, so the navigation is handed to htmx instead.
func TestNavigateContent_OffOriginToHTMX(t *testing.T) {

	result := applied(navigateContent(htmxCaller, http.StatusTemporaryRedirect, "https://instagram.com"))

	require.Equal(t, "https://instagram.com", result.Headers["Hx-Redirect"])
	require.Empty(t, result.Headers["Location"])

	// htmx acts on the header, so the status must stay in the range it will accept
	require.Equal(t, http.StatusOK, result.GetStatusCode())
	require.True(t, result.Halt)
}

// TestNavigateContent_OffOriginToBrowser confirms that a plain <a href> still gets a real
// HTTP redirect, which is the only thing a browser outside htmx would act on.
func TestNavigateContent_OffOriginToBrowser(t *testing.T) {

	result := applied(navigateContent(browserCaller, http.StatusTemporaryRedirect, "https://instagram.com"))

	require.Equal(t, "https://instagram.com", result.Headers["Location"])
	require.Equal(t, http.StatusTemporaryRedirect, result.GetStatusCode())
	require.Empty(t, result.Headers["Hx-Redirect"])
}

// TestNavigateDocument_ToHTMX confirms that the htmx path is unchanged, including the
// closeModal event that ends a modal workflow.
func TestNavigateDocument_ToHTMX(t *testing.T) {

	result := applied(navigateDocument(htmxCaller, "/admin/navigation"))

	require.Equal(t, "/admin/navigation", result.Headers["Hx-Redirect"])
	require.Equal(t, "true", result.Events["closeModal"])
	require.Empty(t, result.Headers["Location"])

	// "forward-to" has always let the pipeline continue, and 22 call sites rely on it
	require.False(t, result.Halt)
}

// TestNavigateDocument_ToBrowser confirms the new fallback.  A non-htmx caller ignores
// "Hx-Redirect" entirely, so before this it received a blank 200 and went nowhere.
func TestNavigateDocument_ToBrowser(t *testing.T) {

	result := applied(navigateDocument(browserCaller, "/admin/navigation"))

	require.Equal(t, "/admin/navigation", result.Headers["Location"])
	require.Equal(t, http.StatusSeeOther, result.GetStatusCode())
	require.Empty(t, result.Headers["Hx-Redirect"])
	require.True(t, result.Halt)
	require.True(t, result.FullPage)
}
