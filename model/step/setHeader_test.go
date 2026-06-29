package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestSetHeader(t *testing.T) {

	step, err := NewSetHeader(mapof.Any{"method": "post", "name": "HX-Trigger", "value": "{{.Event}}"})
	require.Nil(t, err)
	require.Equal(t, "post", step.Method)
	require.Equal(t, "HX-Trigger", step.HeaderName)
	require.NotNil(t, step.Value)

	// "method" defaults to "both".
	step, err = NewSetHeader(mapof.Any{"name": "X-Test", "value": "x"})
	require.Nil(t, err)
	require.Equal(t, "both", step.Method)

	require.Equal(t, "set-header", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

func TestSetHeader_InvalidTemplate(t *testing.T) {
	_, err := NewSetHeader(mapof.Any{"value": "{{ .Unclosed"})
	require.NotNil(t, err)
}
