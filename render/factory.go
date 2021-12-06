package render

import (
	"github.com/benpate/form"
	"github.com/benpate/ghost/service"
	"github.com/benpate/mediaserver"
)

type Factory interface {
	Attachment() *service.Attachment
	FormLibrary() form.Library
	Layout() *service.Layout
	MediaServer() mediaserver.MediaServer
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	Template() *service.Template
}
