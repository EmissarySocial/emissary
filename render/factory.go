package render

import (
	"github.com/benpate/form"
	"github.com/benpate/ghost/service"
)

type Factory interface {
	FormLibrary() form.Library
	Layout() *service.Layout
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	Template() *service.Template
}
