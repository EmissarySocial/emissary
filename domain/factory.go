package domain

import (
	"context"
	"strings"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/EmissarySocial/emissary/tools/httpcache"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/form"
	"github.com/benpate/icon"
	"github.com/benpate/mediaserver"
	"github.com/benpate/steranko"
	"github.com/benpate/steranko/plugin/hash"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Factory knows how to create an populate all services
type Factory struct {
	Session   data.Session
	config    config.Domain
	port      string
	providers []config.Provider

	// services (from server)
	activityCache       *mongo.Collection
	contentService      *service.Content
	httpCache           *httpcache.HTTPCache
	providerService     *service.Provider
	registrationService *service.Registration
	queue               *queue.Queue
	templateService     *service.Template
	themeService        *service.Theme
	widgetService       *service.Widget
	workingDirectory    *mediaserver.WorkingDirectory

	// Upload Directories (from server)
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs
	exportCache         afero.Fs

	// services (within this domain/factory)
	activityService        service.ActivityStream
	attachmentService      service.Attachment
	connectionService      service.Connection
	domainService          service.Domain
	emailService           service.DomainEmail
	encryptionKeyService   service.EncryptionKey
	folderService          service.Folder
	followerService        service.Follower
	followingService       service.Following
	groupService           service.Group
	inboxService           service.Inbox
	jwtService             service.JWT
	locatorService         service.Locator
	mentionService         service.Mention
	merchantAccountService service.MerchantAccount
	oauthClient            service.OAuthClient
	oauthUserToken         service.OAuthUserToken
	outboxService          service.Outbox
	responseService        service.Response
	ruleService            service.Rule
	searchDomainService    service.SearchDomain
	searchNotifierService  service.SearchNotifier
	searchQueryService     service.SearchQuery
	searchTagService       service.SearchTag
	searchResultService    service.SearchResult
	streamService          service.Stream
	streamArchiveService   service.StreamArchive
	streamDraftService     service.StreamDraft
	subscriberService      service.Subscriber
	subscriptionService    service.Subscription
	realtimeBroker         RealtimeBroker
	userService            service.User
	webhookService         service.Webhook

	// real-time watchers
	refreshContext   context.CancelFunc
	sseUpdateChannel chan primitive.ObjectID

	MarkForDeletion bool
}

// NewFactory creates a new factory tied to a MongoDB database
func NewFactory(domain config.Domain, port string, providers []config.Provider, activityCache *mongo.Collection, registrationService *service.Registration, serverEmail *service.ServerEmail, themeService *service.Theme, templateService *service.Template, widgetService *service.Widget, contentService *service.Content, providerService *service.Provider, queue *queue.Queue, attachmentOriginals afero.Fs, attachmentCache afero.Fs, exportCache afero.Fs, httpCache *httpcache.HTTPCache, workingDirectory *mediaserver.WorkingDirectory) (*Factory, error) {

	log.Info().Msg("Starting domain: " + domain.Hostname)

	// Base Factory object
	factory := Factory{
		activityCache:       activityCache,
		registrationService: registrationService,
		themeService:        themeService,
		templateService:     templateService,
		widgetService:       widgetService,
		contentService:      contentService,
		providerService:     providerService,
		queue:               queue,
		workingDirectory:    workingDirectory,

		httpCache:           httpCache,
		attachmentOriginals: attachmentOriginals,
		attachmentCache:     attachmentCache,
		exportCache:         exportCache,
		sseUpdateChannel:    make(chan primitive.ObjectID, 1000),
		port:                port,
	}

	factory.config.Hostname = domain.Hostname

	// Services are created empty, then populated in a second "Refresh" step later.  This
	// servse two purposes:
	//
	// 1. It resolves the problem of circular dependencies
	// 2. It allows us to load (and reload) service configuration separately, as config files are loaded and changed.

	// Start the Realtime Broker
	factory.realtimeBroker = NewRealtimeBroker(&factory, factory.SSEUpdateChannel())

	// Create empty service pointers.  These will be populated in the Refresh() step.
	factory.activityService = service.NewActivityStream()
	factory.attachmentService = service.NewAttachment()
	factory.connectionService = service.NewConnection()
	factory.domainService = service.NewDomain()
	factory.emailService = service.NewDomainEmail(serverEmail)
	factory.encryptionKeyService = service.NewEncryptionKey()
	factory.folderService = service.NewFolder()
	factory.followerService = service.NewFollower()
	factory.followingService = service.NewFollowing()
	factory.groupService = service.NewGroup()
	factory.inboxService = service.NewInbox()
	factory.jwtService = service.NewJWT()
	factory.locatorService = service.NewLocator()
	factory.mentionService = service.NewMention()
	factory.merchantAccountService = service.NewMerchantAccount()
	factory.oauthClient = service.NewOAuthClient()
	factory.oauthUserToken = service.NewOAuthUserToken()
	factory.outboxService = service.NewOutbox()
	factory.responseService = service.NewResponse()
	factory.ruleService = service.NewRule()
	factory.searchDomainService = service.NewSearchDomain()
	factory.searchNotifierService = service.NewSearchNotifier()
	factory.searchQueryService = service.NewSearchQuery()
	factory.searchResultService = service.NewSearchResult()
	factory.searchTagService = service.NewSearchTag()
	factory.streamService = service.NewStream()
	factory.streamArchiveService = service.NewStreamArchive()
	factory.streamDraftService = service.NewStreamDraft()
	factory.subscriberService = service.NewSubscriber()
	factory.subscriptionService = service.NewSubscription()
	factory.userService = service.NewUser()
	factory.webhookService = service.NewWebhook()

	// Refresh the configuration with values that (may) change during the lifetime of the factory
	if err := factory.Refresh(domain, providers, attachmentOriginals, attachmentCache); err != nil {
		return nil, derp.Wrap(err, "domain.NewFactory", "Error creating factory", domain)
	}

	// Start() is okay here because it will check for nil configuration before polling.
	go factory.followingService.Start()

	// Success!
	return &factory, nil
}

func (factory *Factory) Refresh(domain config.Domain, providers []config.Provider, attachmentOriginals afero.Fs, attachmentCache afero.Fs) error {

	// Update global pointers
	factory.attachmentOriginals = attachmentOriginals
	factory.attachmentCache = attachmentCache

	// If the database connect string has changed, then update the database connection
	if (factory.config.ConnectString != domain.ConnectString) || (factory.config.DatabaseName != domain.DatabaseName) {

		// If the connect string is empty, then we don't need to (re-)connect to a database
		if domain.ConnectString == "" {
			factory.config = domain
			return nil
		}

		// Fall through means we need to connect to the database
		server, err := mongodb.New(domain.ConnectString, domain.DatabaseName)

		if err != nil {
			return derp.Wrap(err, "domain.factory.UpdateConfig", "Error connecting to MongoDB (Server)", domain)
		}

		// Establish a connection
		session, err := server.Session(context.Background())

		if err != nil {
			return derp.Wrap(err, "domain.factory.UpdateConfig", "Error connecting to MongoDB (Session)", domain)
		}

		// If we already have a database connection, then close it
		if factory.Session != nil {
			factory.Session.Close()
		}

		refreshContext := factory.newRefreshContext()

		// Save the new session into the factory.
		factory.Session = session

		// REFRESH CACHED SERVICES

		// Populate Activity Stream Service
		factory.activityService.Refresh(
			factory.Domain(),
			factory.activityCache,
			factory.Hostname(),
		)

		// Populate Attachment Service
		factory.attachmentService.Refresh(
			factory.collection(CollectionAttachment),
			factory.MediaServer(),
			factory.Host(),
		)

		factory.connectionService.Refresh(
			factory.collection(CollectionConnection),
			factory.Provider(),
		)

		// Populate Domain Service
		factory.domainService.Refresh(
			factory.collection(CollectionDomain),
			domain,
			factory.ActivityStream(),
			factory.Connection(),
			factory.Provider(),
			factory.Registration(),
			factory.Theme(),
			factory.User(),
			build.FuncMap(factory.Icons()),
			factory.Hostname(),
		)

		// Populate EncryptionKey Service
		factory.encryptionKeyService.Refresh(
			factory.collection(CollectionEncryptionKey),
			factory.Host(),
		)

		// Populate Folder Service
		factory.folderService.Refresh(
			factory.collection(CollectionFolder),
			factory.Theme(),
			factory.Domain(),
			factory.Inbox(),
		)

		// Populate Follower Service
		factory.followerService.Refresh(
			factory.collection(CollectionFollower),
			factory.User(),
			factory.Stream(),
			factory.Rule(),
			factory.Email(),
			factory.ActivityStream(),
			factory.Queue(),
			factory.Host(),
		)

		// Populate Following Service
		factory.followingService.Refresh(
			factory.collection(CollectionFollowing),
			factory.Stream(),
			factory.User(),
			factory.Inbox(),
			factory.Folder(),
			factory.EncryptionKey(),
			factory.ActivityStream(),
			factory.Host(),
		)

		// Populate Group Service
		factory.groupService.Refresh(
			factory.collection(CollectionGroup),
		)

		// Populate Inbox Service
		factory.inboxService.Refresh(
			factory.collection(CollectionInbox),
			factory.Rule(),
			factory.Folder(),
			factory.Host(),
		)

		// Populate the JWT Key Service
		factory.jwtService.Refresh(
			factory.collection(CollectionJWT),
			domain.MasterKey,
		)

		// Populate the Locator service
		factory.locatorService.Refresh(
			factory.Domain(),
			factory.SearchDomain(),
			factory.SearchQuery(),
			factory.Stream(),
			factory.User(),
			factory.Host(),
		)

		// Populate Mention service
		factory.mentionService.Refresh(
			factory.collection(CollectionMention),
			factory.Rule(),
			factory.ActivityStream(),
			factory.Host(),
		)

		// Populate MerchantAccount Service
		factory.merchantAccountService.Refresh(
			factory.collection(CollectionMerchantAccount),
			domain.MasterKey,
		)

		// Populate OAuthClient
		factory.oauthClient.Refresh(
			factory.collection(CollectionOAuthClient),
			factory.OAuthUserToken(),
			factory.Host(),
		)

		// Populate OAuthUserToken
		factory.oauthUserToken.Refresh(
			factory.collection(CollectionOAuthUserToken),
			factory.OAuthClient(),
			factory.JWT(),
			factory.Host(),
		)

		// Populate Outbox Service
		factory.outboxService.Refresh(
			factory.collection(CollectionOutbox),
			factory.Stream(),
			factory.ActivityStream(),
			factory.Follower(),
			factory.Template(),
			factory.User(),
			factory.Email(),
			factory.Queue(),
		)

		// Populate RealtimeBroker Service
		factory.realtimeBroker.Refresh(
			factory.Follower(),
			factory.Queue(),
		)

		// Populate the Response Service
		factory.responseService.Refresh(
			factory.collection(CollectionResponse),
			factory.User(),
			factory.Outbox(),
			factory.Host(),
		)

		// Populate the Rule Service
		factory.ruleService.Refresh(
			factory.collection(CollectionRule),
			factory.Outbox(),
			factory.User(),
			factory.Queue(),
			factory.Host(),
		)

		// Populate the SearchDomain Service
		factory.searchDomainService.Refresh(
			factory.collection(CollectionSearchQuery),
			factory.Domain(),
			factory.Follower(),
			factory.Rule(),
			factory.SearchTag(),
			factory.ActivityStream(),
			factory.Queue(),
			factory.Host(),
		)

		// Populate the Search Notifier Service
		factory.searchNotifierService.Refresh(
			factory.SearchDomain(),
			factory.SearchResult(),
			factory.SearchQuery(),
			factory.Queue(),
			factory.Host(),
			refreshContext,
		)

		// Populate the SearchQuery Service
		factory.searchQueryService.Refresh(
			factory.collection(CollectionSearchQuery),
			factory.Domain(),
			factory.Follower(),
			factory.Rule(),
			factory.SearchTag(),
			factory.ActivityStream(),
			factory.Queue(),
			factory.Host(),
		)

		// Populate the Search Service
		factory.searchResultService.Refresh(
			factory.collection(CollectionSearchResult),
			factory.SearchTag(),
			factory.Host(),
		)

		// Populate the SearchTag Service
		factory.searchTagService.Refresh(
			factory.collection(CollectionSearchTag),
			factory.Host(),
		)

		// Populate Stream Service
		factory.streamService.Refresh(
			factory.collection(CollectionStream),
			factory.Domain(),
			factory.SearchTag(),
			factory.Template(),
			factory.StreamDraft(),
			factory.Outbox(),
			factory.Attachment(),
			factory.ActivityStream(),
			factory.Content(),
			factory.EncryptionKey(),
			factory.Follower(),
			factory.Rule(),
			factory.User(),
			factory.Webhook(),
			factory.MediaServer(),
			factory.Queue(),
			factory.Host(),
			factory.SSEUpdateChannel(),
		)

		// Populate StreamArchive Service
		factory.streamArchiveService.Refresh(
			factory.Stream(),
			factory.Attachment(),
			factory.MediaServer(),
			factory.exportCache,
			factory.Queue(),
			factory.Host(),
		)

		// Populate StreamDraft Service
		factory.streamDraftService.Refresh(
			factory.collection(CollectionStreamDraft),
			factory.Template(),
			factory.Stream(),
		)

		// Populate Subscriber Service
		factory.subscriberService.Refresh(
			factory.collection(CollectionSubscriber),
		)

		// Populate Subscription Service
		factory.subscriptionService.Refresh(
			factory.collection(CollectionSubscription),
		)

		// Populate User Service
		factory.userService.Refresh(
			factory.collection(CollectionUser),
			factory.collection(CollectionFollower),
			factory.collection(CollectionFollowing),
			factory.collection(CollectionRule),
			factory.Attachment(),
			factory.Domain(),
			factory.Email(),
			factory.Folder(),
			factory.Follower(),
			factory.Following(),
			factory.Inbox(),
			factory.EncryptionKey(),
			factory.Outbox(),
			factory.Response(),
			factory.Rule(),
			factory.SearchTag(),
			factory.Stream(),
			factory.Template(),
			factory.Webhook(),
			factory.Queue(),
			factory.ActivityStream(),
			factory.SSEUpdateChannel(),
			factory.Host(),
		)

		// Populate Webhook Service
		factory.webhookService.Refresh(
			factory.collection(CollectionWebhook),
			factory.Queue(),
		)

		// Watch for updates to Stream records
		go queries.WatchStreams(refreshContext, factory.collection(CollectionStream), factory.sseUpdateChannel)

		// Watch for updates to User records
		go queries.WatchUsers(refreshContext, factory.collection(CollectionUser), factory.sseUpdateChannel)

		// Run search notifications
		go factory.searchNotifierService.Run()
	}

	// Re-Populate Email Service
	// This is separate because it may change separately from the DNS
	factory.emailService.Refresh(
		domain,
		factory.Domain(),
		factory.Steranko(),
	)

	if err := factory.domainService.Start(); err != nil {
		return derp.Wrap(err, "domain.NewFactory", "Error starting domain service", domain)
	}

	factory.config = domain
	factory.providers = providers
	return nil
}

// Close disconnects any background processes before this factory is destroyed
func (factory *Factory) Close() {

	if factory.Session != nil {
		factory.Session.Close()
	}

	close(factory.sseUpdateChannel)

	factory.domainService.Close()
	factory.realtimeBroker.Close()
	factory.streamService.Close()
	factory.followingService.Close()
	factory.followerService.Close()
	factory.jwtService.Close()
	factory.userService.Close()
}

/******************************************
 * Domain Data Accessors
 ******************************************/

// ID implements the set.Set interface.  (Domains are indexed by their hostname)
func (factory *Factory) ID() string {
	return factory.config.Hostname
}

// Host returns the domain name AND protocol (probably HTTPS) => e.g. "https://example.com"
func (factory *Factory) Host() string {
	return domain.Protocol(factory.config.Hostname) + factory.config.Hostname + factory.port
}

// Hostname returns the domain name only (without a protocol) => e.g. "example.com
func (factory *Factory) Hostname() string {
	return factory.config.Hostname
}

// IsLocalhost returns TRUE if this is a local domain (localhost, *.local, etc)
func (factory *Factory) IsLocalhost() bool {
	return domain.IsLocalhost(factory.Hostname())
}

func (factory *Factory) Config() config.Domain {
	return factory.config
}

func (factory *Factory) Providers() set.Slice[config.Provider] {
	return factory.providers
}

/******************************************
 * Domain Model Services
 ******************************************/

func (factory *Factory) Model(name string) (service.ModelService, error) {

	switch strings.ToLower(name) {

	case "activity":
		return factory.Inbox(), nil

	case "folder":
		return factory.Folder(), nil

	case "follower":
		return factory.Follower(), nil

	case "following":
		return factory.Following(), nil

	case "merchantAccount":
		return factory.MerchantAccount(), nil

	case "stream":
		return factory.Stream(), nil

	case "subscriber":
		return factory.Subscriber(), nil

	case "subscription":
		return factory.Subscription(), nil

	case "user":
		return factory.User(), nil

	}

	return nil, derp.NewInternalError("domain.Factory.Model", "Unknown model", name)
}

func (factory *Factory) ActivityStream() *service.ActivityStream {
	return &factory.activityService
}

// Attachment returns a fully populated Attachment service
func (factory *Factory) Attachment() *service.Attachment {
	return &factory.attachmentService
}

// Rule returns a fully populated Rule service
func (factory *Factory) Rule() *service.Rule {
	return &factory.ruleService
}

// Domain returns a fully populated Domain service
func (factory *Factory) Domain() *service.Domain {
	return &factory.domainService
}

// Connection returns a fully populated Connection service
func (factory *Factory) Connection() *service.Connection {
	return &factory.connectionService
}

// EncryptionKey returns a fully populated EncryptionKey service
func (factory *Factory) EncryptionKey() *service.EncryptionKey {
	return &factory.encryptionKeyService
}

// Follower returns a fully populated Follower service
func (factory *Factory) Follower() *service.Follower {
	return &factory.followerService
}

// Following returns a fully populated Following service
func (factory *Factory) Following() *service.Following {
	return &factory.followingService
}

// Folder returns a fully populated Folder service
func (factory *Factory) Folder() *service.Folder {
	return &factory.folderService
}

// Geocode returns a fully populated Geocode service
func (factory *Factory) Geocode() service.Geocode {
	return service.NewGeocode(factory.Hostname(), factory.Queue(), factory.Connection())
}

// Group returns a fully populated Group service
func (factory *Factory) Group() *service.Group {
	return &factory.groupService
}

// Inbox returns a fully populated Inbox service
func (factory *Factory) Inbox() *service.Inbox {
	return &factory.inboxService
}

// MerchantAccount returns a fully populated MerchantAccount service
func (factory *Factory) MerchantAccount() *service.MerchantAccount {
	return &factory.merchantAccountService
}

// Mention returns a fully populated Mention service
func (factory *Factory) Mention() *service.Mention {
	return &factory.mentionService
}

// OAuthClient returns a fully populated OAuthClient service
func (factory *Factory) OAuthClient() *service.OAuthClient {
	return &factory.oauthClient
}

// OAuthUserToken returns a fully populated OAuthUserToken service
func (factory *Factory) OAuthUserToken() *service.OAuthUserToken {
	return &factory.oauthUserToken
}

// Outbox returns a fully populated Outbox service
func (factory *Factory) Outbox() *service.Outbox {
	return &factory.outboxService
}

// SearchDomain returns a fully populated SearchDomain service
func (factory *Factory) SearchDomain() *service.SearchDomain {
	return &factory.searchDomainService
}

// SearchResult returns a fully populated SearchResult service
func (factory *Factory) SearchResult() *service.SearchResult {
	return &factory.searchResultService
}

// SearchQuery returns a fully populated SearchQuery service
func (factory *Factory) SearchQuery() *service.SearchQuery {
	return &factory.searchQueryService
}

// SearchTag returns a fully populated SearchTag service
func (factory *Factory) SearchTag() *service.SearchTag {
	return &factory.searchTagService
}

// Stream returns a fully populated Stream service
func (factory *Factory) Stream() *service.Stream {
	return &factory.streamService
}

// StreamArchive returns a fully populated StreamArchive service
func (factory *Factory) StreamArchive() *service.StreamArchive {
	return &factory.streamArchiveService
}

// StreamDraft returns a fully populated StreamDraft service
func (factory *Factory) StreamDraft() *service.StreamDraft {
	return &factory.streamDraftService
}

// Subscriber returns a fully populated Subscriber service
func (factory *Factory) Subscriber() *service.Subscriber {
	return &factory.subscriberService
}

// Subscription returns a fully populated Subscription service
func (factory *Factory) Subscription() *service.Subscription {
	return &factory.subscriptionService
}

// Response returns a fully populated Response service
func (factory *Factory) Response() *service.Response {
	return &factory.responseService
}

// User returns a fully populated User service
func (factory *Factory) User() *service.User {
	return &factory.userService
}

// Widget returns a fully populated Widget service
func (factory *Factory) Widget() *service.Widget {
	return factory.widgetService
}

// Webhook returns a fully populated Webhook service
func (factory *Factory) Webhook() *service.Webhook {
	return &factory.webhookService
}

/******************************************
 * Render Objects
 ******************************************/

// Theme service manages global website themes (managed globally by the server.Factory)
func (factory *Factory) Theme() *service.Theme {
	return factory.themeService
}

// Template returns a fully populated Template service (managed globally by the server.Factory)
func (factory *Factory) Template() *service.Template {
	return factory.templateService
}

/******************************************
 * Real-Time Update Channels
 ******************************************/

// RealtimeBroker returns a new RealtimeBroker that can push stream updates to connected clients.
func (factory *Factory) RealtimeBroker() *RealtimeBroker {
	return &factory.realtimeBroker
}

// SSEUpdateChannel initializes a background watcher and returns a channel containing any streams that have changed.
func (factory *Factory) SSEUpdateChannel() chan primitive.ObjectID {
	return factory.sseUpdateChannel
}

/******************************************
 * Media Server
 ******************************************/

// MediaServer manages all file uploads
func (factory *Factory) MediaServer() mediaserver.MediaServer {

	// Wrap the remote cache in a local filesystem cache.
	// tempFS := afero.NewBasePathFs(afero.NewOsFs(), os.TempDir())
	// cacheFS := afero.NewCacheOnReadFs(factory.AttachmentCache(), tempFS, 10*time.Minute)
	// return mediaserver.New(factory.AttachmentOriginals(), cacheFS)

	return mediaserver.New(factory.AttachmentOriginals(), factory.AttachmentCache(), factory.workingDirectory)
}

// AttachmentOriginals returns a reference to the Filesystem where original attachment files are stored
func (factory *Factory) AttachmentOriginals() afero.Fs {
	return factory.getSubFolder(factory.attachmentOriginals, factory.Hostname())
}

// AttachmentCache returns a reference to the Filesystem where cached/manipulated attachment files are stored.
func (factory *Factory) AttachmentCache() afero.Fs {
	return factory.getSubFolder(factory.attachmentCache, factory.Hostname())
}

// getSubFolder guarantees that a subfolder exists within the provided afero.Fs, or panics
func (factory *Factory) getSubFolder(base afero.Fs, path string) afero.Fs {

	// Try to make a new subfolder at the chosen path (returns nil if already exists)
	if err := base.MkdirAll(path, 0777); err != nil {
		derp.Report(derp.Wrap(err, "domain.factory.getSubFolder", "Error creating subfolder", path))
		// panic(err)
	}

	// Return a filesystem pointing to the new subfolder.
	return afero.NewBasePathFs(base, path)
}

/******************************************
 * Other Non-Model Services
 ******************************************/

// Camper returns a fully initialized Camper client (for Activity Intents)
func (factory *Factory) Camper() camper.Camper {
	client := httpcache.NewHTTPClient(factory.HTTPCache())
	return camper.New(camper.WithClient(client))
}

// Content returns the Content transformation service
func (factory *Factory) Content() *service.Content {
	return factory.contentService
}

// Email returns the Domain Email service, which sends email on behalf of the domain
func (factory *Factory) Email() *service.DomainEmail {
	return &factory.emailService
}

// Key returns an instance of the Key Manager Service (KMS)
func (factory *Factory) JWT() *service.JWT {
	return &factory.jwtService
}

// Icons returns the icon manager service, which manages
// aliases for icons in the UI
func (factory *Factory) Icons() icon.Provider {
	return service.Icons{}
}

// Locator returns the locator service, which locates records based on their URLs
func (factory *Factory) Locator() *service.Locator {
	return &factory.locatorService
}

func (factory *Factory) HTTPCache() *httpcache.HTTPCache {
	return factory.httpCache
}

// Queue returns the Queue service, which manages background jobs
func (factory *Factory) Queue() *queue.Queue {
	return factory.queue
}

// Registration returns the Registration service, which managaes new user registrations
func (factory *Factory) Registration() *service.Registration {
	return factory.registrationService
}

// Steranko returns a fully populated Steranko adapter for the User service.
func (factory *Factory) Steranko() *steranko.Steranko {

	return steranko.New(
		service.NewSterankoUserService(factory.User(), factory.Email()),
		factory.JWT(),
		steranko.WithPasswordHasher(hash.BCrypt(15), hash.Plaintext{}),
	)
}

// LookupProvider returns a fully populated LookupProvider service
func (factory *Factory) LookupProvider(userID primitive.ObjectID) form.LookupProvider {
	return service.NewLookupProvider(
		factory.Domain(),
		factory.Folder(),
		factory.Group(),
		factory.Registration(),
		factory.SearchTag(),
		factory.Template(),
		factory.Theme(),
		userID,
	)
}

/******************************************
 * External APIs
 ******************************************/

// OAuth returns a fully populated OAuth service
func (factory *Factory) Provider() *service.Provider {
	return factory.providerService
}

// RSS returns a fully populated RSS service
func (factory *Factory) RSS() *service.RSS {
	return service.NewRSS(factory.Stream(), factory.Host())
}

// Other libraries to make it here eventually...
// Service APIs (like Twitter? Slack? Discord?, The FB?)

/******************************************
 * Helper Utilities
 ******************************************/

// collection returns a data.Collection that matches the requested name.
func (factory *Factory) collection(name string) data.Collection {

	if factory.Session == nil {
		return nil
	}
	return factory.Session.Collection(name)
}

// ModelService returns the correct service to use for this particular Model object
func (factory *Factory) ModelService(object data.Object) service.ModelService {

	switch object.(type) {

	case *model.Folder:
		return factory.Folder()

	case *model.Follower:
		return factory.Follower()

	case *model.Following:
		return factory.Following()

	case *model.MerchantAccount:
		return factory.MerchantAccount()

	case *model.Message:
		return factory.Inbox()

	case *model.Response:
		return factory.Response()

	case *model.Rule:
		return factory.Rule()

	case *model.Stream:
		return factory.Stream()

	case *model.Subscriber:
		return factory.Subscriber()

	case *model.Subscription:
		return factory.Subscription()

	default:
		return nil
	}
}

func (factory *Factory) newRefreshContext() context.Context {

	if factory.refreshContext != nil {
		factory.refreshContext()
	}

	ctx, cancelFunction := context.WithCancel(context.Background())

	factory.refreshContext = cancelFunction

	return ctx
}
