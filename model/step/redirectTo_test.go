package step

import (
	"net/http"
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestRedirectTo verifies that a "redirect-to" step parses its configuration
func TestRedirectTo(t *testing.T) {

	step, err := NewRedirectTo(mapof.Any{"url": "/home", "method": "GET", "status": http.StatusFound})
	require.Nil(t, err)
	require.NotNil(t, step.URL)
	require.Equal(t, "get", step.Method) // lower-cased
	require.Equal(t, http.StatusFound, step.StatusCode)

	// Defaults: method "both", status 307.
	step, err = NewRedirectTo(mapof.Any{"url": "/home"})
	require.Nil(t, err)
	require.Equal(t, "both", step.Method)
	require.Equal(t, http.StatusTemporaryRedirect, step.StatusCode)

	require.Equal(t, "redirect-to", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestRedirectTo_InvalidTemplate verifies that an invalid template is rejected
func TestRedirectTo_InvalidTemplate(t *testing.T) {
	_, err := NewRedirectTo(mapof.Any{"url": "{{ .Unclosed"})
	require.NotNil(t, err)
}
