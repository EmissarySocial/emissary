package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSetThumbnail verifies that a "set-thumbnail" step parses its configuration
func TestSetThumbnail(t *testing.T) {
	step, err := NewSetThumbnail(mapof.Any{"path": "image"})
	require.Nil(t, err)
	require.Equal(t, "image", step.Path)

	require.Equal(t, "set-thumbnail", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
