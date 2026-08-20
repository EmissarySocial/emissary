package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSetResponse verifies that a "set-response" step parses its configuration
func TestSetResponse(t *testing.T) {
	step, err := NewSetResponse(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "set-response", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
