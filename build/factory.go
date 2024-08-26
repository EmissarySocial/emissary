package build

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/httpcache"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data"
	"github.com/benpate/form"
	"github.com/benpate/hannibal/queue"
	"github.com/benpate/icon"
	"github.com/benpate/mediaserver"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ServerFactory interface {
	ByDomainName(string) (Factory, error)
}

// Factory is used to locate all necessary services
type Factory interface {
	// Model Services
	Model(string) (service.ModelService, error)
	ActivityStream() *service.ActivityStream
	Attachment() *service.Attachment
	Connection() *service.Connection
	Folder() *service.Folder
	Following() *service.Following
	Follower() *service.Follower
	Group() *service.Group
	Inbox() *service.Inbox
	Mention() *service.Mention
	Outbox() *service.Outbox
	Provider() *service.Provider
	Registration() *service.Registration
	Response() *service.Response
	Rule() *service.Rule
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	Template() *service.Template
	Theme() *service.Theme
	User() *service.User
	Widget() *service.Widget

	// Other data services
	Config() config.Domain
	Content() *service.Content
	Domain() *service.Domain
	Email() *service.DomainEmail
	Host() string
	Hostname() string
	HTTPCache() *httpcache.HTTPCache
	Icons() icon.Provider
	MediaServer() mediaserver.MediaServer
	ModelService(data.Object) service.ModelService
	Locator() service.Locator
	LookupProvider(primitive.ObjectID) form.LookupProvider
	OAuthClient() *service.OAuthClient
	OAuthUserToken() *service.OAuthUserToken
	Providers() set.Slice[config.Provider]
	Queue() queue.Queue
	Steranko() *steranko.Steranko
	StreamUpdateChannel() chan primitive.ObjectID
}
