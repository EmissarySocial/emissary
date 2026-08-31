package service

import (
	"os"
	"strings"
	"testing"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// widgetEditorStub stands in for the Stream builder, supplying just the accessors that the
// widget editor's own markup reads.  Every list is empty on purpose: the assertions below are
// about the editor's fixed chrome, which is present whatever is placed in it.
type widgetEditorStub struct{}

// StreamID returns a stubbed stream ID, mirroring build.Stream.StreamID
func (stub widgetEditorStub) StreamID() string {
	return "000000000000000000000001"
}

// ListAllWidgets returns no widget definitions, mirroring build.Stream.ListAllWidgets
func (stub widgetEditorStub) ListAllWidgets() []form.LookupCode {
	return nil
}

// ListWidgetsByLocation returns no placed widgets, mirroring build.Stream.ListWidgetsByLocation
func (stub widgetEditorStub) ListWidgetsByLocation(location string) []mapof.Any {
	return nil
}

// TestWidgetEditor_Chrome renders the shared widget editor and asserts that the three pieces of
// chrome it draws unconditionally are actually there.
//
// Each one is a contract spanning files that cannot see each other.  The save marker is written
// by the hyperscript behavior and styled by the stylesheet, so the element it writes into has to
// exist in this markup under exactly that class -- and a save is otherwise SILENT, because
// StepSortWidgets answers with HX-Reswap:none, so losing the marker would look like nothing at
// all rather than like a bug.  The two sidebar notes are the only place the editor says that the
// page stacks below 768px; the canvas is a three-column row at every width and will never show
// it.
func TestWidgetEditor_Chrome(t *testing.T) {

	templateService := loadEmbeddedTemplates(t)

	template, exists := templateService.templatePrep["base-widget-editor"]
	require.True(t, exists, "base-widget-editor template not found")

	var buffer strings.Builder
	require.NoError(t, template.HTMLTemplate.ExecuteTemplate(&buffer, "widgets-list", widgetEditorStub{}))

	output := buffer.String()

	require.Contains(t, output, `class="widget-editor-status"`, "the save marker must be in the markup")
	require.Equal(t, 2, strings.Count(output, `class="widget-editor-note"`), "both sidebars must carry a narrow-screen note")
	require.Contains(t, output, "stack above the page content", "the LEFT note names where LEFT lands")
	require.Contains(t, output, "stack below the page content", "the RIGHT note names where RIGHT lands")
}

// TestWidgetEditor_SaveMarkerIsWired asserts that the three files behind the save marker agree on
// the class names they pass between each other.
//
// The behavior writes the classes, the stylesheet paints them, and neither one can see the other.
// A rename in either file leaves a marker that is written and never shown, or shown and never
// written, and nothing reports it: the save itself still succeeds.
func TestWidgetEditor_SaveMarkerIsWired(t *testing.T) {

	const directory = "../_embed/templates/base-widget-editor/"

	behavior, err := os.ReadFile(directory + "hyperscript/widgetEditor._hs")
	require.NoError(t, err)

	stylesheet, err := os.ReadFile(directory + "stylesheet/editor.css")
	require.NoError(t, err)

	for _, class := range []string{"widget-editor-status", "widget-editor-saved", "widget-editor-failed"} {
		require.Contains(t, string(behavior), class, "the behavior must write .%s", class)
		require.Contains(t, string(stylesheet), class, "the stylesheet must paint .%s", class)
	}

	// The marker reports failure as well as success, so both htmx outcomes have to be read: a
	// non-2xx response, and the 200 that WrapInlineError returns because htmx discards a bare 422.
	require.Contains(t, string(behavior), "successful", "the behavior must read detail.successful")
	require.Contains(t, string(behavior), "htmx-response-message", "the behavior must recognize a retargeted error")
}
