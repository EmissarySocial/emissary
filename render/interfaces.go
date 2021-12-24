package render

import (
	"github.com/benpate/content"
	"github.com/benpate/form"
	"github.com/benpate/ghost/service"
	"github.com/benpate/mediaserver"
)

// Factory is used to locate all necessary services
type Factory interface {
	Attachment() *service.Attachment
	ContentViewer() content.Widget
	ContentEditor(string) content.Widget
	Domain() *service.Domain
	FormLibrary() form.Library
	Group() *service.Group
	Layout() *service.Layout
	MediaServer() mediaserver.MediaServer
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	Template() *service.Template
	User() *service.User
}
