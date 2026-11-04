package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestEditAttachments verifies that an "edit-attachments" step parses its configuration
func TestEditAttachments(t *testing.T) {

	step, err := NewEditAttachments(mapof.Any{
		"category":    "photo",
		"fieldname":   "photos",
		"accept-type": "image/*",
		"minimum":     1,
		"maximum":     64,
		"rules":       map[string]any{"height": 100, "width": 200, "types": []string{"webp"}},
	})
	require.Nil(t, err)
	require.Equal(t, "photo", step.Category)
	require.Equal(t, "photos", step.Fieldname)
	require.Equal(t, "image/*", step.AcceptType)
	require.Equal(t, 1, step.Minimum)
	require.Equal(t, 64, step.Maximum)
	require.Equal(t, 100, step.RuleHeight)
	require.Equal(t, 200, step.RuleWidth)
	require.Equal(t, []string{"webp"}, step.RuleTypes)

	require.Equal(t, "edit-attachments", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Empty(t, step.RequiredStates())
	require.Empty(t, step.RequiredRoles())
}

// TestEditAttachments_Defaults verifies the values the step falls back to: the fieldname is the category, and the limits are 0 and 1
func TestEditAttachments_Defaults(t *testing.T) {

	step, err := NewEditAttachments(mapof.Any{"category": "photo"})
	require.Nil(t, err)
	require.Equal(t, "photo", step.Fieldname)
	require.Equal(t, "", step.AcceptType)
	require.Equal(t, 0, step.Minimum)
	require.Equal(t, 1, step.Maximum)
	require.Empty(t, step.RuleTypes)
}

// TestEditAttachments_CategoryIsRequired verifies that the step cannot load without a category
func TestEditAttachments_CategoryIsRequired(t *testing.T) {

	_, err := NewEditAttachments(mapof.Any{})
	require.Error(t, err)

	_, err = NewEditAttachments(mapof.Any{"fieldname": "photos"})
	require.Error(t, err)
}

// TestEditAttachments_Limits verifies that the limits never go below their floors and that the minimum never exceeds the maximum
func TestEditAttachments_Limits(t *testing.T) {

	step, err := NewEditAttachments(mapof.Any{"category": "photo", "minimum": -3, "maximum": -3})
	require.Nil(t, err)
	require.Equal(t, 0, step.Minimum)
	require.Equal(t, 1, step.Maximum)

	step, err = NewEditAttachments(mapof.Any{"category": "photo", "minimum": 1, "maximum": 1})
	require.Nil(t, err)
	require.Equal(t, 1, step.Minimum)
	require.Equal(t, 1, step.Maximum)

	// A minimum above the maximum (which defaults to 1) fails to load
	_, err = NewEditAttachments(mapof.Any{"category": "photo", "minimum": 2})
	require.Error(t, err)

	_, err = NewEditAttachments(mapof.Any{"category": "photo", "minimum": 65, "maximum": 64})
	require.Error(t, err)
}
