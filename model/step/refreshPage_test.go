package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestRefreshPage verifies that a "refresh-page" step parses its configuration
func TestRefreshPage(t *testing.T) {
	step, err := NewRefreshPage(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "refresh-page", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
