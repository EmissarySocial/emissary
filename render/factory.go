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
	"github.com/benpate/nebula"
	"github.com/davidscottmills/goeditorjs"
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
	Mention() *service.Mention
	Queue() *queue.Queue
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	StreamUpdateChannel() chan model.Stream
	StripeClient() (client.API, error)
	Subscription() *service.Subscription
	Template() *service.Template
	User() *service.User

	// Other data services
	Config() config.Domain
	EditorJS() *goeditorjs.HTMLEngine
	Providers() set.Slice[config.Provider]
	LookupProvider() form.LookupProvider
	Host() string
	Hostname() string
	Icons() icon.Provider
}
