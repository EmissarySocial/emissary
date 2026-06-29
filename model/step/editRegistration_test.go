package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestEditRegistration(t *testing.T) {
	step, err := NewEditRegistration(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "edit-registration", step.Name())
	require.Equal(t, "Domain", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
