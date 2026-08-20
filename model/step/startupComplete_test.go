package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestStartupComplete verifies that a "startup-complete" step parses its configuration
func TestStartupComplete(t *testing.T) {

	step, err := NewStartupComplete(mapof.Any{})
	require.Nil(t, err)

	require.Equal(t, "startup-complete", step.Name())
	require.Equal(t, "Domain", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestStartupComplete_TemplateRoles verifies that the step restricts itself to admin Templates, and
// that it does so through the optional interface the Template service actually type-asserts.
// Asserting the method alone would still pass if the interface were renamed.
func TestStartupComplete_TemplateRoles(t *testing.T) {

	step, err := New(mapof.Any{"do": "startup-complete"})
	require.Nil(t, err)

	requirer, ok := step.(TemplateRoleRequirer)
	require.True(t, ok, "StartupComplete must implement TemplateRoleRequirer")
	require.Equal(t, []string{"admin"}, requirer.RequiredTemplateRoles())
}
