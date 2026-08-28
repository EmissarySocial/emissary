package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSetSharing verifies that a "set-sharing" step parses each of the three magic Group names
func TestSetSharing(t *testing.T) {

	for _, group := range []string{"anonymous", "authenticated", "owner"} {

		step, err := NewSetSharing(mapof.Any{"role": "viewer", "group": group})
		require.Nil(t, err, "group %q must parse", group)
		require.Equal(t, "viewer", step.Role)
		require.Equal(t, group, step.Group)
		require.Equal(t, []string{"viewer"}, step.RequiredRoles())

		require.Equal(t, "set-sharing", step.Name())
		require.Equal(t, "Stream", step.RequiredModel())
		require.Equal(t, []string{}, step.RequiredStates())
	}
}

// TestSetSharing_RequiresRole verifies that a "set-sharing" step requires a role
func TestSetSharing_RequiresRole(t *testing.T) {
	_, err := NewSetSharing(mapof.Any{"group": "anonymous"})
	require.NotNil(t, err)
}

// TestSetSharing_RequiresGroup verifies that a "set-sharing" step rejects missing or unrecognized groups
func TestSetSharing_RequiresGroup(t *testing.T) {

	for _, group := range []string{"", "everyone", "ANONYMOUS", "owners", "author"} {
		_, err := NewSetSharing(mapof.Any{"role": "viewer", "group": group})
		require.NotNil(t, err, "group %q must be rejected", group)
	}
}
