package main

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/rosetta/schema/format"
	"github.com/stretchr/testify/require"
)

// TestRoutes_ReservedStreamTokens asserts that service.ReservedStreamTokens names exactly the first
// path segments of the application routes that a Stream token could spell.
func TestRoutes_ReservedStreamTokens(t *testing.T) {

	e := makeTestRoutes()
	reserved := service.ReservedStreamTokens()
	segments := map[string]bool{}

	// Collect each route's first segment that is a valid token, which skips params, handles, and dotted files
	for _, route := range e.Routes() {

		segment, _, _ := strings.Cut(strings.TrimPrefix(route.Path, "/"), "/")

		if segment == "" {
			continue
		}

		if _, err := format.Token("")(segment); err != nil {
			continue
		}

		segments[segment] = true
	}

	// RULE: Every route a token could shadow is reserved
	for segment := range segments {
		require.Contains(t, reserved, segment, "/%s is a route, so add %q to reservedStreamTokens", segment, segment)
	}

	// RULE: Every reserved token is still a route, so the list cannot rot
	for _, token := range reserved {
		require.True(t, segments[token], "%q is no longer a route, so remove it from reservedStreamTokens", token)
	}
}
