package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestGetArchive verifies that a "get-archive" step parses its configuration
func TestGetArchive(t *testing.T) {

	step, err := NewGetArchive(mapof.Any{
		"token":       "archive-1",
		"depth":       2,
		"json":        true,
		"attachments": true,
	})
	require.Nil(t, err)
	require.Equal(t, "archive-1", step.Token)
	require.Equal(t, 2, step.Depth)
	require.True(t, step.JSON)
	require.True(t, step.Attachments)

	require.Equal(t, "get-archive", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
