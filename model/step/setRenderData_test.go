package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSetRenderData verifies that a "set-args" step parses its configuration
func TestSetRenderData(t *testing.T) {

	step, err := NewSetRenderData(mapof.Any{"do": "set-args", "title": "{{.Label}}", "count": "5"})
	require.Nil(t, err)
	require.Contains(t, step.Values, "title")
	require.Contains(t, step.Values, "count")
	require.NotContains(t, step.Values, "do")

	// Note: Name() is "set-args".
	require.Equal(t, "set-args", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestSetRenderData_InvalidTemplate verifies that an invalid template is rejected
func TestSetRenderData_InvalidTemplate(t *testing.T) {
	_, err := NewSetRenderData(mapof.Any{"bad": "{{ .Unclosed"})
	require.NotNil(t, err)
}
