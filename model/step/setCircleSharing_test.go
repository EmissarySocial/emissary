package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSetCircleSharing verifies that a "set-circle-sharing" step parses its configuration
func TestSetCircleSharing(t *testing.T) {

	step, err := NewSetCircleSharing(mapof.Any{
		"method":  "POST",
		"title":   "Share",
		"message": "Pick circles",
		"button":  "Apply",
		"role":    "viewer",
	})
	require.Nil(t, err)
	require.Equal(t, "post", step.Method) // lower-cased
	require.Equal(t, "Share", step.Title)
	require.Equal(t, "Pick circles", step.Message)
	require.Equal(t, "Apply", step.Button)
	require.Equal(t, "viewer", step.Role)
	require.Equal(t, []string{"viewer"}, step.RequiredRoles())

	require.Equal(t, "set-circle-sharing", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
}

// TestSetCircleSharing_RequiresRole verifies that a "set-circle-sharing" step requires a role
func TestSetCircleSharing_RequiresRole(t *testing.T) {
	_, err := NewSetCircleSharing(mapof.Any{})
	require.NotNil(t, err)
}
