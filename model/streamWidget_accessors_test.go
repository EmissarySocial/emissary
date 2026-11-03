package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestStreamWidgetSchema confirms that every StreamWidget field round-trips through its schema
func TestStreamWidgetSchema(t *testing.T) {

	s := schema.New(StreamWidgetSchema())
	widget := NewStreamWidget("", "", "")

	tests := []tableTestItem{
		{"streamWidgetId", "00000000000000000000000c", nil},
		{"type", "markdown", nil},
		{"location", "TOP", nil},
		{"label", "Markdown", nil},
		{"data.markdown", "# Hello", nil},
	}

	tableTest_Schema(t, &s, &widget, tests)
}

// TestStreamWidget_SetString confirms that SetString reports every write it makes, and only those
func TestStreamWidget_SetString(t *testing.T) {

	widget := NewStreamWidget("markdown", "Markdown", "TOP")
	originalID := widget.StreamWidgetID

	// A malformed ID is refused, and leaves the existing ID in place
	require.False(t, widget.SetString("streamWidgetId", "not-an-object-id"))
	require.Equal(t, originalID, widget.StreamWidgetID)

	// A valid ID is written, and the write is reported
	require.True(t, widget.SetString("streamWidgetId", "00000000000000000000000d"))
	require.Equal(t, "00000000000000000000000d", widget.StreamWidgetID.Hex())

	// Every other field is written, and each write is reported
	require.True(t, widget.SetString("type", "html"))
	require.True(t, widget.SetString("location", "BOTTOM"))
	require.True(t, widget.SetString("label", "HTML Code"))
	require.Equal(t, "html", widget.Type)
	require.Equal(t, "BOTTOM", widget.Location)
	require.Equal(t, "HTML Code", widget.Label)

	// An unknown field is refused
	require.False(t, widget.SetString("unknown", "value"))
}

// TestStreamSchema_WidgetsAreNormalized confirms that the Stream schema reaches every widget in
// the list, which is what Stream saves rely on to sanitize them
func TestStreamSchema_WidgetsAreNormalized(t *testing.T) {

	s := schema.New(StreamSchema())
	stream := newWidgetTestStream(
		NewStreamWidget("markdown", "<b>Markdown</b>", "TOP"),
		NewStreamWidget("html", "<i>HTML Code</i>", "BOTTOM"),
	)

	_, err := s.Normalize(&stream)
	require.NoError(t, err)

	require.Len(t, stream.Widgets, 2)
	require.Equal(t, "Markdown", stream.Widgets[0].Label)
	require.Equal(t, "HTML Code", stream.Widgets[1].Label)
	require.Equal(t, "TOP", stream.Widgets[0].Location)
	require.Equal(t, "BOTTOM", stream.Widgets[1].Location)
}

// TestStreamSchema_WidgetsAreValidated confirms that the Stream schema refuses a widget that
// breaks the StreamWidget schema
func TestStreamSchema_WidgetsAreValidated(t *testing.T) {

	s := schema.New(StreamSchema())

	// A Stream with no widgets, and one with a valid widget, both pass
	empty := newWidgetTestStream()
	_, err := s.Normalize(&empty)
	require.NoError(t, err)

	valid := newWidgetTestStream(NewStreamWidget("markdown", "Markdown", "TOP"))
	_, err = s.Normalize(&valid)
	require.NoError(t, err)

	// A location is a token, so a space is refused
	badLocation := newWidgetTestStream(NewStreamWidget("markdown", "Markdown", "TOP BOTTOM"))
	_, err = s.Normalize(&badLocation)
	require.Error(t, err)

	// Every widget names its type
	noType := newWidgetTestStream(StreamWidget{StreamWidgetID: primitive.NewObjectID(), Location: "TOP"})
	_, err = s.Normalize(&noType)
	require.Error(t, err)
}

// newWidgetTestStream returns a Stream that passes the Stream schema, holding the provided widgets
func newWidgetTestStream(widgets ...StreamWidget) Stream {

	stream := NewStream()
	stream.TemplateID = "test-post"

	for _, widget := range widgets {
		stream.Widgets.Append(widget)
	}

	return stream
}
