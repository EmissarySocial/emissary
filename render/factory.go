package render

import (
	"github.com/benpate/form"
	"github.com/benpate/nebula"
	"github.com/whisperverse/mediaserver"
	"github.com/whisperverse/whisperverse/service"
	"github.com/whisperverse/whisperverse/singleton"
)

// Factory is used to locate all necessary services
type Factory interface {
	Attachment() *service.Attachment
	ContentLibrary() *nebula.Library
	Domain() *service.Domain
	FormLibrary() *form.Library
	Group() *service.Group
	Layout() *singleton.Layout
	MediaServer() mediaserver.MediaServer
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	Subscription() *service.Subscription
	Template() *singleton.Template
	User() *service.User
}
