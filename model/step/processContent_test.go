package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestProcessContent(t *testing.T) {

	step, err := NewProcessContent(mapof.Any{
		"format":      "HTML",
		"remove-html": true,
		"add-links":   true,
		"add-tags":    true,
		"tag-path":    "tags",
	})
	require.Nil(t, err)
	require.Equal(t, "HTML", step.Format)
	require.True(t, step.RemoveHTML)
	require.True(t, step.AddLinks)
	require.True(t, step.AddTags)
	require.Equal(t, "tags", step.TagPath)

	// Empty format is allowed.
	step, err = NewProcessContent(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "", step.Format)

	require.Equal(t, "process-content", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

func TestProcessContent_InvalidFormat(t *testing.T) {
	_, err := NewProcessContent(mapof.Any{"format": "not-allowed"})
	require.NotNil(t, err)
}
