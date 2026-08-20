package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestInlineError verifies that an "inline-error" step parses its configuration
func TestInlineError(t *testing.T) {
	step, err := NewInlineError(mapof.Any{"message": "Something went wrong"})
	require.Nil(t, err)
	require.NotNil(t, step.Message)

	require.Equal(t, "inline-error", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestInlineError_InvalidTemplate verifies that an invalid template is rejected
func TestInlineError_InvalidTemplate(t *testing.T) {
	_, err := NewInlineError(mapof.Any{"message": "{{ .Unclosed"})
	require.NotNil(t, err)
}
