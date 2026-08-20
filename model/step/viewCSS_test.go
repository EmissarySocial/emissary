package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestViewCSS verifies that a "view-css" step parses its configuration
func TestViewCSS(t *testing.T) {
	step, err := NewViewCSS(mapof.Any{"file": "theme"})
	require.Nil(t, err)
	require.Equal(t, "theme", step.File)

	require.Equal(t, "view-css", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
