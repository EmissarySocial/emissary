package render

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queue"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/form"
	"github.com/benpate/icon"
	"github.com/benpate/mediaserver"
	"github.com/stripe/stripe-go/v72/client"
)

// Factory is used to locate all necessary services
type Factory interface {
	// Model Services
	Activity() *service.Activity
	Attachment() *service.Attachment
	Block() *service.Block
	Folder() *service.Folder
	Following() *service.Following
	Follower() *service.Follower
	Group() *service.Group
	Layout() *service.Layout
	Mention() *service.Mention
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	Template() *service.Template
	User() *service.User

	// Other data services
	Config() config.Domain
	Content() *service.Content
	Domain() *service.Domain
	Host() string
	Hostname() string
	Icons() icon.Provider
	MediaServer() mediaserver.MediaServer
	Locator() service.Locator
	LookupProvider() form.LookupProvider
	Providers() set.Slice[config.Provider]
	Publisher() service.Publisher
	Queue() *queue.Queue
	StreamUpdateChannel() chan model.Stream
	StripeClient() (client.API, error)
}
