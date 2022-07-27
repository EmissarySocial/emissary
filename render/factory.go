package render

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/mediaserver"
	"github.com/benpate/nebula"
	"github.com/stripe/stripe-go/v72/client"
)

// Factory is used to locate all necessary services
type Factory interface {
	Attachment() *service.Attachment
	ContentLibrary() *nebula.Library
	Domain() *service.Domain
	Group() *service.Group
	Layout() *service.Layout
	MediaServer() mediaserver.MediaServer
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	StreamUpdateChannel() chan model.Stream
	StripeClient() (client.API, error)
	Subscription() *service.Subscription
	Template() *service.Template
	User() *service.User

	// Other data services
	Host() string
	Hostname() string
}
