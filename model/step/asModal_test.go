package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestAsModal verifies that an "as-modal" step parses its configuration
func TestAsModal(t *testing.T) {

	step, err := NewAsModal(mapof.Any{
		"steps":      []mapof.Any{{"do": "set-state", "state": "published"}},
		"options":    []string{"size:large"},
		"background": "refresh",
	})
	require.Nil(t, err)
	require.Len(t, step.SubSteps, 1)
	require.Equal(t, []string{"size:large"}, step.Options)
	require.Equal(t, "refresh", step.Background)

	// Required states/roles bubble up from the sub-steps.
	require.Contains(t, step.RequiredStates(), "published")

	require.Equal(t, "as-modal", step.Name())
	require.Equal(t, "", step.RequiredModel())
}

// TestAsModal_InvalidSteps verifies that an invalid steps is rejected
func TestAsModal_InvalidSteps(t *testing.T) {
	_, err := NewAsModal(mapof.Any{"steps": []mapof.Any{{"do": "nonexistent-step"}}})
	require.NotNil(t, err)
}
