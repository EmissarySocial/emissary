package step

import (
	"strings"
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestStartupSaveTask verifies that a "startup-save-task" step parses its configuration
func TestStartupSaveTask(t *testing.T) {

	step, err := NewStartupSaveTask(mapof.Any{"value": "sample-content"})
	require.Nil(t, err)
	require.Equal(t, "sample-content", step.Value)

	require.Equal(t, "startup-save-task", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestStartupSaveTask_InvalidConfig asserts that a bad "value" is rejected when the Template
// loads.  The 32-character ceiling mirrors the "startupTasks" property in model.DomainSchema(),
// so an over-long value fails at boot instead of failing validation on save.
func TestStartupSaveTask_InvalidConfig(t *testing.T) {

	// Missing "value"
	_, err := NewStartupSaveTask(mapof.Any{})
	require.NotNil(t, err)

	// Empty "value"
	_, err = NewStartupSaveTask(mapof.Any{"value": ""})
	require.NotNil(t, err)

	// "value" longer than the Domain schema allows
	_, err = NewStartupSaveTask(mapof.Any{"value": strings.Repeat("x", 33)})
	require.NotNil(t, err)

	// Exactly at the limit is fine
	step, err := NewStartupSaveTask(mapof.Any{"value": strings.Repeat("x", 32)})
	require.Nil(t, err)
	require.Equal(t, strings.Repeat("x", 32), step.Value)
}
