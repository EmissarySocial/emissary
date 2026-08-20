package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSetPassword verifies that a "set-password" step parses its configuration
func TestSetPassword(t *testing.T) {
	step, err := NewSetPassword(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "set-password", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
