package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestWebSub(t *testing.T) {
	step, err := NewWebSub(mapof.Any{})
	require.Nil(t, err)

	// Note: Name() is "web-sub".
	require.Equal(t, "web-sub", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
