package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestEditWidget(t *testing.T) {
	step, err := NewEditWidget(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "edit-widget", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
