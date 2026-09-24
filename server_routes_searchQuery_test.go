package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// RULE: The SearchQuery actor routes must carry ":searchId" and never ":stream".  A SearchQuery has
// no Stream, so these routes cannot be wrapped in anything that resolves one. (BUG-148)
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
