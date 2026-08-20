package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSendEmail verifies that a "send-email" step parses its configuration
func TestSendEmail(t *testing.T) {
	step, err := NewSendEmail(mapof.Any{"email": "welcome"})
	require.Nil(t, err)
	require.Equal(t, "welcome", step.Email)

	require.Equal(t, "send-email", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
