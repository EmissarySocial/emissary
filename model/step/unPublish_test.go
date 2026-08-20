package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestUnPublish verifies that an "unpublish" step parses its configuration
func TestUnPublish(t *testing.T) {

	step, err := NewUnPublish(mapof.Any{"state": "archived", "outbox": true})
	require.Nil(t, err)
	require.Equal(t, "archived", step.StateID)
	require.True(t, step.Outbox)
	require.Equal(t, []string{"archived"}, step.RequiredStates())

	// "state" defaults to "default".
	step, err = NewUnPublish(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "default", step.StateID)

	require.Equal(t, "unpublish", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredRoles())
}
