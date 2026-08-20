package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestViewFeed verifies that a "view-feed" step parses its configuration
func TestViewFeed(t *testing.T) {
	step, err := NewViewFeed(mapof.Any{"search-types": []string{"streams", "users"}})
	require.Nil(t, err)
	require.Equal(t, []string{"streams", "users"}, step.SearchTypes)

	require.Equal(t, "view-feed", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
