package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestDeleteAttachments verifies that a "delete-attachments" step parses its configuration
func TestDeleteAttachments(t *testing.T) {

	step, err := NewDeleteAttachments(mapof.Any{
		"all":      true,
		"field":    "avatar",
		"category": "images",
	})
	require.Nil(t, err)
	require.True(t, step.All)
	require.Equal(t, "avatar", step.Field)
	require.Equal(t, "images", step.Category)

	// Defaults.
	step, err = NewDeleteAttachments(mapof.Any{})
	require.Nil(t, err)
	require.False(t, step.All)
	require.Equal(t, "", step.Field)
	require.Equal(t, "", step.Category)

	require.Equal(t, "delete-attachments", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
