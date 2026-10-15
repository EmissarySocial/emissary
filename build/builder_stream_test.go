package build

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
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

// treeFixture is a site map of Streams keyed by ID, with each Stream's next sibling by rank
type treeFixture struct {
	streams     map[primitive.ObjectID]model.Stream
	nextSibling map[primitive.ObjectID]primitive.ObjectID
	loads       int
}

// loadParent looks up a parent in the fixture, returning NotFound for anything else
func (fixture *treeFixture) loadParent(parentID primitive.ObjectID, parent *model.Stream) error {
	fixture.loads++
	if stream, ok := fixture.streams[parentID]; ok {
		*parent = stream
		return nil
	}
	return derp.NotFound("treeFixture.loadParent", "Not a stream", parentID)
}

// next returns a builder for the stream's next sibling, or an empty one
func (fixture *treeFixture) next(stream *model.Stream) Stream {
	if nextID, ok := fixture.nextSibling[stream.StreamID]; ok {
		sibling := fixture.streams[nextID]
		return Stream{_stream: &sibling}
	}
	return Stream{}
}

// TestNextAfterParent walks up the tree to the nearest ancestor with a next sibling.
func TestNextAfterParent(t *testing.T) {

	// owner -> A -> A1 -> A1a, and owner -> B; A1 has no next sibling, A does
	ownerID := primitive.NewObjectID()
	a := model.Stream{StreamID: primitive.NewObjectID(), ParentID: ownerID}
	b := model.Stream{StreamID: primitive.NewObjectID(), ParentID: ownerID}
	a1 := model.Stream{StreamID: primitive.NewObjectID(), ParentID: a.StreamID}
	a2 := model.Stream{StreamID: primitive.NewObjectID(), ParentID: a.StreamID}
	a1a := model.Stream{StreamID: primitive.NewObjectID(), ParentID: a1.StreamID}

	newFixture := func() *treeFixture {
		return &treeFixture{
			streams:     map[primitive.ObjectID]model.Stream{a.StreamID: a, b.StreamID: b, a1.StreamID: a1, a2.StreamID: a2, a1a.StreamID: a1a},
			nextSibling: map[primitive.ObjectID]primitive.ObjectID{a.StreamID: b.StreamID, a1.StreamID: a2.StreamID},
		}
	}

	t.Run("parent's sibling", func(t *testing.T) {
		fixture := newFixture()
		result := nextAfterParent(&a1a, fixture.loadParent, fixture.next)
		require.Equal(t, a2.StreamID, result._stream.StreamID)
	})

	t.Run("grandparent's sibling", func(t *testing.T) {
		fixture := newFixture()
		delete(fixture.nextSibling, a1.StreamID)
		result := nextAfterParent(&a1a, fixture.loadParent, fixture.next)
		require.Equal(t, b.StreamID, result._stream.StreamID)
	})

	t.Run("end of the tree", func(t *testing.T) {
		fixture := newFixture()
		result := nextAfterParent(&a2, fixture.loadParent, fixture.next)
		require.Equal(t, b.StreamID, result._stream.StreamID, "the last child of A continues to B")

		result = nextAfterParent(&b, fixture.loadParent, fixture.next)
		require.True(t, result.IsEmpty(), "a top-level Stream's owner is not a Stream")
	})

	t.Run("no parent", func(t *testing.T) {
		fixture := newFixture()
		orphan := model.Stream{StreamID: primitive.NewObjectID()}
		require.True(t, nextAfterParent(&orphan, fixture.loadParent, fixture.next).IsEmpty())
		require.Zero(t, fixture.loads, "a Stream with no parent loads nothing")
	})

	t.Run("parent loop", func(t *testing.T) {
		loopA := model.Stream{StreamID: primitive.NewObjectID()}
		loopB := model.Stream{StreamID: primitive.NewObjectID(), ParentID: loopA.StreamID}
		loopA.ParentID = loopB.StreamID
		fixture := &treeFixture{streams: map[primitive.ObjectID]model.Stream{loopA.StreamID: loopA, loopB.StreamID: loopB}}

		require.True(t, nextAfterParent(&loopA, fixture.loadParent, fixture.next).IsEmpty())
		require.Equal(t, maxTreeDepth, fixture.loads)
	})
}
