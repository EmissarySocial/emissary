package server

import (
	"net/http"
	"slices"

	"github.com/benpate/digital-dome/dome"
)

// NewDigitalDome returns the Digital Dome that guards every domain, with Emissary's logged status
// codes, scored status codes, and soft-blocked paths.
func NewDigitalDome(clientIP dome.ClientIPResolver) *dome.Dome {

	return dome.New(
		clientIP, // resolve client IPs using the configured trusted-proxy strategy
		dome.SoftBlockPaths(softBlockedPaths()...),
		dome.LogStatusCodes(
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusMethodNotAllowed,
			http.StatusInternalServerError,
		),
		dome.BlockStatusCodes( // replaces Dome's default list, so 403 is restated here
			http.StatusForbidden,
			http.StatusMethodNotAllowed, // safe only because every GET route also answers HEAD (BUG-143)
		),
	)
}

// softBlockedPaths returns Dome's suspicious paths, less the ones that occur inside Emissary's own
// API routes.
func softBlockedPaths() []string {

	// RULE: Dome matches these anywhere in a path (BUG-267), so "/app" would score every client
	// error on the Mastodon API's /api/v1/apps routes as a probe, and lock out the client's IP
	return slices.DeleteFunc(slices.Clone(dome.SuspiciousPaths), func(path string) bool {
		return path == "/app"
	})
}
