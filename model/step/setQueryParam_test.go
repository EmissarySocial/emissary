package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSetQueryParam verifies that a "set-query-param" step parses its configuration
func TestSetQueryParam(t *testing.T) {

	// All keys except "do" become value templates.
	step, err := NewSetQueryParam(mapof.Any{"do": "set-query-param", "page": "2", "sort": "{{.Sort}}"})
	require.Nil(t, err)
	require.Contains(t, step.Values, "page")
	require.Contains(t, step.Values, "sort")
	require.NotContains(t, step.Values, "do")

	require.Equal(t, "set-query-param", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestSetQueryParam_InvalidTemplate verifies that an invalid template is rejected
func TestSetQueryParam_InvalidTemplate(t *testing.T) {
	_, err := NewSetQueryParam(mapof.Any{"bad": "{{ .Unclosed"})
	require.NotNil(t, err)
}
