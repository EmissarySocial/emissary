package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestStreamPromoteDraft verifies that a "promote-draft" step parses its configuration
func TestStreamPromoteDraft(t *testing.T) {

	step, err := NewStreamPromoteDraft(mapof.Any{"state": "live"})
	require.Nil(t, err)
	require.Equal(t, "live", step.StateID)
	require.Equal(t, []string{"live"}, step.RequiredStates())

	// "state" defaults to "published".
	step, err = NewStreamPromoteDraft(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "published", step.StateID)

	require.Equal(t, "promote-draft", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredRoles())
}
