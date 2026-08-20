package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestAddEvent verifies that an "add-event" step parses its configuration
func TestAddEvent(t *testing.T) {

	// Explicit values are parsed through.
	step, err := NewAddEvent(mapof.Any{"method": "get", "event": "refreshSidebar"})
	require.Nil(t, err)
	require.Equal(t, "get", step.Method)
	require.Equal(t, "refreshSidebar", step.Event)

	// "method" defaults to "post" when omitted.
	step, err = NewAddEvent(mapof.Any{"event": "ping"})
	require.Nil(t, err)
	require.Equal(t, "post", step.Method)
	require.Equal(t, "ping", step.Event)

	// Interface methods.
	require.Equal(t, "add-event", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
