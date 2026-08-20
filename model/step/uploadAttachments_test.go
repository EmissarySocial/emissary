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
