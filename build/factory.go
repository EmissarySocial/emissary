package build

import (
	"net/http"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/httpcache"
	"github.com/benpate/data"
	"github.com/benpate/form"
	"github.com/benpate/icon"
	"github.com/benpate/mediaserver"
	"github.com/benpate/steranko"
	"github.com/benpate/turbine/queue"
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
	Circle() *service.Circle
	Connection() *service.Connection
	Folder() *service.Folder
	Following() *service.Following
	Follower() *service.Follower
	Geocode() service.Geocode
	Group() *service.Group
	Guest() *service.Guest
	Inbox() *service.Inbox
	Mention() *service.Mention
	MerchantAccount() *service.MerchantAccount
	Outbox() *service.Outbox
	Permission() *service.Permission
	Provider() *service.Provider
	Registration() *service.Registration
	Response() *service.Response
	Rule() *service.Rule
	SearchResult() *service.SearchResult
	SearchTag() *service.SearchTag
	Stream() *service.Stream
	StreamArchive() *service.StreamArchive
	StreamDraft() *service.StreamDraft
	Purchase() *service.Purchase
	Template() *service.Template
	Theme() *service.Theme
	User() *service.User
	Webhook() *service.Webhook
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
	Locator() *service.Locator
	LookupProvider(*http.Request, primitive.ObjectID) form.LookupProvider
	OAuthClient() *service.OAuthClient
	OAuthUserToken() *service.OAuthUserToken
	Queue() *queue.Queue
	Steranko() *steranko.Steranko
	SSEUpdateChannel() chan primitive.ObjectID
}
