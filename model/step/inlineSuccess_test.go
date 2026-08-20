package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestInlineSuccess verifies that an "inline-success" step parses its configuration
func TestInlineSuccess(t *testing.T) {
	step, err := NewInlineSuccess(mapof.Any{"message": "Saved!"})
	require.Nil(t, err)
	require.NotNil(t, step.Message)

	require.Equal(t, "inline-success", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestInlineSuccess_InvalidTemplate verifies that an invalid template is rejected
func TestInlineSuccess_InvalidTemplate(t *testing.T) {
	_, err := NewInlineSuccess(mapof.Any{"message": "{{ .Unclosed"})
	require.NotNil(t, err)
}
