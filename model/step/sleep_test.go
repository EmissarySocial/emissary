package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSleep verifies that a "sleep" step parses its configuration
func TestSleep(t *testing.T) {
	step, err := NewSleep(mapof.Any{"duration": 500})
	require.Nil(t, err)
	require.Equal(t, 500, step.Duration)

	require.Equal(t, "sleep", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
