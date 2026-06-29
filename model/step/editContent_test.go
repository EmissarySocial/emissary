package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestEditContent(t *testing.T) {

	step, err := NewEditContent(mapof.Any{
		"file":   "my-file",
		"field":  "body",
		"format": "HTML",
	})
	require.Nil(t, err)
	require.Equal(t, "my-file", step.Filename)
	require.Equal(t, "body", step.Fieldname)
	require.Equal(t, "HTML", step.Format)

	require.Equal(t, "edit-content", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

func TestEditContent_Defaults(t *testing.T) {

	// "field" defaults to "content", "format" defaults to "editorjs".
	step, err := NewEditContent(mapof.Any{"format": "HTML"})
	require.Nil(t, err)
	require.Equal(t, "content", step.Fieldname)
}

func TestEditContent_InvalidFormat(t *testing.T) {

	// "format" is a required enum; an invalid value fails schema validation.
	_, err := NewEditContent(mapof.Any{"format": "not-a-format"})
	require.NotNil(t, err)
}
