package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestViewAttachment verifies that a "view-attachment" step parses its configuration
func TestViewAttachment(t *testing.T) {

	step, err := NewViewAttachment(mapof.Any{
		"format":   []string{"pdf", "docx"},
		"category": []string{"documents"},
		"height":   []int{100},
		"width":    []int{200},
		"bitrate":  []int{128},
		"cache":    "true",
	})
	require.Nil(t, err)
	require.Equal(t, []string{"pdf", "docx"}, []string(step.Formats))
	require.Equal(t, []string{"documents"}, []string(step.Categories))
	require.True(t, step.Cache)

	require.Equal(t, "view-attachment", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestViewAttachment_RequiresFormat verifies that a "view-attachment" step requires a format
func TestViewAttachment_RequiresFormat(t *testing.T) {
	// At least one format is required.
	_, err := NewViewAttachment(mapof.Any{})
	require.NotNil(t, err)
}

// TestViewAttachment_CacheDefaultsTrue verifies that caching is enabled unless the configuration turns it off
func TestViewAttachment_CacheDefaultsTrue(t *testing.T) {
	step, err := NewViewAttachment(mapof.Any{"format": []string{"pdf"}})
	require.Nil(t, err)
	require.True(t, step.Cache) // cache defaults to true when unset
}
