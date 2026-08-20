package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestEditConnection verifies that an "edit-connection" step parses its configuration
func TestEditConnection(t *testing.T) {
	step, err := NewEditConnection(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "edit-connection", step.Name())
	require.Equal(t, "Domain", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
