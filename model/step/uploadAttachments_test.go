package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestUploadAttachments verifies that an "upload-attachments" step parses its configuration
func TestUploadAttachments(t *testing.T) {

	step, err := NewUploadAttachments(mapof.Any{
		"action":          "replace",
		"fieldname":       "upload",
		"attachment-path": "iconId",
		"accept-type":     "image/*",
		"maximum":         5,
		"category":        "avatars",
		"json-result":     true,
		"rules":           map[string]any{"height": 100, "width": 200, "types": []string{"png", "jpg"}},
	})
	require.Nil(t, err)
	require.Equal(t, "replace", step.Action)
	require.Equal(t, "upload", step.Fieldname)
	require.Equal(t, "iconId", step.AttachmentPath)
	require.Equal(t, "image/*", step.AcceptType)
	require.Equal(t, 5, step.Maximum)
	require.Equal(t, "avatars", step.Category)
	require.True(t, step.JSONResult)
	require.Equal(t, 100, step.RuleHeight)
	require.Equal(t, 200, step.RuleWidth)
	require.Equal(t, []string{"png", "jpg"}, step.RuleTypes)

	require.Equal(t, "upload-attachments", step.Name())
}

// TestUploadAttachments_Defaults verifies the values an "upload-attachments" step falls back to when its configuration is empty
func TestUploadAttachments_Defaults(t *testing.T) {

	// Defaults: action "append", fieldname "file", maximum at least 1.
	step, err := NewUploadAttachments(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "append", step.Action)
	require.Equal(t, "file", step.Fieldname)
	require.Equal(t, 1, step.Maximum)

	// An unknown action falls back to "append".
	step, err = NewUploadAttachments(mapof.Any{"action": "bogus"})
	require.Nil(t, err)
	require.Equal(t, "append", step.Action)
}

// TestUploadAttachments_Minimum verifies that the minimum defaults to 0, never goes below it, and never exceeds the maximum
func TestUploadAttachments_Minimum(t *testing.T) {

	step, err := NewUploadAttachments(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, 0, step.Minimum)

	step, err = NewUploadAttachments(mapof.Any{"minimum": 2, "maximum": 5})
	require.Nil(t, err)
	require.Equal(t, 2, step.Minimum)
	require.Equal(t, 5, step.Maximum)

	step, err = NewUploadAttachments(mapof.Any{"minimum": 1, "maximum": 1})
	require.Nil(t, err)
	require.Equal(t, 1, step.Minimum)

	step, err = NewUploadAttachments(mapof.Any{"minimum": -3})
	require.Nil(t, err)
	require.Equal(t, 0, step.Minimum)

	// A minimum above the maximum (which defaults to 1) fails to load
	_, err = NewUploadAttachments(mapof.Any{"minimum": 2})
	require.Error(t, err)

	_, err = NewUploadAttachments(mapof.Any{"minimum": 6, "maximum": 5})
	require.Error(t, err)
}

// TestUploadAttachments_Actions verifies that only "replace" and "fill" survive parsing, and every other action becomes "append"
func TestUploadAttachments_Actions(t *testing.T) {

	for action, expected := range map[string]string{
		"replace": "replace",
		"fill":    "fill",
		"append":  "append",
		"":        "append",
		"unknown": "append",
	} {
		step, err := NewUploadAttachments(mapof.Any{"action": action})
		require.Nil(t, err)
		require.Equal(t, expected, step.Action, "action=%q", action)
	}
}
