package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestAsModal verifies that an "as-modal" step parses its configuration
func TestAsModal(t *testing.T) {

	step, err := NewAsModal(mapof.Any{
		"steps":      []mapof.Any{{"do": "set-state", "state": "published"}},
		"options":    []string{"size:large"},
		"background": "refresh",
	}, nil)
	require.Nil(t, err)
	require.Len(t, step.SubSteps, 1)
	require.Len(t, step.Options, 1)
	require.Equal(t, "size:large", step.Options[0].Root.String())
	require.Equal(t, "refresh", step.Background)

	// Required states/roles bubble up from the sub-steps.
	require.Contains(t, step.RequiredStates(), "published")

	require.Equal(t, "as-modal", step.Name())
	require.Equal(t, "", step.RequiredModel())
}

// TestAsModal_InvalidSteps verifies that an invalid steps is rejected
func TestAsModal_InvalidSteps(t *testing.T) {
	_, err := NewAsModal(mapof.Any{"steps": []mapof.Any{{"do": "nonexistent-step"}}}, nil)
	require.NotNil(t, err)
}

// TestAsModal_Background verifies that "as-modal" parses background-url, replace-url, and head, and
// leaves each one empty when it is absent
func TestAsModal_Background(t *testing.T) {

	step, err := NewAsModal(mapof.Any{
		"steps":          []mapof.Any{{"do": "view-html", "file": "modal"}},
		"background-url": "/@{{.ParentID}}?tab={{.StreamID}}",
		"replace-url":    "/{{.StreamID}}",
		"head":           []mapof.Any{{"do": "set-state", "state": "archived"}},
	}, nil)
	require.Nil(t, err)
	require.Equal(t, "/@{{.ParentID}}?tab={{.StreamID}}", step.BackgroundURL.Root.String())
	require.Equal(t, "/{{.StreamID}}", step.ReplaceURL.Root.String())
	require.Len(t, step.Head, 1)

	// Required states bubble up from the head steps too
	require.Contains(t, step.RequiredStates(), "archived")

	// Each one is empty when it is absent
	empty, err := NewAsModal(mapof.Any{"steps": []mapof.Any{}}, nil)
	require.Nil(t, err)
	require.Nil(t, empty.BackgroundURL)
	require.Nil(t, empty.ReplaceURL)
	require.Empty(t, empty.Head)
}

// TestAsModal_InvalidBackground verifies that a background-url, replace-url, or head that cannot be
// parsed is rejected at load
func TestAsModal_InvalidBackground(t *testing.T) {

	_, err := NewAsModal(mapof.Any{"background-url": "{{.Unclosed"}, nil)
	require.NotNil(t, err)

	_, err = NewAsModal(mapof.Any{"replace-url": "{{.Unclosed"}, nil)
	require.NotNil(t, err)

	_, err = NewAsModal(mapof.Any{"head": []mapof.Any{{"do": "nonexistent-step"}}}, nil)
	require.NotNil(t, err)
}
