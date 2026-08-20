package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSearchIndex verifies that a "search-index" step parses its configuration
func TestSearchIndex(t *testing.T) {

	step, err := NewSearchIndex(mapof.Any{"if": "{{.IsPublic}}"})
	require.Nil(t, err)
	require.NotNil(t, step.If)

	// "if" defaults to a parseable "true" template.
	step, err = NewSearchIndex(mapof.Any{})
	require.Nil(t, err)
	require.NotNil(t, step.If)

	require.Equal(t, "search-index", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestSearchIndex_InvalidTemplate verifies that an invalid template is rejected
func TestSearchIndex_InvalidTemplate(t *testing.T) {
	_, err := NewSearchIndex(mapof.Any{"if": "{{ .Unclosed"})
	require.NotNil(t, err)
}
