package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestDump verifies that a "dump" step parses its configuration
func TestDump(t *testing.T) {

	step, err := NewDump(mapof.Any{"value": "{{.Label}}"})
	require.Nil(t, err)
	require.NotNil(t, step.Value)

	require.Equal(t, "dump", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestDump_InvalidTemplate verifies that an invalid template is rejected
func TestDump_InvalidTemplate(t *testing.T) {
	_, err := NewDump(mapof.Any{"value": "{{ .Unclosed"})
	require.NotNil(t, err)
}
