package render

import "github.com/EmissarySocial/emissary/model"

type Widget struct {
	Widget model.StreamWidget
	Stream
}

func NewWidget(renderer Stream, widget model.StreamWidget) Widget {
	return Widget{
		Widget: widget,
		Stream: renderer,
	}
}
