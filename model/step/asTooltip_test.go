package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestAsTooltip verifies that an "as-tooltip" step parses its configuration
func TestAsTooltip(t *testing.T) {

	step, err := NewAsTooltip(mapof.Any{
		"steps": []mapof.Any{{"do": "set-state", "state": "published"}},
	})
	require.Nil(t, err)
	require.Len(t, step.SubSteps, 1)
	require.Contains(t, step.RequiredStates(), "published")

	require.Equal(t, "as-tooltip", step.Name())
	require.Equal(t, "", step.RequiredModel())
}

// TestAsTooltip_InvalidSteps verifies that an invalid steps is rejected
func TestAsTooltip_InvalidSteps(t *testing.T) {
	_, err := NewAsTooltip(mapof.Any{"steps": []mapof.Any{{"do": "nonexistent-step"}}})
	require.NotNil(t, err)
}
