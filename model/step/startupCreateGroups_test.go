package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestStartupCreateGroups verifies that a "startup-create-groups" step parses its configuration
func TestStartupCreateGroups(t *testing.T) {

	step, err := NewStartupCreateGroups(mapof.Any{})
	require.Nil(t, err)

	require.Equal(t, "startup-create-groups", step.Name())
	require.Equal(t, "Domain", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestStartupCreateGroups_TemplateRoles verifies that the step restricts itself to admin Templates,
// through the optional interface that the Template service type-asserts.
func TestStartupCreateGroups_TemplateRoles(t *testing.T) {

	step, err := New(mapof.Any{"do": "startup-create-groups"}, nil)
	require.Nil(t, err)

	requirer, ok := step.(TemplateRoleRequirer)
	require.True(t, ok, "StartupCreateGroups must implement TemplateRoleRequirer")
	require.Equal(t, []string{"admin"}, requirer.RequiredTemplateRoles())
}
