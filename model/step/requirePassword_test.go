package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestRequirePassword verifies that a "requirePassword" step parses its configuration
func TestRequirePassword(t *testing.T) {

	step, err := NewRequirePassword(mapof.Any{
		"title":       "Confirm",
		"message":     "Enter your password",
		"submit":      "OK",
		"submitClass": "danger",
		"cancel":      "Back",
	})
	require.Nil(t, err)
	require.NotNil(t, step.Title)
	require.NotNil(t, step.Message)
	require.Equal(t, "OK", step.Submit)
	require.Equal(t, "danger", step.SubmitClass)
	require.Equal(t, "Back", step.Cancel)

	// Note: Name() is "requirePassword" (camelCase), matching the New() factory key.
	require.Equal(t, "requirePassword", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestRequirePassword_Defaults verifies the values a "requirePassword" step falls back to when its configuration is empty
func TestRequirePassword_Defaults(t *testing.T) {
	step, err := NewRequirePassword(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "Confirm", step.Submit)
	require.Equal(t, "warning", step.SubmitClass)
	require.Equal(t, "Cancel", step.Cancel)
}
