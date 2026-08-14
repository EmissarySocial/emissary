package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestViewHTML(t *testing.T) {

	step, err := NewViewHTML(mapof.Any{"file": "detail", "method": "post", "cache-control": "public, max-age=300", "as-full-page": true})
	require.Nil(t, err)
	require.Equal(t, "detail", step.File)
	require.Equal(t, "post", step.Method)
	require.Equal(t, "public, max-age=300", step.CacheControl)
	require.True(t, step.AsFullPage)

	// "method" defaults to "get".  "cache-control" does NOT default here -- an empty value means
	// "whatever the build step considers safe", which is where that decision is made.
	step, err = NewViewHTML(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "get", step.Method)
	require.Equal(t, "", step.CacheControl)

	require.Equal(t, "view-html", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
