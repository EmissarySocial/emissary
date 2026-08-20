package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestAsConfirmation verifies that an "as-confirmation" step parses its configuration
func TestAsConfirmation(t *testing.T) {

	step, err := NewAsConfirmation(mapof.Any{
		"icon":    "warning",
		"title":   "Are you sure?",
		"message": "This cannot be undone.",
		"submit":  "Do it",
	})
	require.Nil(t, err)
	require.Equal(t, "warning", step.Icon)
	require.Equal(t, "Are you sure?", step.Title)
	require.Equal(t, "This cannot be undone.", step.Message)
	require.Equal(t, "Do it", step.Submit)

	// "submit" defaults to "Continue".
	step, err = NewAsConfirmation(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "Continue", step.Submit)

	require.Equal(t, "as-confirmation", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
