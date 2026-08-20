package build

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// Widget builder scopes a pipeline to a single Widget that is embedded in a
// Stream, so that steps read and write the Widget's own data instead of the
// Stream's.  To save the final result, you must call "save" on the stream
// itself, not within this widget.
type Widget struct {
	Widget *model.StreamWidget
	*Stream
}

// NewWidget returns a fully initialized Widget builder.
func NewWidget(builder *Stream, widget *model.StreamWidget) Widget {
	return Widget{
		Widget: widget,
		Stream: builder,
	}
}

// Data returns the custom data stored in this Widget, which is how templates
// and pipeline steps read the Widget's own values.
func (w Widget) Data() mapof.Any {
	return w.Widget.Data
}

// object returns the StreamWidget being built, so that steps operate on the
// Widget rather than on its containing Stream.
func (w Widget) object() data.Object {
	return w.Widget
}

// schema returns a Schema that addresses this Widget's custom data.  The Widget
// definition's own schema is nested beneath "data" to match the shape of the
// StreamWidget, so pipeline steps address values as "data.{property}".
func (w Widget) schema() schema.Schema {

	element := w.Widget.Widget.Schema.Element

	// A Widget that declares no schema still accepts arbitrary data
	if element == nil {
		element = schema.Object{Wildcard: schema.Any{}}
	}

	return schema.New(schema.Object{
		Properties: schema.ElementMap{
			"data": element,
		},
	})
}
