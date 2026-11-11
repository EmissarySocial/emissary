package step

import (
	"bytes"
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
	}, nil)
	require.Nil(t, err)
	require.Equal(t, "HTML", step.Format)
	require.True(t, step.RemoveHTML)
	require.True(t, step.AddLinks)
	require.True(t, step.AddTags)
	require.Equal(t, "tags", step.TagPath)

	// Empty format is allowed.
	step, err = NewProcessContent(mapof.Any{}, nil)
	require.Nil(t, err)
	require.Equal(t, "", step.Format)

	require.Equal(t, "process-content", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestProcessContent_InvalidFormat verifies that an invalid format is rejected
func TestProcessContent_InvalidFormat(t *testing.T) {
	_, err := NewProcessContent(mapof.Any{"format": "not-allowed"}, nil)
	require.NotNil(t, err)
}

// TestProcessContent_DeprecatedTagFieldsStillParse guards the backward-compatibility contract:
// the deprecated "add-tags"/"tag-path" options must still parse without error so that older
// (external) Templates that set them continue to load. They are ignored at build time.
func TestProcessContent_DeprecatedTagFieldsStillParse(t *testing.T) {
	step, err := NewProcessContent(mapof.Any{"add-tags": true, "tag-path": "/home?q="}, nil)
	require.Nil(t, err)
	require.True(t, step.AddTags)
	require.Equal(t, "/home?q=", step.TagPath)
}

// TestProcessContent_HardWraps verifies that "hard-wraps" parses as a template from a bool or a string, and defaults to "false"
func TestProcessContent_HardWraps(t *testing.T) {

	data := map[string]any{"wrap": true}

	tests := []struct {
		name     string
		stepInfo mapof.Any
		expected string
	}{
		{name: "missing", stepInfo: mapof.Any{"format": "MARKDOWN"}, expected: "false"},
		{name: "bool true", stepInfo: mapof.Any{"format": "MARKDOWN", "hard-wraps": true}, expected: "true"},
		{name: "string true", stepInfo: mapof.Any{"format": "MARKDOWN", "hard-wraps": "true"}, expected: "true"},
		{name: "string false", stepInfo: mapof.Any{"format": "MARKDOWN", "hard-wraps": "false"}, expected: "false"},
		{name: "template", stepInfo: mapof.Any{"format": "MARKDOWN", "hard-wraps": "{{.wrap}}"}, expected: "true"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			step, err := NewProcessContent(test.stepInfo, nil)
			require.Nil(t, err)
			require.NotNil(t, step.HardWraps)

			var buffer bytes.Buffer
			require.Nil(t, step.HardWraps.Execute(&buffer, data))
			require.Equal(t, test.expected, buffer.String())
		})
	}
}

// TestProcessContent_HardWrapsRequiresMarkdown verifies that "hard-wraps" beside any other format is a load-time error
func TestProcessContent_HardWrapsRequiresMarkdown(t *testing.T) {

	for _, format := range []string{"", "HTML", "EDITORJS"} {
		_, err := NewProcessContent(mapof.Any{"format": format, "hard-wraps": true}, nil)
		require.NotNil(t, err, format)
	}
}

// TestProcessContent_HardWrapsBadTemplate verifies that an unparseable "hard-wraps" template is a load-time error
func TestProcessContent_HardWrapsBadTemplate(t *testing.T) {
	_, err := NewProcessContent(mapof.Any{"format": "MARKDOWN", "hard-wraps": "{{.wrap"}, nil)
	require.NotNil(t, err)
}
