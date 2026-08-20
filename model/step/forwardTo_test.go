package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestForwardTo verifies that a "forward-to" step parses its configuration
func TestForwardTo(t *testing.T) {

	step, err := NewForwardTo(mapof.Any{"url": "/next/{{.Token}}", "method": "GET"})
	require.Nil(t, err)
	require.NotNil(t, step.URL)
	require.Equal(t, "get", step.Method) // lower-cased

	// "method" defaults to "post".
	step, err = NewForwardTo(mapof.Any{"url": "/next"})
	require.Nil(t, err)
	require.Equal(t, "post", step.Method)

	require.Equal(t, "forward-to", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestForwardTo_InvalidTemplate verifies that an invalid template is rejected
func TestForwardTo_InvalidTemplate(t *testing.T) {
	_, err := NewForwardTo(mapof.Any{"url": "{{ .Unclosed"})
	require.NotNil(t, err)
}
