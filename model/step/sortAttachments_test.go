package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSortAttachments verifies that a "sort-attachments" step parses its configuration
func TestSortAttachments(t *testing.T) {

	step, err := NewSortAttachments(mapof.Any{"keys": "token", "values": "order", "message": "Done"})
	require.Nil(t, err)
	require.Equal(t, "token", step.Keys)
	require.Equal(t, "order", step.Values)
	require.Equal(t, "Done", step.Message)

	// Defaults: keys "_id", values "rank".
	step, err = NewSortAttachments(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "_id", step.Keys)
	require.Equal(t, "rank", step.Values)

	require.Equal(t, "sort-attachments", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
