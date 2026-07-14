package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestSetSimpleSharing(t *testing.T) {

	step, err := NewSetSimpleSharing(mapof.Any{"title": "Share", "message": "msg", "role": "viewer"})
	require.Nil(t, err)
	require.Equal(t, "Share", step.Title)
	require.Equal(t, "msg", step.Message)
	require.Equal(t, "viewer", step.Role)
	require.Equal(t, []string{"viewer"}, step.RequiredRoles())

	require.Equal(t, "set-simple-sharing", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
}

func TestSetSimpleSharing_RequiresRole(t *testing.T) {
	_, err := NewSetSimpleSharing(mapof.Any{})
	require.NotNil(t, err)
}
