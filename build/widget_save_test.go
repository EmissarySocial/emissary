package build

import (
	"bytes"
	"os"
	"testing"
	"text/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/rosetta/mapof"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// markdownWidgetBuilder returns a Widget builder wrapping a StreamWidget that carries
// the shipped widget-markdown definition and the given Markdown source.  The Stream is
// left nil because the methods under test (object, schema, Data) never reach it.
func markdownWidgetBuilder(t *testing.T, source string) (Widget, *model.StreamWidget) {

	t.Helper()

	definition, err := os.ReadFile("../_embed/templates/widget-markdown/widget.hjson")
	require.NoError(t, err)

	widget := model.Widget{}
	require.NoError(t, hjson.Unmarshal(definition, &widget))

	streamWidget := model.NewStreamWidget("markdown", "Label", "TOP")
	streamWidget.Data = mapof.Any{"markdown": source}
	streamWidget.Widget = widget

	return NewWidget(nil, &streamWidget), &streamWidget
}

// TestWidgetBuilder_SetData confirms that the Widget builder's object and schema
// resolve a "data.{property}" path into the Widget's own data map, which is the
// exact mechanism the set-data step uses.
func TestWidgetBuilder_SetData(t *testing.T) {

	widgetBuilder, streamWidget := markdownWidgetBuilder(t, "# Title")

	require.NoError(t, widgetBuilder.schema().Set(widgetBuilder.object(), "data.html", "<p>Hello</p>"))
	require.Equal(t, "<p>Hello</p>", streamWidget.Data.GetString("html"))

	// The Markdown source must survive untouched
	require.Equal(t, "# Title", streamWidget.Data.GetString("markdown"))
}

// TestWidgetBuilder_SetData_UnknownProperty confirms that the Widget's schema
// rejects a property it does not declare, instead of writing arbitrary data.
func TestWidgetBuilder_SetData_UnknownProperty(t *testing.T) {

	widgetBuilder, streamWidget := markdownWidgetBuilder(t, "# Title")

	require.Error(t, widgetBuilder.schema().Set(widgetBuilder.object(), "data.nonsense", "value"))

	_, ok := streamWidget.Data.GetStringOK("nonsense")
	require.False(t, ok)
}

// TestWidgetBuilder_SetData_NoSchema confirms that a Widget declaring no schema
// still accepts data, rather than failing to resolve any path at all.
func TestWidgetBuilder_SetData_NoSchema(t *testing.T) {

	streamWidget := model.NewStreamWidget("test", "Label", "TOP")
	widgetBuilder := NewWidget(nil, &streamWidget)

	require.NoError(t, widgetBuilder.schema().Set(widgetBuilder.object(), "data.anything", "value"))
	require.Equal(t, "value", streamWidget.Data.GetString("anything"))
}

// TestWidgetBuilder_MarkdownPipeline confirms the full save pipeline for
// widget-markdown: the declared template renders the Widget's Markdown source
// into sanitized HTML, and the result lands in the Widget's data.
func TestWidgetBuilder_MarkdownPipeline(t *testing.T) {

	widgetBuilder, streamWidget := markdownWidgetBuilder(t, "# Title\n\n**bold** <script>alert(1)</script>")

	// Render the same template that widget.hjson declares
	valueTemplate, err := template.New("value").Funcs(step.FuncMap()).Parse("{{.Data.markdown | markdown}}")
	require.NoError(t, err)

	var buffer bytes.Buffer
	require.NoError(t, valueTemplate.Execute(&buffer, widgetBuilder))

	// Store the rendered value the way set-data does
	require.NoError(t, widgetBuilder.schema().Set(widgetBuilder.object(), "data.html", buffer.String()))

	html := streamWidget.Data.GetString("html")
	require.Contains(t, html, "<h1")
	require.Contains(t, html, "Title")
	require.Contains(t, html, "<strong>bold</strong>")
	require.NotContains(t, html, "<script")
	require.NotContains(t, html, "alert(1)")
}

// TestWidgetBuilder_MarkdownPipeline_KeepsEmbeds confirms that markup the application's
// policy allows -- an iframe embed -- survives all the way into stored data.  The schema
// must not re-sanitize with a stricter policy, which would silently delete the embed.
func TestWidgetBuilder_MarkdownPipeline_KeepsEmbeds(t *testing.T) {

	source := `Watch this:

<iframe src="https://www.youtube.com/embed/abc123" width="560" height="315" allowfullscreen></iframe>`

	widgetBuilder, streamWidget := markdownWidgetBuilder(t, source)

	valueTemplate, err := template.New("value").Funcs(step.FuncMap()).Parse("{{.Data.markdown | markdown}}")
	require.NoError(t, err)

	var buffer bytes.Buffer
	require.NoError(t, valueTemplate.Execute(&buffer, widgetBuilder))
	require.NoError(t, widgetBuilder.schema().Set(widgetBuilder.object(), "data.html", buffer.String()))

	html := streamWidget.Data.GetString("html")
	require.Contains(t, html, "<iframe")
	require.Contains(t, html, "https://www.youtube.com/embed/abc123")
	require.Contains(t, html, "Watch this:")
}

// TestWidgetBuilder_MarkdownPipeline_EmptySource confirms that empty Markdown
// produces no stored HTML, instead of an error.
func TestWidgetBuilder_MarkdownPipeline_EmptySource(t *testing.T) {

	widgetBuilder, streamWidget := markdownWidgetBuilder(t, "")

	valueTemplate, err := template.New("value").Funcs(step.FuncMap()).Parse("{{.Data.markdown | markdown}}")
	require.NoError(t, err)

	var buffer bytes.Buffer
	require.NoError(t, valueTemplate.Execute(&buffer, widgetBuilder))
	require.NoError(t, widgetBuilder.schema().Set(widgetBuilder.object(), "data.html", buffer.String()))

	require.Equal(t, "", streamWidget.Data.GetString("html"))
}

// TestWidgetBuilder_Data_NilMap confirms that a Widget with no data yet reports
// an empty map instead of panicking when a template reads from it.
func TestWidgetBuilder_Data_NilMap(t *testing.T) {

	streamWidget := model.StreamWidget{}
	widgetBuilder := NewWidget(nil, &streamWidget)

	require.Empty(t, widgetBuilder.Data())

	valueTemplate, err := template.New("value").Funcs(step.FuncMap()).Parse("{{.Data.markdown | markdown}}")
	require.NoError(t, err)

	var buffer bytes.Buffer
	require.NoError(t, valueTemplate.Execute(&buffer, widgetBuilder))
	require.Equal(t, "", buffer.String())
}
