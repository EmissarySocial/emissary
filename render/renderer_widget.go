package render

import (
	"github.com/EmissarySocial/emissary/model"
)

// Widget renderer is created by the "with-widget" action, and
// can execute additional action steps on a widget that is
// embedded in a stream.  To save the final result, you must
// call "save" on the stream itself, not within this widget.
type Widget struct {
	Widget model.StreamWidget
	*Stream
}

func NewWidget(renderer *Stream, widget model.StreamWidget) Widget {
	return Widget{
		Widget: widget,
		Stream: renderer,
	}
}
