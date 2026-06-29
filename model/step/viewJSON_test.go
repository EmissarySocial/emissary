package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestViewJSON(t *testing.T) {

	step, err := NewViewJSON(mapof.Any{"value": ".Object"})
	require.Nil(t, err)
	require.NotNil(t, step.Value)

	// A jsonp wrapper is also accepted.
	step, err = NewViewJSON(mapof.Any{"value": ".Object", "jsonp": "callback"})
	require.Nil(t, err)
	require.NotNil(t, step.Value)

	require.Equal(t, "view-json", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

func TestViewJSON_RequiresValue(t *testing.T) {
	// A query template is required.
	_, err := NewViewJSON(mapof.Any{})
	require.NotNil(t, err)
}
