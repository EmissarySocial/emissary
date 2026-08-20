package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSort verifies that a "set-sort" step parses its configuration
func TestSort(t *testing.T) {

	step, err := NewSort(mapof.Any{"model": "Stream", "keys": "token", "values": "order", "message": "Reordered"})
	require.Nil(t, err)
	require.Equal(t, "Stream", step.Model)
	require.Equal(t, "token", step.Keys)
	require.Equal(t, "order", step.Values)
	require.Equal(t, "Reordered", step.Message)

	// Defaults: keys "_id", values "rank".
	step, err = NewSort(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "_id", step.Keys)
	require.Equal(t, "rank", step.Values)

	// Note: Name() is "set-sort".
	require.Equal(t, "set-sort", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
