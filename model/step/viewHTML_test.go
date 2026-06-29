package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestViewHTML(t *testing.T) {

	step, err := NewViewHTML(mapof.Any{"file": "detail", "method": "post", "as-full-page": true})
	require.Nil(t, err)
	require.Equal(t, "detail", step.File)
	require.Equal(t, "post", step.Method)
	require.True(t, step.AsFullPage)

	// "method" defaults to "get".
	step, err = NewViewHTML(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "get", step.Method)

	require.Equal(t, "view-html", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
