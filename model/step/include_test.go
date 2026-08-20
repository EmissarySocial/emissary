package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestInclude verifies that an "include" step parses its configuration
func TestInclude(t *testing.T) {
	step, err := NewInclude(mapof.Any{"action": "view"})
	require.Nil(t, err)
	require.Equal(t, "view", step.Action)

	require.Equal(t, "include", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
