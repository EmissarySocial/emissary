package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestStartupCreateStreams verifies that a "startup-create-streams" step parses its configuration
func TestStartupCreateStreams(t *testing.T) {

	step, err := NewStartupCreateStreams(mapof.Any{})
	require.Nil(t, err)

	require.Equal(t, "startup-create-streams", step.Name())
	require.Equal(t, "Domain", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestStartupCreateStreams_TemplateRoles verifies that the step restricts itself to admin
// Templates, and that it does so through the optional interface the Template service actually
// type-asserts.  Asserting the method alone would still pass if the interface were renamed.
func TestStartupCreateStreams_TemplateRoles(t *testing.T) {

	step, err := New(mapof.Any{"do": "startup-create-streams"})
	require.Nil(t, err)

	requirer, ok := step.(TemplateRoleRequirer)
	require.True(t, ok, "StartupCreateStreams must implement TemplateRoleRequirer")
	require.Equal(t, []string{"admin"}, requirer.RequiredTemplateRoles())
}
