package tests

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// contentRawMarkdown is source that every content.raw format keeps, except the no-html fallback
const contentRawMarkdown = "First paragraph\n\n- one\n- two\n"

// contentRawEditorJS is an EditorJS document whose text carries a line break, bold, a link, and a color
const contentRawEditorJS = `{"blocks":[{"type":"paragraph","data":{"text":"Line one<br>Line <b>two</b> and <a href=\"https://example.com/\">a link</a> in <span class=\"cdx-text-color\" style=\"color: #0055cc;\">blue</span>"}}]}`

// TestContentRaw_SurvivesSave normalizes content through every Stream template's content schema, as
// Stream.Save does, and requires content.raw to come back unchanged
func TestContentRaw_SurvivesSave(t *testing.T) {

	// BUG-326: a template that restates content.raw with no format falls back to no-html,
	// which strips every tag out of EditorJS JSON and every newline out of Markdown
	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {
			requireContentRawSurvivesSave(t, loadGroup(t, group))
		})
	}
}

// requireContentRawSurvivesSave checks content.raw in every Stream template of one loaded group
func requireContentRawSurvivesSave(t *testing.T, loaded *loadedGroup) {

	t.Helper()

	checked := 0

	for _, templateID := range loaded.templateIDs() {

		template, err := loaded.Templates.Load(templateID)
		require.NoError(t, err)

		if _, isStream := template.NewObject().(*model.Stream); !isStream {
			continue
		}

		checked++

		// Only the content subtree, so a template's other required fields cannot fail the save
		element, ok := template.Schema.GetElement("content")
		require.True(t, ok, "%s: content is not in the schema", templateID)
		contentSchema := schema.New(element)

		for format, raw := range contentRawProbes(t, contentSchema) {
			content := model.Content{Format: format, Raw: raw}

			_, err := contentSchema.Normalize(&content)
			require.NoError(t, err, "%s: normalizing %s content", templateID, format)
			require.Equal(t, raw, content.Raw, "%s: %s content.raw changed on save", templateID, format)
		}
	}

	// A silent zero would pass without checking anything
	require.NotZero(t, checked, "no Stream templates loaded")
}

// contentRawProbes returns the raw values a content schema must keep, by content format
func contentRawProbes(t *testing.T, contentSchema schema.Schema) map[string]string {

	t.Helper()

	element, ok := contentSchema.GetElement("raw")
	require.True(t, ok, "content.raw is not in the schema")

	rawString, ok := element.(schema.String)
	require.True(t, ok, "content.raw is a %T, not a string", element)

	// A Markdown-validated raw refuses HTML on purpose, so it is probed with Markdown alone
	if strings.Contains(rawString.Format, "markdown") {
		return map[string]string{model.ContentFormatMarkdown: contentRawMarkdown}
	}

	return map[string]string{
		model.ContentFormatMarkdown: contentRawMarkdown,
		model.ContentFormatEditorJS: contentRawEditorJS,
	}
}
