package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// RULE: The SearchQuery actor routes must carry ":searchId" and never ":stream".  They declare no
// stream, which is exactly why they cannot be wrapped in anything that resolves one: getStreamToken
// turns the absent parameter into "home", and WithStream answers a missing home page with a redirect
// to /startup, which then refuses every visitor who is not a domain owner. (BUG-148)
func TestSearchQueryRoutes_DeclareNoStreamParameter(t *testing.T) {

	e := makeTestRoutes()

	found := 0

	for _, route := range e.Routes() {

		if !strings.HasPrefix(route.Path, "/@search_") {
			continue
		}

		found++
		require.Contains(t, route.Path, ":searchId", route.Path)
		require.NotContains(t, route.Path, ":stream", route.Path)
	}

	require.Equal(t, 7, found, "expected every SearchQuery actor route to be checked")
}
