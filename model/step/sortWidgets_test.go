package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSortWidgets verifies that a "sort-widgets" step parses its configuration
func TestSortWidgets(t *testing.T) {
	step, err := NewSortWidgets(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "sort-widgets", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
