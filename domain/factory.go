package domain

import (
	"context"
	"fmt"
	"strings"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queue"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/form"
	"github.com/benpate/icon"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"github.com/spf13/afero"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Factory knows how to create an populate all services
type Factory struct {
	Session   data.Session
	config    config.Domain
	providers []config.Provider

	// services (from server)
	themeService           *service.Theme
	templateService        *service.Template
	widgetService          *service.Widget
	contentService         *service.Content
	providerService        *service.Provider
	taskQueue              *queue.Queue
	activityStreamsService *service.ActivityStreams

	// Upload Directories (from server)
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs

	// services (within this domain/factory)
	attachmentService     service.Attachment
	blockService          service.Block
	groupService          service.Group
	domainService         service.Domain
	emailService          service.DomainEmail
	encryptionKeyService  service.EncryptionKey
	folderService         service.Folder
	followerService       service.Follower
	followingService      service.Following
	inboxService          service.Inbox
	mentionService        service.Mention
	outboxService         service.Outbox
	responseService       service.Response
	streamService         service.Stream
	streamDraftService    service.StreamDraft
	streamResponseService service.StreamResponse
	realtimeBroker        RealtimeBroker
	userService           service.User

	// real-time watchers
	streamUpdateChannel chan model.Stream

	MarkForDeletion bool
}

// NewFactory creates a new factory tied to a MongoDB database
func NewFactory(domain config.Domain, providers []config.Provider, activityStreamsService *service.ActivityStreams, serverEmail *service.ServerEmail, themeService *service.Theme, templateService *service.Template, widgetService *service.Widget, contentService *service.Content, providerService *service.Provider, taskQueue *queue.Queue, attachmentOriginals afero.Fs, attachmentCache afero.Fs) (*Factory, error) {

	fmt.Println("Starting domain: " + domain.Hostname + "...")

	// Base Factory object
	factory := Factory{
		themeService:           themeService,
		templateService:        templateService,
		widgetService:          widgetService,
		contentService:         contentService,
		providerService:        providerService,
		taskQueue:              taskQueue,
		activityStreamsService: activityStreamsService,

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
	factory.blockService = service.NewBlock()
	factory.domainService = service.NewDomain()
	factory.emailService = service.NewDomainEmail(serverEmail)
	factory.encryptionKeyService = service.NewEncryptionKey()
	factory.folderService = service.NewFolder()
	factory.followerService = service.NewFollower()
	factory.followingService = service.NewFollowing()
	factory.groupService = service.NewGroup()
	factory.mentionService = service.NewMention()
	factory.inboxService = service.NewInbox()
	factory.outboxService = service.NewOutbox()
	factory.responseService = service.NewResponse()
	factory.streamService = service.NewStream()
	factory.streamDraftService = service.NewStreamDraft()
	factory.streamResponseService = service.NewStreamResponse()
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
			return derp.Report(derp.Wrap(err, "domain.factory.UpdateConfig", "Error connecting to MongoDB (Server)", domain))
		}

		// Establish a connection
		session, err := server.Session(context.Background())

		if err != nil {
			return derp.Report(derp.Wrap(err, "domain.factory.UpdateConfig", "Error connecting to MongoDB (Session)", domain))
		}

		// If we already have a database connection, then close it
		if factory.Session != nil {
			factory.Session.Close()
		}

		// Save the new session into the factory.
		factory.Session = session

		// Refresh cached services
		// TODO: CRITICAL: There are potential circular references here.  Perhaps move all init parameters to the Refresh methods??

		// Populate Attachment Service
		factory.attachmentService.Refresh(
			factory.collection(CollectionAttachment),
			factory.MediaServer(),
		)

		// Populate the Block Service
		factory.blockService.Refresh(
			factory.collection(CollectionBlock),
			factory.Follower(),
			factory.User(),
			factory.Queue(),
		)

		// Populate Domain Service
		factory.domainService.Refresh(
			factory.collection(CollectionDomain),
			domain,
			factory.Theme(),
			factory.User(),
			factory.Provider(),
			render.FuncMap(factory.Icons()),
		)

		// Populate Email Service
		factory.emailService.Refresh(
			factory.config,
		)

		// Populate EncryptionKey Service
		factory.encryptionKeyService.Refresh(
			factory.collection(CollectionEncryptionKey),
			factory.Host(),
		)

		// Populate Folder Service
		factory.folderService.Refresh(
			factory.collection(CollectionFolder),
			factory.Inbox(),
		)

		// Populate Follower Service
		factory.followerService.Refresh(
			factory.collection(CollectionFollower),
			factory.User(),
			factory.Block(),
			factory.Queue(),
			factory.Host(),
		)

		// Populate Following Service
		factory.followingService.Refresh(
			factory.collection(CollectionFollowing),
			factory.Stream(),
			factory.User(),
			factory.Inbox(),
			factory.EncryptionKey(),
			factory.ActivityStreams(),
			factory.Host(),
		)

		// Populate Group Service
		factory.groupService.Refresh(
			factory.collection(CollectionGroup),
		)

		// Populate Inbox Service
		factory.inboxService.Refresh(
			factory.collection(CollectionInbox),
			factory.Block(),
			factory.StreamResponse(),
			factory.Host(),
		)

		// Populate Mention Service
		factory.mentionService.Refresh(
			factory.collection(CollectionMention),
			factory.Block(),
			factory.Host(),
		)

		// Populate Outbox Service
		factory.outboxService.Refresh(
			factory.collection(CollectionOutbox),
			factory.Stream(),
			factory.Follower(),
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
			factory.Inbox(),
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
			factory.Host(),
			factory.StreamUpdateChannel(),
		)

		// Populate StreamDraft Service
		factory.streamDraftService.Refresh(
			factory.collection(CollectionStreamDraft),
			factory.Stream(),
		)

		// Populate the StreamResponse Service
		factory.streamResponseService.Refresh(
			factory.collection(CollectionStreamResponse),
			factory.collection(CollectionStream),
			factory.Block(),
			factory.Host(),
		)

		// Populate User Service
		factory.userService.Refresh(
			factory.collection(CollectionUser),
			factory.collection(CollectionFollower),
			factory.collection(CollectionFollowing),
			factory.collection(CollectionBlock),
			factory.Stream(),
			factory.EncryptionKey(),
			factory.Email(),
			factory.Folder(),
			factory.Host(),
		)

		// Watch for updates to streams
		if session, ok := factory.Session.(*mongodb.Session); ok {
			if collection, ok := session.Collection("Stream").(*mongodb.Collection); ok {
				go service.WatchStreams(collection.Mongo(), factory.streamUpdateChannel)
			}
		}
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

	close(factory.streamUpdateChannel)

	factory.domainService.Close()
	factory.realtimeBroker.Close()
	factory.streamService.Close()
	factory.followingService.Close()
	factory.followerService.Close()
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

func (factory *Factory) ActivityStreams() *service.ActivityStreams {
	return factory.activityStreamsService
}

// Attachment returns a fully populated Attachment service
func (factory *Factory) Attachment() *service.Attachment {
	return &factory.attachmentService
}

// Block returns a fully populated Block service
func (factory *Factory) Block() *service.Block {
	return &factory.blockService
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

// Outbox returns a fully populated Outbox service
func (factory *Factory) Outbox() *service.Outbox {
	return &factory.outboxService
}

// StreamResponse returns a fully populated StreamResponse service
func (factory *Factory) StreamResponse() *service.StreamResponse {
	return &factory.streamResponseService
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
func (factory *Factory) JWT() service.JWT {
	return service.JWT{}
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
func (factory *Factory) Queue() *queue.Queue {
	return factory.taskQueue
}

// Steranko returns a fully populated Steranko adapter for the User service.
func (factory *Factory) Steranko() *steranko.Steranko {

	return steranko.New(
		service.NewSterankoUserService(factory.User(), factory.Email()),
		factory.JWT(),
		steranko.Config{
			PasswordSchema: schema.Schema{Element: schema.String{}},
		},
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
