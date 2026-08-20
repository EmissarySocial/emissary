package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestDelete verifies that a "delete" step parses its configuration
func TestDelete(t *testing.T) {

	step, err := NewDelete(mapof.Any{
		"title":   "Remove it?",
		"message": "Gone forever.",
		"submit":  "Yes",
		"cancel":  "No",
		"method":  "post",
	})
	require.Nil(t, err)
	require.NotNil(t, step.Title)
	require.NotNil(t, step.Message)
	require.Equal(t, "Yes", step.Submit)
	require.Equal(t, "No", step.Cancel)
	require.Equal(t, "post", step.Method)

	require.Equal(t, "delete", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestDelete_Defaults verifies the values a "delete" step falls back to when its configuration is empty
func TestDelete_Defaults(t *testing.T) {

	step, err := NewDelete(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "Delete", step.Submit)
	require.Equal(t, "Cancel", step.Cancel)
	require.Equal(t, "both", step.Method)
	require.NotNil(t, step.Title)   // default title template
	require.NotNil(t, step.Message) // default message template
}

// TestDelete_InvalidConfig verifies that an invalid config is rejected
func TestDelete_InvalidConfig(t *testing.T) {

	// "method" outside the schema enum fails validation.
	_, err := NewDelete(mapof.Any{"method": "not-a-method"})
	require.NotNil(t, err)
}
