package build

import (
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
