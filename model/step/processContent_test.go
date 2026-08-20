package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestProcessContent verifies that a "process-content" step parses its configuration
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

// TestProcessContent_InvalidFormat verifies that an invalid format is rejected
func TestProcessContent_InvalidFormat(t *testing.T) {
	_, err := NewProcessContent(mapof.Any{"format": "not-allowed"})
	require.NotNil(t, err)
}

// TestProcessContent_DeprecatedTagFieldsStillParse guards the backward-compatibility contract:
// the deprecated "add-tags"/"tag-path" options must still parse without error so that older
// (external) Templates that set them continue to load. They are ignored at build time.
func TestProcessContent_DeprecatedTagFieldsStillParse(t *testing.T) {
	step, err := NewProcessContent(mapof.Any{"add-tags": true, "tag-path": "/home?q="})
	require.Nil(t, err)
	require.True(t, step.AddTags)
	require.Equal(t, "/home?q=", step.TagPath)
}
