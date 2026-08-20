package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSetData verifies that a "set-data" step parses its configuration
func TestSetData(t *testing.T) {

	step, err := NewSetData(mapof.Any{
		"from-url":  []string{"token"},
		"from-form": []string{"name"},
		"values":    map[string]any{"label": "{{.Label}}"},
		"defaults":  map[string]any{"color": "blue"},
	})
	require.Nil(t, err)
	require.Equal(t, []string{"token"}, step.FromURL)
	require.Equal(t, []string{"name"}, step.FromForm)
	require.Contains(t, step.Values, "label")
	require.Equal(t, "blue", step.Defaults.GetString("color"))

	require.Equal(t, "set-data", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestSetData_InvalidTemplate verifies that an invalid template is rejected
func TestSetData_InvalidTemplate(t *testing.T) {
	_, err := NewSetData(mapof.Any{"values": map[string]any{"bad": "{{ .Unclosed"}})
	require.NotNil(t, err)
}
