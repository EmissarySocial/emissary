package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestProcessTags verifies that a "process-tags" step parses its configuration
func TestProcessTags(t *testing.T) {

	// Comma-separated paths are split and trimmed.
	step, err := NewProcessTags(mapof.Any{"paths": "name, summary , content"})
	require.Nil(t, err)
	require.Equal(t, []string{"name", "summary", "content"}, step.Paths)

	require.Equal(t, "process-tags", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
