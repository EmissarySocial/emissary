package render

import (
	"github.com/benpate/form"
	"github.com/benpate/ghost/service"
)

type Factory interface {
	Attachment() *service.Attachment
	FormLibrary() form.Library
	Layout() *service.Layout
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	Template() *service.Template
}
