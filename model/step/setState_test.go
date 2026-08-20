package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSetState verifies that a "set-state" step parses its configuration
func TestSetState(t *testing.T) {

	step, err := NewSetState(mapof.Any{"state": "published"})
	require.Nil(t, err)
	require.Equal(t, "published", step.State)
	require.Equal(t, []string{"published"}, step.RequiredStates())

	require.Equal(t, "set-state", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestSetState_RequiresState verifies that a "set-state" step requires a state
func TestSetState_RequiresState(t *testing.T) {
	// "state" is required.
	_, err := NewSetState(mapof.Any{})
	require.NotNil(t, err)
}
