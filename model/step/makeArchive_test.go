package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestMakeArchive verifies that a "make-archive" step parses its configuration
func TestMakeArchive(t *testing.T) {

	step, err := NewMakeArchive(mapof.Any{
		"token":       "archive-1",
		"depth":       3,
		"json":        true,
		"attachments": false,
	})
	require.Nil(t, err)
	require.Equal(t, "archive-1", step.Token)
	require.Equal(t, 3, step.Depth)
	require.True(t, step.JSON)
	require.False(t, step.Attachments)

	require.Equal(t, "make-archive", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
