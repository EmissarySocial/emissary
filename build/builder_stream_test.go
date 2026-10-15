package build

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestStream_WidgetIDsByLocation asserts that the accessor yields exactly the string the widget
// editor's hidden inputs must carry: every widget in the location, in stored order, joined by
// commas, and nothing from any other location.
func TestStream_WidgetIDsByLocation(t *testing.T) {

	first := primitive.NewObjectID()
	second := primitive.NewObjectID()
	elsewhere := primitive.NewObjectID()

	stream := model.NewStream()
	stream.Widgets = model.NewStreamWidgets()
	stream.Widgets.Append(model.StreamWidget{StreamWidgetID: first, Type: "markdown", Location: "TOP"})
	stream.Widgets.Append(model.StreamWidget{StreamWidgetID: elsewhere, Type: "markdown", Location: "LEFT"})
	stream.Widgets.Append(model.StreamWidget{StreamWidgetID: second, Type: "markdown", Location: "TOP"})

	builder := Stream{_stream: &stream}

	require.Equal(t, first.Hex()+","+second.Hex(), builder.WidgetIDsByLocation("TOP"), "stored order, comma-joined")
	require.Equal(t, elsewhere.Hex(), builder.WidgetIDsByLocation("LEFT"), "a single widget has no separator")
	require.Equal(t, "", builder.WidgetIDsByLocation("RIGHT"), "an empty location is an empty string, not a stray comma")
	require.Equal(t, "", builder.WidgetIDsByLocation("NOWHERE"), "an unknown location is empty rather than an error")

	// A Stream that has never had a widget carries a nil slice, and must read the same as empty
	empty := model.Stream{}
	require.Equal(t, "", Stream{_stream: &empty}.WidgetIDsByLocation("TOP"))
}

// TestStream_Icon confirms that Icon returns the Stream's icon name, which widgets render with
// the "icon" template function, and an empty string when the Stream has none.
func TestStream_Icon(t *testing.T) {

	stream := model.NewStream()
	stream.Icon = "envelope"
	require.Equal(t, "envelope", Stream{_stream: &stream}.Icon())

	empty := model.NewStream()
	require.Equal(t, "", Stream{_stream: &empty}.Icon())
}

// appendWidgetData is the data each widget test template renders, with no Missing field
type appendWidgetData struct {
	Label string
}

// newAppendWidgetTemplate parses source as a widget's "widget" template
func newAppendWidgetTemplate(t *testing.T, source string) *template.Template {
	t.Helper()
	result, err := template.New("").Parse(`{{define "widget"}}` + source + `{{end}}`)
	require.NoError(t, err)
	return result
}

// TestAppendWidget_FailingWidgetLeavesNoMarkup renders a slot with a widget that fails partway
// between two that succeed, and keeps both good widgets and none of the failing one's tags.
func TestAppendWidget_FailingWidgetLeavesNoMarkup(t *testing.T) {

	good := newAppendWidgetTemplate(t, `<div class="good">{{.Label}}</div>`)
	failing := newAppendWidgetTemplate(t, `<div class="failing"><a href="/x">{{.Missing}}</a></div>`)
	data := appendWidgetData{Label: "Hello"}

	var buffer bytes.Buffer
	buffer.WriteString(`<div class="widgets">`)
	require.NoError(t, appendWidget(&buffer, good, data))
	require.Error(t, appendWidget(&buffer, failing, data), "a missing field is an execution error")
	require.NoError(t, appendWidget(&buffer, good, data))
	buffer.WriteString(`</div>`)

	result := buffer.String()
	require.Equal(t, `<div class="widgets"><div class="good">Hello</div><div class="good">Hello</div></div>`, result)
	require.NotContains(t, result, "failing")
	require.Equal(t, strings.Count(result, "<div"), strings.Count(result, "</div>"), "the slot stays balanced")
}

// TestAppendWidget_MissingDefinition appends nothing for a template with no "widget" definition.
func TestAppendWidget_MissingDefinition(t *testing.T) {

	undefined, err := template.New("").Parse(`<div class="stray">`)
	require.NoError(t, err)

	var buffer bytes.Buffer
	buffer.WriteString("before")

	require.Error(t, appendWidget(&buffer, undefined, appendWidgetData{}))
	require.Equal(t, "before", buffer.String(), "existing output is untouched")
}

// TestAppendWidget_EscapesData confirms that a widget's output is escaped by html/template on
// its way through the widget's own buffer.
func TestAppendWidget_EscapesData(t *testing.T) {

	widgetTemplate := newAppendWidgetTemplate(t, `<span>{{.Label}}</span>`)

	var buffer bytes.Buffer
	require.NoError(t, appendWidget(&buffer, widgetTemplate, appendWidgetData{Label: `<script>alert(1)</script>`}))

	require.Equal(t, `<span>&lt;script&gt;alert(1)&lt;/script&gt;</span>`, buffer.String())
}

// TestAppendWidget_EmptyOutput accepts a widget that renders nothing, as the Previous/Next
// widget does for a Stream with no neighbors.
func TestAppendWidget_EmptyOutput(t *testing.T) {

	widgetTemplate := newAppendWidgetTemplate(t, ``)

	var buffer bytes.Buffer
	buffer.WriteString("before")

	require.NoError(t, appendWidget(&buffer, widgetTemplate, appendWidgetData{}))
	require.Equal(t, "before", buffer.String())
}
