package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestTableEditor verifies that an "edit-table" step parses its configuration
func TestTableEditor(t *testing.T) {

	step, err := NewTableEditor(mapof.Any{
		"path": "links",
		"form": map[string]any{"type": "layout-vertical"},
	})
	require.Nil(t, err)
	require.Equal(t, "links", step.Path)
	require.Equal(t, step.Form, step.GetForm())

	require.Equal(t, "edit-table", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestTableEditor_InvalidForm verifies that an invalid form is rejected
func TestTableEditor_InvalidForm(t *testing.T) {
	_, err := NewTableEditor(mapof.Any{"form": "not-valid-json"})
	require.NotNil(t, err)
}
