package service

import (
	"context"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/EmissarySocial/emissary/tools/httpcache"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome"
	"github.com/benpate/form"
	"github.com/benpate/icon"
	"github.com/benpate/mediaserver"
	"github.com/benpate/steranko"
	"github.com/benpate/steranko/plugin/hash"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Factory knows how to create an populate all services
//
// RULE: Server-level resources that a config reload can REBUILD (the common database connection,
// the task queue) are never captured here.  They are read through serverFactory on every use --
// see CommonDatabase() and Queue() -- so a reload can never strand this domain on a dead handle.
type Factory struct {
	serverFactory ServerFactory
	server        mongodb.Server
	config        config.Domain
	port          string

	// services (from server)
	contentService      *Content
	httpCache           *httpcache.HTTPCache
	jwtService          *JWT
	registrationService *Registration
	templateService     *Template
	themeService        *Theme
	widgetService       *Widget
	workingDirectory    *mediaserver.WorkingDirectory

	// Upload Directories (from server)
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs
	exportCache         afero.Fs

	// services (within this domain/factory)
	activityStream          ActivityStream
	annotationService       Annotation
	attachmentService       Attachment
	circleService           Circle
	collectionService       Collection
	collectionItemService   CollectionItem
	connectionService       Connection
	domainService           Domain
	emailService            DomainEmail
	encryptionKeyService    EncryptionKey
	folderService           Folder
	followerService         Follower
	followingService        Following
	groupService            Group
	identityService         Identity
	importService           Import
	importItemService       ImportItem
	inboxService            Inbox
	keyPackageService       KeyPackage
	locatorService          Locator
	merchantAccountService  MerchantAccount
	newsFeedService         NewsFeed
	notificationService     Notification
	oauthClient             OAuthClient
	oauthUserToken          OAuthUserToken
	objectService           Object
	outboxService           Outbox
	outbox2Service          Outbox2
	permissionService       Permission
	productService          Product
	providerService         Provider
	pushSubscriptionService PushSubscription
	responseService         Response
	webPushService          WebPush
	ruleService             Rule
	ruleSuppressionService  RuleSuppression
	searchDomainService     SearchDomain
	searchQueryService      SearchQuery
	searchTagService        SearchTag
	searchResultService     SearchResult
	streamService           Stream
	streamArchiveService    StreamArchive
	streamDraftService      StreamDraft
	privilegeService        Privilege
	realtimeBroker          *realtime.Broker
	userService             User
	webhookService          Webhook

	// real-time watchers
	refreshContext   context.CancelFunc
	sseUpdateChannel chan realtime.Message

	MarkForDeletion bool
}

// NewFactory creates a new factory tied to a MongoDB database.  The common database and the task
// queue are NOT parameters: both can be rebuilt by a config reload, so the factory reads them
// through serverFactory on every use instead of capturing them here.  (The server email service
// is reached the same way -- see ServerEmail() -- so it is not a parameter either.)
func NewFactory(serverFactory ServerFactory, domain config.Domain, port string, contentService *Content, jwtService *JWT, registrationService *Registration, templateService *Template, themeService *Theme, widgetService *Widget, attachmentOriginals afero.Fs, attachmentCache afero.Fs, exportCache afero.Fs, httpCache *httpcache.HTTPCache, workingDirectory *mediaserver.WorkingDirectory) (*Factory, error) { // NOSONAR: this constructor really needs this many arguments.

	const location = "domain.factory.NewFactory"
	log.Info().Msg("Starting domain: " + domain.Hostname)

	// Base Factory object
	factory := Factory{
		serverFactory:       serverFactory,
		contentService:      contentService,
		jwtService:          jwtService,
		registrationService: registrationService,
		themeService:        themeService,
		templateService:     templateService,
		widgetService:       widgetService,
		workingDirectory:    workingDirectory,

		httpCache:           httpCache,
		attachmentOriginals: attachmentOriginals,
		attachmentCache:     attachmentCache,
		exportCache:         exportCache,
		sseUpdateChannel:    make(chan realtime.Message, 256),
		port:                port,
	}

	factory.config.Hostname = domain.Hostname

	// Create empty service pointers.  These will be populated in the Refresh() step.
	// This is so we can:
	// 1. resolve the problem of circular dependencies
	// 2. reload service configurations separate from the services themselves.

	factory.activityStream = NewActivityStream()
	factory.annotationService = NewAnnotation()
	factory.attachmentService = NewAttachment()
	factory.circleService = NewCircle()
	factory.collectionService = NewCollection()
	factory.collectionItemService = NewCollectionItem()
	factory.connectionService = NewConnection()
	factory.domainService = NewDomain()
	factory.emailService = NewDomainEmail()
	factory.encryptionKeyService = NewEncryptionKey()
	factory.folderService = NewFolder()
	factory.followerService = NewFollower()
	factory.followingService = NewFollowing()
	factory.groupService = NewGroup()
	factory.identityService = NewIdentity()
	factory.importService = NewImport()
	factory.importItemService = NewImportItem()
	factory.inboxService = NewInbox()
	factory.newsFeedService = NewNewsFeed()
	factory.keyPackageService = NewKeyPackage()
	factory.locatorService = NewLocator()
	factory.merchantAccountService = NewMerchantAccount()
	factory.notificationService = NewNotification()
	factory.oauthClient = NewOAuthClient()
	factory.oauthUserToken = NewOAuthUserToken()
	factory.objectService = NewObject()
	factory.outboxService = NewOutbox()
	factory.outbox2Service = NewOutbox2()
	factory.permissionService = NewPermission()
	factory.productService = NewProduct()
	factory.providerService = NewProvider()
	factory.pushSubscriptionService = NewPushSubscription()
	factory.webPushService = NewWebPush()
	factory.collectionItemService = NewCollectionItem()
	factory.responseService = NewResponse()
	factory.realtimeBroker = realtime.NewBroker(factory.SSEUpdateChannel())
	factory.ruleService = NewRule()
	factory.ruleSuppressionService = NewRuleSuppression()
	factory.searchDomainService = NewSearchDomain()
	factory.searchQueryService = NewSearchQuery()
	factory.searchResultService = NewSearchResult()
	factory.searchTagService = NewSearchTag()
	factory.streamService = NewStream()
	factory.streamArchiveService = NewStreamArchive()
	factory.streamDraftService = NewStreamDraft()
	factory.privilegeService = NewPrivilege()
	factory.userService = NewUser()
	factory.webhookService = NewWebhook()

	// Refresh the configuration with values that (may) change during the lifetime of the factory
	if err := factory.Refresh(domain, attachmentOriginals, attachmentCache); err != nil {
		return nil, derp.Wrap(err, location, "Creating factory", domain)
	}

	// Success!
	return &factory, nil
}

// Refresh applies a (possibly changed) domain configuration to this Factory and re-links every
// service to its dependencies.  It runs once from NewFactory and again on every config reload.
func (factory *Factory) Refresh(newConfig config.Domain, attachmentOriginals afero.Fs, attachmentCache afero.Fs) error {

	const location = "domain.factory.Refresh"

	// Track changes for additional steps below
	hasConfigChanged := factory.dbConfigChanged(newConfig) // nolint:scopeguard - this cached value is used below.

	// Update the factory with the new configuration
	factory.config = newConfig

	// Update global pointers
	factory.attachmentOriginals = attachmentOriginals
	factory.attachmentCache = attachmentCache

	// Refresh all services
	factory.activityStream.Refresh(factory)
	factory.annotationService.Refresh(factory)
	factory.attachmentService.Refresh(factory)
	factory.circleService.Refresh(factory)
	factory.collectionService.Refresh(factory)
	factory.collectionItemService.Refresh(factory)
	factory.connectionService.Refresh(factory)
	factory.domainService.Refresh(factory)
	factory.emailService.Refresh(factory)
	factory.encryptionKeyService.Refresh(factory)
	factory.folderService.Refresh(factory)
	factory.followerService.Refresh(factory)
	factory.followingService.Refresh(factory)
	factory.groupService.Refresh(factory)
	factory.identityService.Refresh(factory)
	factory.importService.Refresh(factory)
	factory.importItemService.Refresh(factory)
	factory.inboxService.Refresh(factory)
	factory.keyPackageService.Refresh(factory)
	factory.locatorService.Refresh(factory)
	factory.merchantAccountService.Refresh(factory)
	factory.newsFeedService.Refresh(factory)
	factory.notificationService.Refresh(factory)
	factory.oauthClient.Refresh(factory)
	factory.oauthUserToken.Refresh(factory)
	factory.objectService.Refresh(factory)
	factory.outboxService.Refresh(factory)
	factory.outbox2Service.Refresh(factory)
	factory.permissionService.Refresh(factory)
	factory.productService.Refresh(factory)
	factory.providerService.Refresh(factory)
	factory.pushSubscriptionService.Refresh(factory)
	factory.webPushService.Refresh(factory)
	factory.collectionItemService.Refresh(factory)
	factory.realtimeBroker.Refresh()
	factory.responseService.Refresh(factory)
	factory.ruleService.Refresh(factory)
	factory.ruleSuppressionService.Refresh(factory)
	factory.searchDomainService.Refresh(factory)
	factory.searchQueryService.Refresh(factory)
	factory.searchResultService.Refresh(factory)
	factory.searchTagService.Refresh(factory)
	factory.streamService.Refresh(factory)
	factory.streamArchiveService.Refresh(factory)
	factory.streamDraftService.Refresh(factory)
	factory.privilegeService.Refresh(factory)
	factory.userService.Refresh(factory)
	factory.webhookService.Refresh(factory)

	// If the database connect string has changed,
	// then reconnect to the new database
	if hasConfigChanged {

		// Use standard mongodb client options
		opts := options.Client()

		// Create a new server connection
		server, err := mongodb.New(newConfig.ConnectString, newConfig.DatabaseName, opts)

		if err != nil {
			return derp.Wrap(err, location, "Connecting to MongoDB (Server)", newConfig)
		}

		factory.server = server
		refreshContext := factory.newRefreshContext()

		// Start the domain service (load domain, upgrade collections, reindex collections)
		if err := factory.domainService.Start(); err != nil {
			return derp.Wrap(err, location, "Starting domain service", newConfig)
		}

		// REALTIME WATCHERS

		// Watch for updates to Import records
		go queries.WatchImports(refreshContext, factory.server, factory.sseUpdateChannel)

		// Watch for updates to Stream records
		go queries.WatchStreams(refreshContext, factory.server, factory.sseUpdateChannel)

		// Watch for updates to User records
		go queries.WatchUsers(refreshContext, factory.server, factory.sseUpdateChannel)
	}

	return nil
}

// Close disconnects any background processes before this factory is destroyed
func (factory *Factory) Close() {
	close(factory.sseUpdateChannel)
	factory.realtimeBroker.Close()
	factory.jwtService.Close()
}

/******************************************
 * Domain Data Accessors
 ******************************************/

// Version returns the current version of the Emissary server software
func (factory *Factory) Version() string {
	return "0.9.0"
}

// ID implements the set.Set interface.  (Domains are indexed by their hostname)
func (factory *Factory) ID() string {
	return factory.config.Hostname
}

// Host returns the domain name AND protocol (probably HTTPS) => e.g. "https://example.com"
func (factory *Factory) Host() string {
	return uri.GuessProtocolForHostname(factory.config.Hostname) + factory.config.Hostname + factory.port
}

// Hostname returns the domain name only (without a protocol) => e.g. "example.com
func (factory *Factory) Hostname() string {
	return factory.config.Hostname
}

// IsLocalhost returns TRUE if this is a local domain (localhost, *.local, etc)
func (factory *Factory) IsLocalhost() bool {
	return uri.IsLocalHostname(factory.Hostname())
}

// Config returns the domain configuration that this Factory currently runs under
func (factory *Factory) Config() config.Domain {
	return factory.config
}

/******************************************
 * Database Connection Methods
 ******************************************/

// CommonDatabase returns the CURRENT connection to the shared (ActivityPub Cache) database.  It
// reads through the server factory on every call -- deliberately never captured -- because a
// config reload can reconnect that database, and a captured handle would then fail every call
// with "client is disconnected" (this stranded the ActivityStream cache, which broke inbound
// signature verification with a bare 401).
func (factory *Factory) CommonDatabase() mongodb.Server {

	commonDatabase := factory.serverFactory.CommonDatabase()

	// While the server factory is disconnected (setup mode, FACTORY-MODES D7), return the zero
	// Server rather than letting mongodb.NewServer panic on a nil database.  Domain factories
	// only exist while the common database is connected, so this is defense in depth.
	if commonDatabase == nil {
		return mongodb.Server{}
	}

	return mongodb.NewServer(commonDatabase)
}

// Server returns the connection to this domain's OWN database (not the shared common database)
func (factory *Factory) Server() mongodb.Server {
	return factory.server
}

// Session returns a new data.Session using the primary database for this domain, using the specified timeout
func (factory *Factory) Session(timeout time.Duration) (data.Session, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	session, err := factory.Server().Session(ctx)

	if session == nil {
		err = derp.Internal("domain.Factory.Session", "Database session is nil")
	}

	return session, cancel, err
}

// WithTransaction executes the callback inside a database transaction, with the post-commit
// task spool attached: tasks emitted via postcommit.Publish during the transaction are held
// and released to the queue only after the transaction commits — a rolled-back transaction
// publishes nothing.  This is the ONLY way transactions should be opened; do not call
// factory.Server().WithTransaction directly.  (See emissary-specs/POST-COMMIT-TASKS-DESIGN.md)
func (factory *Factory) WithTransaction(ctx context.Context, callback data.TransactionCallbackFunc) (any, error) {
	return postcommit.WithTransaction(ctx, factory.server, factory.Queue(), callback)
}

/******************************************
 * Domain Model Services
 ******************************************/

// ActivityStream returns a fully populated ActivityStream service
func (factory *Factory) ActivityStream() *ActivityStream {
	return &factory.activityStream
}

// Annotation returns a fully populated Annotation service
func (factory *Factory) Annotation() *Annotation {
	return &factory.annotationService
}

// Attachment returns a fully populated Attachment service
func (factory *Factory) Attachment() *Attachment {
	return &factory.attachmentService
}

// Circle returns a fully populated Circle service
func (factory *Factory) Circle() *Circle {
	return &factory.circleService
}

// Collection returns a fully populated Collection service
func (factory *Factory) Collection() *Collection {
	return &factory.collectionService
}

// CollectionItem returns a fully populated CollectionItem service
func (factory *Factory) CollectionItem() *CollectionItem {
	return &factory.collectionItemService
}

// Connection returns a fully populated Connection service
func (factory *Factory) Connection() *Connection {
	return &factory.connectionService
}

// Domain returns a fully populated Domain service
func (factory *Factory) Domain() *Domain {
	return &factory.domainService
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

// GeocodeAddress returns a fully populated Geocode service
func (factory *Factory) GeocodeAddress() GeocodeAddress {
	return NewGeocodeAddress(factory.Hostname(), factory.Queue(), factory.Connection(), factory.GeocodeTimezone())
}

// GeocodeAutocomplete returns a fully populated Geocode service
func (factory *Factory) GeocodeAutocomplete() GeocodeAutocomplete {
	return NewGeocodeAutocomplete(factory.Connection(), factory.Hostname())
}

// GeocodeNetwork returns a fully populated Geocode service
func (factory *Factory) GeocodeNetwork() GeocodeNetwork {
	return NewGeocodeNetwork(factory.Connection(), factory.Queue(), factory.Hostname())
}

// GeocodeTiles returns a fully populated Geocode service
func (factory *Factory) GeocodeTiles() GeocodeTiles {
	return NewGeocodeTiles(factory.Connection(), factory.Hostname())
}

// GeocodeTimezone returns a fully populated Geocode service
func (factory *Factory) GeocodeTimezone() GeocodeTimezone {
	return NewGeocodeTimezone(factory.Connection(), factory.Hostname())
}

// Group returns a fully populated Group service
func (factory *Factory) Group() *Group {
	return &factory.groupService
}

// Identity returns a fully populated Identity service
func (factory *Factory) Identity() *Identity {
	return &factory.identityService
}

// Import returns the Import service, which imports user records from other systems
func (factory *Factory) Import() *Import {
	return &factory.importService
}

// ImportItem returns the ImportItem service, which manages individual records to be imported
func (factory *Factory) ImportItem() *ImportItem {
	return &factory.importItemService
}

// Inbox returns a fully populated Inbox service
func (factory *Factory) Inbox() *Inbox {
	return &factory.inboxService
}

// KeyPackage returns a fully populated KeyPackage service
func (factory *Factory) KeyPackage() *KeyPackage {
	return &factory.keyPackageService
}

// MerchantAccount returns a fully populated MerchantAccount service
func (factory *Factory) MerchantAccount() *MerchantAccount {
	return &factory.merchantAccountService
}

// Notification returns a fully populated Notification service
func (factory *Factory) Notification() *Notification {
	return &factory.notificationService
}

// NewsFeed returns a fully populated NewsFeed service
func (factory *Factory) NewsFeed() *NewsFeed {
	return &factory.newsFeedService
}

// OAuthClient returns a fully populated OAuthClient service
func (factory *Factory) OAuthClient() *OAuthClient {
	return &factory.oauthClient
}

// OAuthUserToken returns a fully populated OAuthUserToken service
func (factory *Factory) OAuthUserToken() *OAuthUserToken {
	return &factory.oauthUserToken
}

// Object returns a fully populated Object service
func (factory *Factory) Object() *Object {
	return &factory.objectService
}

// Outbox returns a fully populated Outbox service
func (factory *Factory) Outbox() *Outbox {
	return &factory.outboxService
}

// Outbox2 returns a fully populated Outbox2 service
// This is a temporary name that will be merged into Outbox
// once I know WTF I'm doing.
func (factory *Factory) Outbox2() *Outbox2 {
	return &factory.outbox2Service
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

// Rule returns a fully populated Rule service
func (factory *Factory) Rule() *Rule {
	return &factory.ruleService
}

// RuleSuppression returns a fully populated RuleSuppression service
func (factory *Factory) RuleSuppression() *RuleSuppression {
	return &factory.ruleSuppressionService
}

// SearchDomain returns a fully populated SearchDomain service
func (factory *Factory) SearchDomain() *SearchDomain {
	return &factory.searchDomainService
}

// SearchResult returns a fully populated SearchResult service
func (factory *Factory) SearchResult() *SearchResult {
	return &factory.searchResultService
}

// SearchQuery returns a fully populated SearchQuery service
func (factory *Factory) SearchQuery() *SearchQuery {
	return &factory.searchQueryService
}

// SearchTag returns a fully populated SearchTag service
func (factory *Factory) SearchTag() *SearchTag {
	return &factory.searchTagService
}

// SendLocator returns a SendLocator bound to the provided session, which resolves outbound
// ActivityPub actors, signing keys, and recipients for this domain
func (factory *Factory) SendLocator(session data.Session) SendLocator {
	return NewSendLocator(factory, session)
}

// ServerEmail returns the server-level email service (read through the server factory)
func (factory *Factory) ServerEmail() *ServerEmail {
	return factory.serverFactory.Email()
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
	return factory.realtimeBroker
}

// SSEUpdateChannel initializes a background watcher and returns a channel containing any streams that have changed.
func (factory *Factory) SSEUpdateChannel() chan realtime.Message {
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
		derp.Report(derp.Wrap(err, "domain.factory.getSubFolder", "Creating subfolder", path))
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
	middleware := httpcache.NewHTTPMiddleware(factory.HTTPCache())
	return camper.New(camper.WithRoundTripper(middleware))
}

// ClientIP returns the real client IP for the provided request, using the server's
// configured client-IP strategy
func (factory *Factory) ClientIP(request *http.Request) string {
	return factory.serverFactory.ClientIP(request)
}

// DigitalDome returns the shared Digital Dome web-application firewall, which
// tracks and blocks abusive client IP addresses.
func (factory *Factory) DigitalDome() *dome.Dome {
	return factory.serverFactory.DigitalDome()
}

// Content returns the Content transformation service
func (factory *Factory) Content() *Content {
	return factory.contentService
}

// Email returns the Domain Email service, which sends email on behalf of the domain
func (factory *Factory) Email() *DomainEmail {
	return &factory.emailService
}

// Export returns the Export service, which can export user records to other systems
func (factory *Factory) Export() *Export {
	result := NewExport(factory)
	return &result
}

// FuncMap returns a template.FuncMap populated with functions for this domain
func (factory *Factory) FuncMap() template.FuncMap {
	return templates.FuncMap(factory.Icons())
}

// Icons returns the icon manager service, which manages
// aliases for icons in the UI
func (factory *Factory) Icons() icon.Provider {
	return Icons{}
}

// JWT returns an instance of the JWT Key Manager Service
func (factory *Factory) JWT() *JWT {
	return factory.jwtService
}

// Locator returns the locator service, which locates records based on their URLs
func (factory *Factory) Locator() *Locator {
	return &factory.locatorService
}

// HTTPCache returns the HTTP Cache service for this domain
func (factory *Factory) HTTPCache() *httpcache.HTTPCache {
	return factory.httpCache
}

// MasterKey returns the master key for this domain
func (factory *Factory) MasterKey() string {
	return factory.config.MasterKey
}

// Queue returns the Queue service, which manages background jobs.  It reads through the server
// factory on every call -- deliberately never captured -- because a config reload can rebuild the
// queue, and a captured pointer would then publish tasks into a stopped queue that silently drops
// them ("Turbine Queue: stopped").
func (factory *Factory) Queue() *queue.Queue {
	return factory.serverFactory.Queue()
}

// Registration returns the Registration service, which managaes new user registrations
func (factory *Factory) Registration() *Registration {
	return factory.registrationService
}

// Steranko returns a Steranko adapter for the provided database session.
func (factory *Factory) Steranko(session data.Session) *steranko.Steranko {

	// This is the ONLY place that password hashing policy is defined: BCrypt cost 12
	// creates all new password hashes (~200ms per hash: slow enough to resist offline
	// cracking, fast enough that signin latency and the CPU cost of a failed-signin
	// flood stay reasonable). The Plaintext fallback exists so that passwords
	// stored before hashing was enforced can still sign in -- steranko re-hashes
	// them on first use.

	return steranko.New(
		factory.SterankoUserService(session),
		factory.JWT(),
		steranko.WithSigninService(factory.SterankoSigninService(session)),
		steranko.WithPasswordHasher(hash.BCrypt(12), hash.Plaintext{}),
	)
}

// SterankoUserService returns a SterankoUserService adapter for the provided database session
func (factory *Factory) SterankoUserService(session data.Session) SterankoUserService {
	return NewSterankoUserService(factory.Identity(), factory.User(), factory.Email(), session)
}

// SterankoSigninService returns a SterankoSigninService adapter for the provided database session
func (factory *Factory) SterankoSigninService(session data.Session) SterankoSigninService {
	return NewSterankoSigninService(factory, session)
}

// LookupProvider returns the LookupProvider service for this UserID
func (factory *Factory) LookupProvider(request *http.Request, session data.Session, userID primitive.ObjectID) form.LookupProvider {
	return NewLookupProvider(factory, request, session, userID)
}

/******************************************
 * External APIs
 ******************************************/

// Provider returns a fully populated Provider service (external OAuth/API providers)
func (factory *Factory) Provider() *Provider {
	return &factory.providerService
}

// PushSubscription returns a fully populated PushSubscription service
func (factory *Factory) PushSubscription() *PushSubscription {
	return &factory.pushSubscriptionService
}

// WebPush returns a fully populated WebPush service
func (factory *Factory) WebPush() *WebPush {
	return &factory.webPushService
}

// RSS returns a fully populated RSS service
func (factory *Factory) RSS() *RSS {
	return NewRSS(factory.Stream(), factory.Host())
}

/******************************************
 * Helper Utilities
 ******************************************/

// ImportableLocator returns an ImportableLocator for this domain
func (factory *Factory) ImportableLocator() ImportableLocator {

	const location = "service.Factory.ImportableLocator"

	return func(name string) (Importable, error) {

		switch name {

		/* THESE TO BE ADDED ONCE WE HAVE OTHER SERVICES TO TEST WITH
		case "outbox":
			return NilImporter(), nil

		case "content":
			return NilImporter(), nil

		case "following":
			return NilImporter(), nil

		case "blocked":
			return NilImporter(), nil
		*/

		case "emissary:annotation":
			return factory.Annotation(), nil

		case "emissary:circle":
			return factory.Circle(), nil

		case "emissary:collection":
			return factory.Collection(), nil

		case "emissary:collectionItem":
			return factory.CollectionItem(), nil

		case "emissary:folder":
			return factory.Folder(), nil

		case "emissary:follower":
			return factory.Follower(), nil

		case "emissary:following":
			return factory.Following(), nil

		case "emissary:newsItem":
			return factory.NewsFeed(), nil

		case "emissary:merchantAccount":
			return factory.MerchantAccount(), nil

		case "emissary:outboxMessage":
			return factory.Outbox(), nil

		case "emissary:privilege":
			return factory.Privilege(), nil

		case "emissary:product":
			return factory.Product(), nil

		case "emissary:response":
			return factory.Response(), nil

		case "emissary:rule":
			return factory.Rule(), nil

		case "emissary:stream":
			return factory.Stream(), nil

		case "emissary:user":
			return factory.User(), nil

		}

		return nil, derp.Internal(location, "Unrecognized service name. This should never happen", name)
	}
}

// Model returns the ModelService that matches the provided name
func (factory *Factory) Model(name string) (ModelService, error) {

	const location = "domain.Factory.Model"

	switch strings.ToLower(name) {

	case "activity":
		return factory.NewsFeed(), nil

	case "annotation":
		return factory.Annotation(), nil

	case "circle":
		return factory.Circle(), nil

	case "collection":
		return factory.Collection(), nil

	case "folder":
		return factory.Folder(), nil

	case "follower":
		return factory.Follower(), nil

	case "following":
		return factory.Following(), nil

	case "identity":
		return factory.Identity(), nil

	case "keyPackage":
		return factory.KeyPackage(), nil

	case "merchantAccount":
		return factory.MerchantAccount(), nil

	case "oauthUserToken":
		return factory.OAuthUserToken(), nil

	case "privilege":
		return factory.Privilege(), nil

	case "stream":
		return factory.Stream(), nil

	case "user":
		return factory.User(), nil

	}

	return nil, derp.Internal(location, "Unknown model", name)
}

// ModelService returns the correct service to use for this particular Model object
func (factory *Factory) ModelService(object data.Object) ModelService {

	switch object.(type) {

	case *model.Annotation:
		return factory.Annotation()

	case *model.Circle:
		return factory.Circle()

	case *model.Folder:
		return factory.Folder()

	case *model.Follower:
		return factory.Follower()

	case *model.Following:
		return factory.Following()

	case *model.Identity:
		return factory.Identity()

	case *model.Import:
		return factory.Import()

	case *model.ImportItem:
		return factory.ImportItem()

	case *model.KeyPackage:
		return factory.KeyPackage()

	case *model.MerchantAccount:
		return factory.MerchantAccount()

	case *model.NewsItem:
		return factory.NewsFeed()

	case *model.OAuthUserToken:
		return factory.OAuthUserToken()

	case *model.Response:
		return factory.Response()

	case *model.Rule:
		return factory.Rule()

	case *model.Stream:
		return factory.Stream()

	case *model.Privilege:
		return factory.Privilege()

	default:
		derp.Report(derp.Internal("factory.ModelService", "Unrecognized object type", object))
		return nil
	}
}

// Collections returns a list of all collection names used by this domain
func (factory *Factory) Collections() []string {

	return []string{
		"Annotation",
		"Attachment",
		"Circle",
		"Connection",
		"Collection",
		"Domain",
		"EncryptionKey",
		"Folder",
		"Follower",
		"Following",
		"Group",
		"Identity",
		"NewsFeed",
		"JWT",
		// "KeyPackage",
		"Notification",
		"MerchantAccount",
		"OAuthClient",
		"OAuthUserToken",
		"Outbox",
		"Privilege",
		"Product",
		"PushSubscription",
		"Response",
		"Rule",
		"SearchQuery",
		"SearchResult",
		"SearchTag",
		"Stream",
		"StreamDraft",
		"StreamOutbox",
		"User",
		"Webhook",
	}
}

// newRefreshContext cancels any existing refresh context and returns a new one
func (factory *Factory) newRefreshContext() context.Context {

	if factory.refreshContext != nil {
		factory.refreshContext()
	}

	ctx, cancelFunction := context.WithCancel(context.Background())

	factory.refreshContext = cancelFunction

	return ctx
}

// dbConfigChanged returns TRUE if the database connection settings have changed
func (factory *Factory) dbConfigChanged(newConfig config.Domain) bool {

	// If we don't have a valid configuration yet, then nothing has changed
	if newConfig.ConnectString == "" {
		return false
	}

	if newConfig.DatabaseName == "" {
		return false
	}

	// Otherwise, return TRUE if new values are different from current ones
	if factory.config.ConnectString != newConfig.ConnectString {
		return true
	}

	if factory.config.DatabaseName != newConfig.DatabaseName {
		return true
	}

	return false
}
