package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestRemoveEvent verifies that a "remove-event" step parses its configuration
func TestRemoveEvent(t *testing.T) {
	step, err := NewRemoveEvent(mapof.Any{"event": "closeModal"})
	require.Nil(t, err)
	require.Equal(t, "closeModal", step.Event)

	require.Equal(t, "remove-event", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
