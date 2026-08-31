package build

import (
	"net/http"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/realtime"
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

// Factory is used to locate all necessary services
type Factory interface {
	// Model Services
	Model(string) (service.ModelService, error)
	ActivityStream() *service.ActivityStream
	Annotation() *service.Annotation
	Attachment() *service.Attachment
	Circle() *service.Circle
	Connection() *service.Connection
	Collection() *service.Collection
	CollectionItem() *service.CollectionItem
	Export() *service.Export
	Folder() *service.Folder
	Following() *service.Following
	Follower() *service.Follower
	GeocodeAddress() service.GeocodeAddress
	GeocodeAutocomplete() service.GeocodeAutocomplete
	GeocodeNetwork() service.GeocodeNetwork
	GeocodeTiles() service.GeocodeTiles
	GeocodeTimezone() service.GeocodeTimezone
	Group() *service.Group
	Identity() *service.Identity
	Import() *service.Import
	ImportItem() *service.ImportItem
	MerchantAccount() *service.MerchantAccount
	KeyPackage() *service.KeyPackage
	NewsFeed() *service.NewsFeed
	Notification() *service.Notification
	Outbox() *service.Outbox
	Permission() *service.Permission
	Product() *service.Product
	Provider() *service.Provider
	PushSubscription() *service.PushSubscription
	Registration() *service.Registration
	Response() *service.Response
	Rule() *service.Rule
	SearchResult() *service.SearchResult
	SearchTag() *service.SearchTag
	Stream() *service.Stream
	StreamArchive() *service.StreamArchive
	StreamDraft() *service.StreamDraft
	Privilege() *service.Privilege
	Template() *service.Template
	Theme() *service.Theme
	User() *service.User
	Webhook() *service.Webhook
	WebPush() *service.WebPush
	Widget() *service.Widget

	// Other data services
	ClientIP(*http.Request) string
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
	LookupProvider(*http.Request, data.Session, primitive.ObjectID) form.LookupProvider
	OAuthClient() *service.OAuthClient
	OAuthUserToken() *service.OAuthUserToken
	Queue() *queue.Queue
	Steranko(data.Session) *steranko.Steranko
	SSEUpdateChannel() chan realtime.Message
}
