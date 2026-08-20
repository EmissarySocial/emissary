package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSaveAndPublish verifies that a "save-and-publish" step parses its configuration
func TestSaveAndPublish(t *testing.T) {

	step, err := NewSaveAndPublish(mapof.Any{"state": "live", "outbox": true, "republish": true})
	require.Nil(t, err)
	require.Equal(t, "live", step.StateID)
	require.True(t, step.Outbox)
	require.True(t, step.Republish)
	require.Equal(t, []string{"live"}, step.RequiredStates())

	// "state" defaults to "published".
	step, err = NewSaveAndPublish(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "published", step.StateID)

	require.Equal(t, "save-and-publish", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredRoles())
}
