package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestDeleteArchive verifies that a "delete-archive" step parses its configuration
func TestDeleteArchive(t *testing.T) {

	step, err := NewDeleteArchive(mapof.Any{"token": "backup-2024"})
	require.Nil(t, err)
	require.Equal(t, "backup-2024", step.Token)

	require.Equal(t, "delete-archive", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
