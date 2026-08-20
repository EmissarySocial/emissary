package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestHalt verifies that a "halt" step parses its configuration
func TestHalt(t *testing.T) {
	step, err := NewHalt(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "halt", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
