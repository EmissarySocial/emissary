package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestEditWidget verifies that an "edit-widget" step parses its configuration
func TestEditWidget(t *testing.T) {
	step, err := NewEditWidget(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "edit-widget", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
