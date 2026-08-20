package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestReloadPage verifies that a "reload-page" step parses its configuration
func TestReloadPage(t *testing.T) {
	step, err := NewReloadPage(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "reload-page", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
