package render

import (
	"github.com/benpate/form"
	"github.com/benpate/mediaserver"
	"github.com/benpate/nebula"
	"github.com/whisperverse/whisperverse/service"
)

// Factory is used to locate all necessary services
type Factory interface {
	Attachment() *service.Attachment
	ContentLibrary() *nebula.Library
	Domain() *service.Domain
	FormLibrary() *form.Library
	Group() *service.Group
	Layout() *service.Layout
	MediaServer() mediaserver.MediaServer
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	Template() *service.Template
	User() *service.User
}
