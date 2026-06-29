package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestTriggerEvent(t *testing.T) {
	step, err := NewTriggerEvent(mapof.Any{"event": "saved", "value": "{{.ID}}"})
	require.Nil(t, err)
	require.Equal(t, "saved", step.Event)
	require.NotNil(t, step.Value)

	require.Equal(t, "trigger-event", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

func TestTriggerEvent_InvalidTemplate(t *testing.T) {
	_, err := NewTriggerEvent(mapof.Any{"value": "{{ .Unclosed"})
	require.NotNil(t, err)
}
