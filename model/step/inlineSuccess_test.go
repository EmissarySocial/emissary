package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestInlineSuccess(t *testing.T) {
	step, err := NewInlineSuccess(mapof.Any{"message": "Saved!"})
	require.Nil(t, err)
	require.NotNil(t, step.Message)

	require.Equal(t, "inline-success", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

func TestInlineSuccess_InvalidTemplate(t *testing.T) {
	_, err := NewInlineSuccess(mapof.Any{"message": "{{ .Unclosed"})
	require.NotNil(t, err)
}
