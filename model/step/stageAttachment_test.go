package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestStageAttachment verifies that a "stage-attachment" step parses with no attributes, and ignores any it is given
func TestStageAttachment(t *testing.T) {

	step, err := NewStageAttachment(mapof.Any{"category": "ignored", "fieldname": "ignored"})
	require.Nil(t, err)

	require.Equal(t, "stage-attachment", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Empty(t, step.RequiredStates())
	require.Empty(t, step.RequiredRoles())

	step, err = NewStageAttachment(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "stage-attachment", step.Name())
}
