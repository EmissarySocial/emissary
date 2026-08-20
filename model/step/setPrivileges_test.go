package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSetPrivileges verifies that a "set-privileges" step parses its configuration
func TestSetPrivileges(t *testing.T) {

	step, err := NewSetPrivileges(mapof.Any{"title": "Memberships"})
	require.Nil(t, err)
	require.Equal(t, "Memberships", step.Title)

	// "title" defaults to "Product Settings".
	step, err = NewSetPrivileges(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "Product Settings", step.Title)

	require.Equal(t, "set-privileges", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
