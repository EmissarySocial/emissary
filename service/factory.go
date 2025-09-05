package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/EmissarySocial/emissary/tools/httpcache"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/form"
	"github.com/benpate/icon"
	"github.com/benpate/mediaserver"
	"github.com/benpate/steranko"
	"github.com/benpate/steranko/plugin/hash"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Factory knows how to create an populate all services
type Factory struct {
	serverFactory  ServerFactory
	commonDatabase mongodb.Server
	server         mongodb.Server
	config         config.Domain
	port           string

	// services (from server)
	contentService      *Content
	httpCache           *httpcache.HTTPCache
	jwtService          *JWT
	registrationService *Registration
	queue               *queue.Queue
	templateService     *Template
	themeService        *Theme
	widgetService       *Widget
	workingDirectory    *mediaserver.WorkingDirectory

	// Upload Directories (from server)
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs
	exportCache         afero.Fs

	// services (within this domain/factory)
	annotationService      Annotation
	attachmentService      Attachment
	circleService          Circle
	connectionService      Connection
	domainService          Domain
	emailService           DomainEmail
	encryptionKeyService   EncryptionKey
	folderService          Folder
	followerService        Follower
	followingService       Following
	groupService           Group
	identityService        Identity
	inboxService           Inbox
	locatorService         Locator
	mentionService         Mention
	merchantAccountService MerchantAccount
	oauthClient            OAuthClient
	oauthUserToken         OAuthUserToken
	outboxService          Outbox
	permissionService      Permission
	productService         Product
	providerService        Provider
	responseService        Response
	ruleService            Rule
	searchDomainService    SearchDomain
	searchNotifierService  SearchNotifier
	searchQueryService     SearchQuery
	searchTagService       SearchTag
	searchResultService    SearchResult
	streamService          Stream
	streamArchiveService   StreamArchive
	streamDraftService     StreamDraft
	privilegeService       Privilege
	realtimeBroker         realtime.Broker
	userService            User
	webhookService         Webhook

	// real-time watchers
	refreshContext   context.CancelFunc
	sseUpdateChannel chan primitive.ObjectID

	MarkForDeletion bool
}

// NewFactory creates a new factory tied to a MongoDB database
func NewFactory(serverFactory ServerFactory, commonDatabase mongodb.Server, domain config.Domain, port string, contentService *Content, emailService *ServerEmail, jwtService *JWT, queue *queue.Queue, registrationService *Registration, templateService *Template, themeService *Theme, widgetService *Widget, attachmentOriginals afero.Fs, attachmentCache afero.Fs, exportCache afero.Fs, httpCache *httpcache.HTTPCache, workingDirectory *mediaserver.WorkingDirectory) (*Factory, error) {

	log.Info().Msg("Starting domain: " + domain.Hostname)

	// Base Factory object
	factory := Factory{
		serverFactory:       serverFactory,
		commonDatabase:      commonDatabase,
		contentService:      contentService,
		jwtService:          jwtService,
		queue:               queue,
		registrationService: registrationService,
		themeService:        themeService,
		templateService:     templateService,
		widgetService:       widgetService,
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
	factory.realtimeBroker = realtime.NewBroker(factory.SSEUpdateChannel())

	// Create empty service pointers.  These will be populated in the Refresh() step.
	factory.annotationService = NewAnnotation(&factory)
	factory.attachmentService = NewAttachment()
	factory.circleService = NewCircle()
	factory.connectionService = NewConnection()
	factory.domainService = NewDomain(&factory)
	factory.emailService = NewDomainEmail(&factory, emailService)
	factory.encryptionKeyService = NewEncryptionKey()
	factory.folderService = NewFolder()
	factory.followerService = NewFollower(&factory)
	factory.followingService = NewFollowing(&factory)
	factory.groupService = NewGroup()
	factory.identityService = NewIdentity(&factory)
	factory.inboxService = NewInbox()
	factory.locatorService = NewLocator()
	factory.mentionService = NewMention(&factory)
	factory.merchantAccountService = NewMerchantAccount()
	factory.oauthClient = NewOAuthClient()
	factory.oauthUserToken = NewOAuthUserToken()
	factory.outboxService = NewOutbox(&factory)
	factory.permissionService = NewPermission(&factory)
	factory.productService = NewProduct()
	factory.providerService = NewProvider()
	factory.responseService = NewResponse()
	factory.ruleService = NewRule(&factory)
	factory.searchDomainService = NewSearchDomain(&factory)
	factory.searchNotifierService = NewSearchNotifier()
	factory.searchQueryService = NewSearchQuery(&factory)
	factory.searchResultService = NewSearchResult()
	factory.searchTagService = NewSearchTag()
	factory.streamService = NewStream(&factory)
	factory.streamArchiveService = NewStreamArchive()
	factory.streamDraftService = NewStreamDraft()
	factory.privilegeService = NewPrivilege()
	factory.userService = NewUser(&factory)
	factory.webhookService = NewWebhook(&factory)

	// Refresh the configuration with values that (may) change during the lifetime of the factory
	if err := factory.Refresh(domain, attachmentOriginals, attachmentCache); err != nil {
		return nil, derp.Wrap(err, "domain.NewFactory", "Error creating factory", domain)
	}

	// Success!
	return &factory, nil
}

func (factory *Factory) Refresh(domain config.Domain, attachmentOriginals afero.Fs, attachmentCache afero.Fs) error {

	// Update global pointers
	factory.attachmentOriginals = attachmentOriginals
	factory.attachmentCache = attachmentCache

	// If the database connect string has changed, then update the database connection
	if (factory.config.ConnectString != domain.ConnectString) || (factory.config.DatabaseName != domain.DatabaseName) {

		// If the connect string has not been changed, we don't need to (re-)connect to a database
		if domain.ConnectString == "" {
			factory.config = domain
			return nil
		}

		// Fall through means we need to connect to the database
		opt := options.Client()

		// Create a new server connection
		server, err := mongodb.New(domain.ConnectString, domain.DatabaseName, opt)

		if err != nil {
			return derp.Wrap(err, "domain.factory.UpdateConfig", "Error connecting to MongoDB (Server)", domain)
		}

		factory.server = server
		refreshContext := factory.newRefreshContext()

		// REFRESH CACHED SERVICES

		factory.annotationService.Refresh()

		// Populate Attachment Service
		factory.attachmentService.Refresh(
			factory.MediaServer(),
			factory.Host(),
		)

		// Populate Circle Service
		factory.circleService.Refresh(
			factory.Privilege(),
		)

		// Populate Connection Service
		factory.connectionService.Refresh(
			factory.Provider(),
			domain.MasterKey,
			factory.Host(),
		)

		// Populate Domain Service
		factory.domainService.Refresh(
			domain,
			factory.Connection(),
			factory.Provider(),
			factory.Registration(),
			factory.Theme(),
			factory.User(),
			FuncMap(factory.Icons()),
			factory.Hostname(),
		)

		// Populate EncryptionKey Service
		factory.encryptionKeyService.Refresh(
			factory.Host(),
		)

		// Populate Folder Service
		factory.folderService.Refresh(
			factory.Domain(),
			factory.Following(),
			factory.Inbox(),
			factory.Theme(),
		)

		// Populate Follower Service
		factory.followerService.Refresh(
			factory.User(),
			factory.Stream(),
			factory.Rule(),
			factory.Email(),
			factory.Queue(),
			factory.Host(),
		)

		// Populate Following Service
		factory.followingService.Refresh(
			factory.Stream(),
			factory.User(),
			factory.Inbox(),
			factory.Folder(),
			factory.EncryptionKey(),
			factory.Host(),
		)

		// Populate Group Service
		factory.groupService.Refresh()

		factory.identityService.Refresh(
			factory.Email(),
			factory.JWT(),
			factory.Privilege(),
			factory.Queue(),
			factory.Host(),
		)

		// Populate Inbox Service
		factory.inboxService.Refresh(
			factory.Rule(),
			factory.Folder(),
			factory.Host(),
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
			factory.Rule(),
			factory.Host(),
		)

		// Populate MerchantAccount Service
		factory.merchantAccountService.Refresh(
			factory.Circle(),
			factory.Connection(),
			factory.JWT(),
			factory.Identity(),
			factory.Privilege(),
			factory.Product(),
			factory.User(),
			domain.MasterKey,
			factory.Host(),
		)

		// Populate OAuthClient
		factory.oauthClient.Refresh(
			factory.OAuthUserToken(),
			factory.Host(),
		)

		// Populate OAuthUserToken
		factory.oauthUserToken.Refresh(
			factory.OAuthClient(),
			factory.JWT(),
			factory.Host(),
		)

		// Populate Outbox Service
		factory.outboxService.Refresh(
			factory.Follower(),
			factory.Identity(),
			factory.Rule(),
			factory.Stream(),
			factory.Template(),
			factory.User(),
			factory.Email(),
			factory.Queue(),
			factory.Hostname(),
		)

		// Populate Permission Service
		factory.permissionService.Refresh(
			factory.Identity(),
			factory.Privilege(),
			factory.User(),
		)

		// Populate Product Service
		factory.productService.Refresh(
			factory.MerchantAccount(),
		)

		// Populate RealtimeBroker Service
		factory.realtimeBroker.Refresh()

		// Populate the Response Service
		factory.responseService.Refresh(
			factory.Inbox(),
			factory.Outbox(),
			factory.User(),
			factory.Host(),
		)

		// Populate the Rule Service
		factory.ruleService.Refresh(
			factory.Outbox(),
			factory.User(),
			factory.Queue(),
			factory.Host(),
		)

		// Populate the SearchDomain Service
		factory.searchDomainService.Refresh(
			factory.Domain(),
			factory.Follower(),
			factory.Rule(),
			factory.SearchTag(),
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
		)

		// Populate the SearchQuery Service
		factory.searchQueryService.Refresh(
			factory.Domain(),
			factory.Follower(),
			factory.Rule(),
			factory.SearchTag(),
			factory.Queue(),
			factory.Host(),
		)

		// Populate the Search Service
		factory.searchResultService.Refresh(
			factory.SearchTag(),
			factory.Host(),
		)

		// Populate the SearchTag Service
		factory.searchTagService.Refresh(
			factory.Host(),
		)

		// Populate Stream Service
		factory.streamService.Refresh(
			factory.Circle(),
			factory.Domain(),
			factory.SearchTag(),
			factory.Template(),
			factory.StreamDraft(),
			factory.Outbox(),
			factory.Attachment(),
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
			factory.Template(),
			factory.Stream(),
		)

		// Populate Privilege Service
		factory.privilegeService.Refresh(
			factory.Circle(),
			factory.Identity(),
			factory.MerchantAccount(),
		)

		// Populate User Service
		factory.userService.Refresh(
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
			factory.SSEUpdateChannel(),
			factory.Host(),
		)

		// Populate Webhook Service
		factory.webhookService.Refresh(
			factory.Queue(),
		)

		// Watch for updates to Stream records
		go queries.WatchStreams(refreshContext, factory.server, factory.sseUpdateChannel)

		// Watch for updates to User records
		go queries.WatchUsers(refreshContext, factory.server, factory.sseUpdateChannel)
	}

	// Re-Populate Email Service
	// This is separate because it may change separately from the DNS
	factory.emailService.Refresh(
		domain,
		factory.Domain(),
	)

	if err := factory.domainService.Start(); err != nil {
		return derp.Wrap(err, "domain.NewFactory", "Error starting domain service", domain)
	}

	factory.config = domain
	return nil
}

// Close disconnects any background processes before this factory is destroyed
func (factory *Factory) Close() {

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

func (factory *Factory) Version() string {
	return "0.9.0"
}

// ID implements the set.Set interface.  (Domains are indexed by their hostname)
func (factory *Factory) ID() string {
	return factory.config.Hostname
}

// Host returns the domain name AND protocol (probably HTTPS) => e.g. "https://example.com"
func (factory *Factory) Host() string {
	return dt.Protocol(factory.config.Hostname) + factory.config.Hostname + factory.port
}

// Hostname returns the domain name only (without a protocol) => e.g. "example.com
func (factory *Factory) Hostname() string {
	return factory.config.Hostname
}

// IsLocalhost returns TRUE if this is a local domain (localhost, *.local, etc)
func (factory *Factory) IsLocalhost() bool {
	return dt.IsLocalhost(factory.Hostname())
}

func (factory *Factory) Config() config.Domain {
	return factory.config
}

/******************************************
 * Database Connection Methods
 ******************************************/

func (factory *Factory) CommonDatabase() mongodb.Server {
	return factory.commonDatabase
}

func (factory *Factory) Server() mongodb.Server {
	return factory.server
}

func (factory *Factory) Session(timeout time.Duration) (data.Session, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	session, err := factory.Server().Session(ctx)
	return session, cancel, err
}

func (factory *Factory) WithTransaction(ctx context.Context, callback data.TransactionCallbackFunc) (any, error) {
	return factory.server.WithTransaction(ctx, callback)
}

/******************************************
 * Domain Model Services
 ******************************************/

func (factory *Factory) Model(name string) (ModelService, error) {

	switch strings.ToLower(name) {

	case "activity":
		return factory.Inbox(), nil

	case "annotation":
		return factory.Annotation(), nil

	case "circle":
		return factory.Circle(), nil

	case "conversation":
		return factory.Conversation(), nil

	case "folder":
		return factory.Folder(), nil

	case "follower":
		return factory.Follower(), nil

	case "following":
		return factory.Following(), nil

	case "identity":
		return factory.Identity(), nil

	case "merchantAccount":
		return factory.MerchantAccount(), nil

	case "privilege":
		return factory.Privilege(), nil

	case "stream":
		return factory.Stream(), nil

	case "user":
		return factory.User(), nil

	}

	return nil, derp.InternalError("domain.Factory.Model", "Unknown model", name)
}

func (factory *Factory) ActivityStream(actorType string, actorID primitive.ObjectID) ActivityStream {
	return NewActivityStream(
		factory.serverFactory,
		factory.commonDatabase,
		factory,
		factory.Hostname(),
		factory.Version(),
		actorType,
		actorID,
	)
}

func (factory *Factory) ActivityStreamCrawler(actorType string, actorID primitive.ObjectID) ActivityStreamCrawler {

	activityStreamService := factory.ActivityStream(actorType, actorID)

	return NewActivityStreamCrawler(
		activityStreamService.Client(),
		factory.Queue().Enqueue,
		factory.Hostname(),
		actorType,
		actorID,
		4,
	)
}

// Annotation returns a fully populated Annotation service
func (factory *Factory) Annotation() *Annotation {
	return &factory.annotationService
}

// Attachment returns a fully populated Attachment service
func (factory *Factory) Attachment() *Attachment {
	return &factory.attachmentService
}

// Rule returns a fully populated Rule service
func (factory *Factory) Rule() *Rule {
	return &factory.ruleService
}

// Domain returns a fully populated Domain service
func (factory *Factory) Domain() *Domain {
	return &factory.domainService
}

// Circle returns a fully populated Circle service
func (factory *Factory) Circle() *Circle {
	return &factory.circleService
}

// Connection returns a fully populated Connection service
func (factory *Factory) Connection() *Connection {
	return &factory.connectionService
}

// Conversation returns a fully populated Conversation service
func (factory *Factory) Conversation() Conversation {
	return NewConversation()
}

// EncryptionKey returns a fully populated EncryptionKey service
func (factory *Factory) EncryptionKey() *EncryptionKey {
	return &factory.encryptionKeyService
}

// Follower returns a fully populated Follower service
func (factory *Factory) Follower() *Follower {
	return &factory.followerService
}

// Following returns a fully populated Following service
func (factory *Factory) Following() *Following {
	return &factory.followingService
}

// Folder returns a fully populated Folder service
func (factory *Factory) Folder() *Folder {
	return &factory.folderService
}

// Geocode returns a fully populated Geocode service
func (factory *Factory) Geocode() Geocode {
	return NewGeocode(factory.Hostname(), factory.Queue(), factory.Connection())
}

// Group returns a fully populated Group service
func (factory *Factory) Group() *Group {
	return &factory.groupService
}

// Identity returns a fully populated Identity service
func (factory *Factory) Identity() *Identity {
	return &factory.identityService
}

// Inbox returns a fully populated Inbox service
func (factory *Factory) Inbox() *Inbox {
	return &factory.inboxService
}

// MerchantAccount returns a fully populated MerchantAccount service
func (factory *Factory) MerchantAccount() *MerchantAccount {
	return &factory.merchantAccountService
}

// Mention returns a fully populated Mention service
func (factory *Factory) Mention() *Mention {
	return &factory.mentionService
}

// OAuthClient returns a fully populated OAuthClient service
func (factory *Factory) OAuthClient() *OAuthClient {
	return &factory.oauthClient
}

// OAuthUserToken returns a fully populated OAuthUserToken service
func (factory *Factory) OAuthUserToken() *OAuthUserToken {
	return &factory.oauthUserToken
}

// Outbox returns a fully populated Outbox service
func (factory *Factory) Outbox() *Outbox {
	return &factory.outboxService
}

// SearchDomain returns a fully populated SearchDomain service
func (factory *Factory) SearchDomain() *SearchDomain {
	return &factory.searchDomainService
}

// SearchResult returns a fully populated SearchResult service
func (factory *Factory) SearchResult() *SearchResult {
	return &factory.searchResultService
}

func (factory *Factory) SearchNotifier() *SearchNotifier {
	return &factory.searchNotifierService
}

// SearchQuery returns a fully populated SearchQuery service
func (factory *Factory) SearchQuery() *SearchQuery {
	return &factory.searchQueryService
}

// SearchTag returns a fully populated SearchTag service
func (factory *Factory) SearchTag() *SearchTag {
	return &factory.searchTagService
}

// Stream returns a fully populated Stream service
func (factory *Factory) Stream() *Stream {
	return &factory.streamService
}

// StreamArchive returns a fully populated StreamArchive service
func (factory *Factory) StreamArchive() *StreamArchive {
	return &factory.streamArchiveService
}

// StreamDraft returns a fully populated StreamDraft service
func (factory *Factory) StreamDraft() *StreamDraft {
	return &factory.streamDraftService
}

// Permission returns a fully populated Permission service
func (factory *Factory) Permission() *Permission {
	return &factory.permissionService
}

// Privilege returns a fully populated Privilege service
func (factory *Factory) Privilege() *Privilege {
	return &factory.privilegeService
}

// Product returns a fully populated Product service
func (factory *Factory) Product() *Product {
	return &factory.productService
}

// Response returns a fully populated Response service
func (factory *Factory) Response() *Response {
	return &factory.responseService
}

// User returns a fully populated User service
func (factory *Factory) User() *User {
	return &factory.userService
}

// Widget returns a fully populated Widget service
func (factory *Factory) Widget() *Widget {
	return factory.widgetService
}

// Webhook returns a fully populated Webhook service
func (factory *Factory) Webhook() *Webhook {
	return &factory.webhookService
}

/******************************************
 * Render Objects
 ******************************************/

// Theme service manages global website themes (managed globally by the server.Factory)
func (factory *Factory) Theme() *Theme {
	return factory.themeService
}

// Template returns a fully populated Template service (managed globally by the server.Factory)
func (factory *Factory) Template() *Template {
	return factory.templateService
}

/******************************************
 * Real-Time Update Channels
 ******************************************/

// RealtimeBroker returns a new RealtimeBroker that can push stream updates to connected clients.
func (factory *Factory) RealtimeBroker() *realtime.Broker {
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
func (factory *Factory) Content() *Content {
	return factory.contentService
}

// Email returns the Domain Email service, which sends email on behalf of the domain
func (factory *Factory) Email() *DomainEmail {
	return &factory.emailService
}

// Key returns an instance of the Key Manager Service (KMS)
func (factory *Factory) JWT() *JWT {
	return factory.jwtService
}

// Icons returns the icon manager service, which manages
// aliases for icons in the UI
func (factory *Factory) Icons() icon.Provider {
	return Icons{}
}

// Locator returns the locator service, which locates records based on their URLs
func (factory *Factory) Locator() *Locator {
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
func (factory *Factory) Registration() *Registration {
	return factory.registrationService
}

// Steranko returns a fully populated Steranko adapter for the User
func (factory *Factory) Steranko(session data.Session) *steranko.Steranko {

	return steranko.New(
		NewSterankoUserService(factory.Identity(), factory.User(), factory.Email(), session),
		factory.JWT(),
		steranko.WithPasswordHasher(hash.BCrypt(15), hash.Plaintext{}),
	)
}

// LookupProvider returns a fully populated LookupProvider service
func (factory *Factory) LookupProvider(request *http.Request, session data.Session, userID primitive.ObjectID) form.LookupProvider {
	return NewLookupProvider(factory, request, session, userID)
}

/******************************************
 * External APIs
 ******************************************/

// OAuth returns a fully populated OAuth service
func (factory *Factory) Provider() *Provider {
	return &factory.providerService
}

// RSS returns a fully populated RSS service
func (factory *Factory) RSS() *RSS {
	return NewRSS(factory.Stream(), factory.Host())
}

// Other libraries to make it here eventually...
// Service APIs (like Twitter? Slack? Discord?, The FB?)

/******************************************
 * Helper Utilities
 ******************************************/

// ModelService returns the correct service to use for this particular Model object
func (factory *Factory) ModelService(object data.Object) ModelService {

	switch object.(type) {

	case *model.Annotation:
		return factory.Annotation()

	case *model.Circle:
		return factory.Circle()

	case *model.Conversation:
		return factory.Conversation()

	case *model.Folder:
		return factory.Folder()

	case *model.Follower:
		return factory.Follower()

	case *model.Following:
		return factory.Following()

	case *model.Identity:
		return factory.Identity()

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

	case *model.Privilege:
		return factory.Privilege()

	default:
		derp.Report(derp.InternalError("factory.ModelService", "Unrecognized object type", object))
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
