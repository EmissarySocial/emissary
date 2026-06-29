package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

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

func TestAsTooltip_InvalidSteps(t *testing.T) {
	_, err := NewAsTooltip(mapof.Any{"steps": []mapof.Any{{"do": "nonexistent-step"}}})
	require.NotNil(t, err)
}
