package server

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome"
	"github.com/stretchr/testify/require"
)

// TestNewDigitalDome_ScoresSoftPathsButNotTheAppsRoute pins both halves of the soft-block policy: a
// missing file on a suspicious path still becomes a scored 403, and a client error on /api/v1/apps does not.
func TestNewDigitalDome_ScoresSoftPathsButNotTheAppsRoute(t *testing.T) {

	digitalDome := NewDigitalDome(func(*http.Request) string { return "203.0.113.9" })

	// A Mastodon client's refused app registration keeps its own status
	apps := httptest.NewRequest(http.MethodPost, "/api/v1/apps", nil)
	refused := digitalDome.HandleError(apps, derp.BadRequest("test", "Missing redirect_uris"))
	require.Equal(t, http.StatusBadRequest, derp.ErrorCode(refused))

	// A probe for a site backup is still scored as one
	backup := httptest.NewRequest(http.MethodGet, "/backup/site.zip", nil)
	probed := digitalDome.HandleError(backup, derp.NotFound("test", "Not found"))
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(probed))
}

// TestSoftBlockedPaths_DropsOnlyApp pins that Emissary soft-blocks every path Dome suggests, except "/app".
func TestSoftBlockedPaths_DropsOnlyApp(t *testing.T) {

	result := softBlockedPaths()

	for _, path := range dome.SuspiciousPaths {

		if path == "/app" {
			require.NotContains(t, result, path)
			continue
		}

		require.Contains(t, result, path)
	}
}

// TestSoftBlockedPaths_LeavesDomesListAlone pins that building the list never writes into Dome's
// package-level SuspiciousPaths, which every Dome in the process shares.
func TestSoftBlockedPaths_LeavesDomesListAlone(t *testing.T) {

	before := slices.Clone(dome.SuspiciousPaths)

	// Build the list in sequence, then from several goroutines at once, for the race detector
	softBlockedPaths()

	var waitGroup sync.WaitGroup

	for range 4 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			softBlockedPaths()
		}()
	}

	waitGroup.Wait()

	require.Equal(t, before, dome.SuspiciousPaths)
}
