package domain

import (
	"context"
	"strings"

	"github.com/EmissarySocial/emissary/builder"
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/form"
	"github.com/benpate/hannibal/queue"
	"github.com/benpate/icon"
	"github.com/benpate/mediaserver"
	"github.com/benpate/steranko"
	"github.com/benpate/steranko/plugin/hash"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Factory knows how to create an populate all services
type Factory struct {
	Session   data.Session
	config    config.Domain
	providers []config.Provider

	// services (from server)
	themeService    *service.Theme
	templateService *service.Template
	widgetService   *service.Widget
	contentService  *service.Content
	providerService *service.Provider
	taskQueue       queue.Queue
	activityService *service.ActivityStream

	// Upload Directories (from server)
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs

	// services (within this domain/factory)
	attachmentService    service.Attachment
	ruleService          service.Rule
	groupService         service.Group
	domainService        service.Domain
	emailService         service.DomainEmail
	encryptionKeyService service.EncryptionKey
	folderService        service.Folder
	followerService      service.Follower
	followingService     service.Following
	inboxService         service.Inbox
	jwtService           service.JWT
	mentionService       service.Mention
	oauthClient          service.OAuthClient
	oauthUserToken       service.OAuthUserToken
	outboxService        service.Outbox
	responseService      service.Response
	streamService        service.Stream
	streamDraftService   service.StreamDraft
	realtimeBroker       RealtimeBroker
	userService          service.User

	// real-time watchers
	streamUpdateChannel chan model.Stream

	MarkForDeletion bool
}

// NewFactory creates a new factory tied to a MongoDB database
func NewFactory(domain config.Domain, providers []config.Provider, activityService *service.ActivityStream, serverEmail *service.ServerEmail, themeService *service.Theme, templateService *service.Template, widgetService *service.Widget, contentService *service.Content, providerService *service.Provider, taskQueue queue.Queue, attachmentOriginals afero.Fs, attachmentCache afero.Fs) (*Factory, error) {

	log.Info().Msg("Starting domain: " + domain.Hostname)

	// Base Factory object
	factory := Factory{
		themeService:    themeService,
		templateService: templateService,
		widgetService:   widgetService,
		contentService:  contentService,
		providerService: providerService,
		taskQueue:       taskQueue,
		activityService: activityService,

		attachmentOriginals: attachmentOriginals,
		attachmentCache:     attachmentCache,
		streamUpdateChannel: make(chan model.Stream),
	}

	factory.config.Hostname = domain.Hostname

	// Services are created empty, then populated in a second "Refresh" step later.  This
	// servse two purposes:
	//
	// 1. It resolves the problem of circular dependencies
	// 2. It allows us to load (and reload) service configuration separately, as config files are loaded and changed.

	// Start the Realtime Broker
	factory.realtimeBroker = NewRealtimeBroker(&factory, factory.StreamUpdateChannel())

	// Create empty service pointers.  These will be populated in the Refresh() step.
	factory.attachmentService = service.NewAttachment()
	factory.ruleService = service.NewRule()
	factory.domainService = service.NewDomain()
	factory.emailService = service.NewDomainEmail(serverEmail)
	factory.encryptionKeyService = service.NewEncryptionKey()
	factory.folderService = service.NewFolder()
	factory.followerService = service.NewFollower()
	factory.followingService = service.NewFollowing()
	factory.groupService = service.NewGroup()
	factory.mentionService = service.NewMention()
	factory.inboxService = service.NewInbox()
	factory.jwtService = service.NewJWT()
	factory.oauthClient = service.NewOAuthClient()
	factory.oauthUserToken = service.NewOAuthUserToken()
	factory.outboxService = service.NewOutbox()
	factory.responseService = service.NewResponse()
	factory.streamService = service.NewStream()
	factory.streamDraftService = service.NewStreamDraft()
	factory.userService = service.NewUser()

	// Start() is okay here because it will check for nil configuration before polling.
	go factory.followingService.Start()

	// Refresh the configuration with values that (may) change during the lifetime of the factory
	if err := factory.Refresh(domain, providers, attachmentOriginals, attachmentCache); err != nil {
		return nil, derp.Wrap(err, "domain.NewFactory", "Error creating factory", domain)
	}

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

		// Save the new session into the factory.
		factory.Session = session

		// REFRESH CACHED SERVICES

		// Populate Attachment Service
		factory.attachmentService.Refresh(
			factory.collection(CollectionAttachment),
			factory.MediaServer(),
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

		// Populate Domain Service
		factory.domainService.Refresh(
			factory.collection(CollectionDomain),
			domain,
			factory.Theme(),
			factory.User(),
			factory.Provider(),
			builder.FuncMap(factory.Icons()),
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
			factory.Rule(),
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
			[]byte(domain.KeyEncryptingKey),
		)

		// Populate Mention Service
		factory.mentionService.Refresh(
			factory.collection(CollectionMention),
			factory.Rule(),
			factory.ActivityStream(),
			factory.Host(),
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
			factory.Queue(),
		)

		// Populate RealtimeBroker Service
		factory.realtimeBroker.Refresh(
			factory.Follower(),
			factory.Queue(),
		)

		factory.responseService.Refresh(
			factory.collection(CollectionResponse),
			factory.User(),
			factory.Outbox(),
			factory.Host(),
		)

		// Populate Stream Service
		factory.streamService.Refresh(
			factory.collection(CollectionStream),
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
			factory.Host(),
			factory.StreamUpdateChannel(),
		)

		// Populate StreamDraft Service
		factory.streamDraftService.Refresh(
			factory.collection(CollectionStreamDraft),
			factory.Stream(),
		)

		// Populate User Service
		factory.userService.Refresh(
			factory.collection(CollectionUser),
			factory.collection(CollectionFollower),
			factory.collection(CollectionFollowing),
			factory.collection(CollectionRule),
			factory.Attachment(),
			factory.Rule(),
			factory.Email(),
			factory.EncryptionKey(),
			factory.Folder(),
			factory.Follower(),
			factory.Stream(),
			factory.Host(),
		)

		// Watch for updates to streams
		if session, ok := factory.Session.(*mongodb.Session); ok {
			if collection, ok := session.Collection("Stream").(*mongodb.Collection); ok {
				go service.WatchStreams(collection.Mongo(), factory.streamUpdateChannel)
			}
		}
	}

	// Re-Populate Email Service
	// This is separate because it may change separately from the DNS
	factory.emailService.Refresh(
		domain,
	)

	factory.config = domain
	factory.providers = providers
	return nil
}

// Close disconnects any background processes before this factory is destroyed
func (factory *Factory) Close() {

	if factory.Session != nil {
		factory.Session.Close()
	}

	close(factory.streamUpdateChannel)

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

// Host returns the domain name AND protocol (probably HTTPS) => "https://example.com")
func (factory *Factory) Host() string {
	return domain.Protocol(factory.config.Hostname) + factory.config.Hostname
}

// Hostname returns the domain name (without anything else) that this factory is responsible for
func (factory *Factory) Hostname() string {
	return factory.config.Hostname
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

	case "folder":
		return factory.Folder(), nil

	case "follower":
		return factory.Follower(), nil

	case "following":
		return factory.Following(), nil

	case "activity":
		return factory.Inbox(), nil

	}

	return nil, derp.NewInternalError("domain.Factory.Model", "Unknown model", name)
}

func (factory *Factory) ActivityStream() *service.ActivityStream {
	return factory.activityService
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

// Group returns a fully populated Group service
func (factory *Factory) Group() *service.Group {
	return &factory.groupService
}

// Inbox returns a fully populated Inbox service
func (factory *Factory) Inbox() *service.Inbox {
	return &factory.inboxService
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

// Stream returns a fully populated Stream service
func (factory *Factory) Stream() *service.Stream {
	return &factory.streamService
}

// StreamDraft returns a fully populated StreamDraft service
func (factory *Factory) StreamDraft() *service.StreamDraft {
	return &factory.streamDraftService
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

// StreamUpdateChannel initializes a background watcher and returns a channel containing any streams that have changed.
func (factory *Factory) StreamUpdateChannel() chan model.Stream {
	return factory.streamUpdateChannel
}

/******************************************
 * Media Server
 ******************************************/

// MediaServer manages all file uploads
func (factory *Factory) MediaServer() mediaserver.MediaServer {
	return mediaserver.New(factory.AttachmentOriginals(), factory.AttachmentCache())
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

// Content returns the Content transformation service
func (factory *Factory) Content() *service.Content {
	return factory.contentService
}

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

func (factory *Factory) Locator() service.Locator {
	return service.NewLocator(
		factory.User(),
		factory.Stream(),
		factory.Hostname(),
	)
}

// Queue returns the Queue service, which manages background jobs
func (factory *Factory) Queue() queue.Queue {
	return factory.taskQueue
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
	return service.NewLookupProvider(factory.Theme(), factory.Group(), factory.Folder(), userID)
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

	case *model.Rule:
		return factory.Rule()

	case *model.Folder:
		return factory.Folder()

	case *model.Follower:
		return factory.Follower()

	case *model.Following:
		return factory.Following()

	case *model.Message:
		return factory.Inbox()

	case *model.Response:
		return factory.Response()

	case *model.Stream:
		return factory.Stream()

	default:
		return nil
	}
}
