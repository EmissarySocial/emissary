package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestAddModelObject verifies that an "add" step parses its configuration
func TestAddModelObject(t *testing.T) {

	// Parses a form and a default pipeline. (form.Parse accepts map[string]any, not mapof.Any.)
	step, err := NewAddModelObject(mapof.Any{
		"form": map[string]any{"type": "layout-vertical"},
		"defaults": []mapof.Any{
			{"do": "set-state", "state": "published"},
		},
	})
	require.Nil(t, err)
	require.Len(t, step.Defaults, 1)
	require.Equal(t, step.Form, step.GetForm()) // GetForm returns the parsed form

	// RequiredStates is derived from the Defaults pipeline ("set-state" requires "published").
	require.Contains(t, step.RequiredStates(), "published")

	// Interface methods.
	require.Equal(t, "add", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestAddModelObject_InvalidDefaults verifies that an invalid defaults is rejected
func TestAddModelObject_InvalidDefaults(t *testing.T) {

	// An unrecognized step in "defaults" propagates an error.
	_, err := NewAddModelObject(mapof.Any{
		"defaults": []mapof.Any{
			{"do": "this-step-does-not-exist"},
		},
	})
	require.NotNil(t, err)
}
