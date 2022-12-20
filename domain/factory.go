package domain

import (
	"context"
	"fmt"
	"strings"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/gofed/activitypub"
	federatingdb "github.com/EmissarySocial/emissary/gofed/db"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queue"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/EmissarySocial/emissary/tools/domain"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/icon"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"github.com/go-fed/activity/pub"
	"github.com/spf13/afero"

	"github.com/stripe/stripe-go/v72/client"
)

// Factory knows how to create an populate all services
type Factory struct {
	Session   data.Session
	config    config.Domain
	providers []config.Provider

	// services (from server)
	layoutService   *service.Layout
	templateService *service.Template
	contentService  *service.Content
	providerService *service.Provider
	taskQueue       *queue.Queue

	// Upload Directories (from server)
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs

	// services (within this domain/factory)
	attachmentService  service.Attachment
	groupService       service.Group
	domainService      service.Domain
	emailService       service.DomainEmail
	folderService      service.Folder
	followerService    service.Follower
	followingService   service.Following
	inboxService       service.Inbox
	mentionService     service.Mention
	streamService      service.Stream
	streamDraftService service.StreamDraft
	realtimeBroker     RealtimeBroker
	userService        service.User

	// real-time watchers
	streamUpdateChannel chan model.Stream

	MarkForDeletion bool
}

// NewFactory creates a new factory tied to a MongoDB database
func NewFactory(domain config.Domain, providers []config.Provider, serverEmail *service.ServerEmail, layoutService *service.Layout, templateService *service.Template, contentService *service.Content, providerService *service.Provider, taskQueue *queue.Queue, attachmentOriginals afero.Fs, attachmentCache afero.Fs) (*Factory, error) {

	fmt.Println("Starting domain: " + domain.Hostname + "...")

	// Base Factory object
	factory := Factory{
		layoutService:   layoutService,
		templateService: templateService,
		contentService:  contentService,
		providerService: providerService,
		taskQueue:       taskQueue,

		attachmentOriginals: attachmentOriginals,
		attachmentCache:     attachmentCache,
		streamUpdateChannel: make(chan model.Stream),
	}

	factory.config.Hostname = domain.Hostname

	factory.realtimeBroker = NewRealtimeBroker(&factory, factory.StreamUpdateChannel())

	factory.mentionService = service.NewMention(factory.collection(CollectionMention))

	factory.emailService = service.NewDomainEmail(serverEmail, domain)

	// Start the Group Service
	factory.groupService = service.NewGroup(
		factory.collection(CollectionGroup),
	)

	// Start the Attachment Service
	factory.attachmentService = service.NewAttachment(
		factory.collection(CollectionAttachment),
		factory.MediaServer(),
	)

	// Start the Stream Service
	factory.streamService = service.NewStream(
		factory.collection(CollectionStream),
		factory.Template(),
		factory.Attachment(),
		domain.Hostname,
		factory.StreamUpdateChannel(),
	)

	// Start the StreamDraft Service
	factory.streamDraftService = service.NewStreamDraft(
		factory.collection(CollectionStreamDraft),
		factory.Stream(),
	)

	// Start the User Service
	factory.userService = service.NewUser(
		factory.collection(CollectionUser),
		factory.collection(CollectionFollower),
		factory.collection(CollectionFollowing),
		factory.collection(CollectionBlock),
		factory.Stream(),
		factory.Email(),
		factory.Host(),
	)

	// Start the Domain Service
	factory.domainService = service.NewDomain(
		factory.collection(CollectionDomain),
		domain,
		factory.User(),
		factory.Provider(),
		render.FuncMap(factory.Icons()),
	)

	factory.inboxService = service.NewInbox(
		factory.collection(CollectionInbox),
	)

	// Start the Following Service
	factory.followingService = service.NewFollowing(
		factory.collection(CollectionFollowing),
		factory.Stream(),
		factory.User(),
		factory.Inbox(),
		factory.Host())

	factory.followerService = service.NewFollower(
		factory.collection(CollectionFollower),
		factory.User(),
		factory.Host(),
	)

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
		factory.inboxService.Refresh(factory.collection(CollectionInbox))
		factory.attachmentService.Refresh(factory.collection(CollectionAttachment))
		factory.groupService.Refresh(factory.collection(CollectionGroup))
		factory.domainService.Refresh(factory.collection(CollectionDomain), domain)
		factory.emailService.Refresh(domain)
		factory.folderService.Refresh(factory.collection(CollectionFolder))
		factory.groupService.Refresh(factory.collection(CollectionGroup))
		factory.inboxService.Refresh(factory.collection(CollectionInbox))
		factory.realtimeBroker.Refresh()
		factory.mentionService.Refresh(factory.collection(CollectionMention))
		factory.streamService.Refresh(domain.Hostname, factory.collection(CollectionStream), factory.StreamDraft()) // handles circular depencency with streamDraftService
		factory.streamDraftService.Refresh(factory.collection(CollectionStreamDraft))
		factory.followerService.Refresh(factory.collection(CollectionFollower))
		factory.followingService.Refresh(factory.collection(CollectionFollowing))

		factory.userService.Refresh(
			factory.collection(CollectionUser),
			factory.collection(CollectionFollower),
			factory.collection(CollectionFollowing),
			factory.collection(CollectionBlock),
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

/*******************************************
 * Domain Data Accessors
 *******************************************/

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

/*******************************************
 * Domain Model Services
 *******************************************/

func (factory *Factory) Model(name string) (service.ModelService, error) {

	switch strings.ToLower(name) {

	case "folder":
		return factory.Folder(), nil

	case "follower":
		return factory.Follower(), nil

	case "following":
		return factory.Following(), nil

	case "inbox":
		return factory.Inbox(), nil

	case "outbox":
		return factory.Inbox(), nil

	}

	return nil, derp.NewInternalError("domain.Factory.Model", "Unknown model", name)
}

// Attachment returns a fully populated Attachment service
func (factory *Factory) Attachment() *service.Attachment {
	return &factory.attachmentService
}

// Domain returns a fully populated Domain service
func (factory *Factory) Domain() *service.Domain {
	return &factory.domainService
}

// EncryptionKey returns a fully populated EncryptionKey service
func (factory *Factory) EncryptionKey() *service.EncryptionKey {
	service := service.NewEncryptionKey(factory.collection(CollectionEncryptionKey))
	return &service
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

// Inbox returns a fully populated Inbox service
func (factory *Factory) Inbox() *service.Inbox {
	return &factory.inboxService
}

// Mention returns a fully populated Mention service
func (factory *Factory) Mention() *service.Mention {
	result := service.NewMention(factory.collection(CollectionMention))
	return &result
}

// Stream returns a fully populated Stream service
func (factory *Factory) Stream() *service.Stream {
	return &factory.streamService
}

// StreamDraft returns a fully populated StreamDraft service.
func (factory *Factory) StreamDraft() *service.StreamDraft {
	return &factory.streamDraftService
}

// User returns a fully populated User service
func (factory *Factory) User() *service.User {
	return &factory.userService
}

// Group returns a fully populated Group service
func (factory *Factory) Group() *service.Group {
	return &factory.groupService
}

/*******************************************
 * Render Objects
 *******************************************/

// Layout service manages global website layouts (managed globally by the server.Factory)
func (factory *Factory) Layout() *service.Layout {
	return factory.layoutService
}

// Template returns a fully populated Template service (managed globally by the server.Factory)
func (factory *Factory) Template() *service.Template {
	return factory.templateService
}

/*******************************************
 * ActivityPub
 *******************************************/

func (factory *Factory) ActivityPub_Actor() pub.Actor {
	return pub.NewActor(
		factory.ActivityPub_CommonBehavior(),
		factory.ActivityPub_SocialProtocol(),
		factory.ActivityPub_FederatingProtocol(),
		factory.ActivityPub_Database(),
		factory.ActivityPub_Clock())
}

func (factory *Factory) ActivityPub_CommonBehavior() pub.CommonBehavior {
	return activitypub.NewCommonBehavior(factory.ActivityPub_Database(), factory.User(), factory.EncryptionKey(), factory.Host())
}

func (factory *Factory) ActivityPub_SocialProtocol() pub.SocialProtocol {
	return activitypub.NewSocialProtocol()
}

func (factory *Factory) ActivityPub_FederatingProtocol() pub.FederatingProtocol {
	return activitypub.NewFederatingProtocol(factory.ActivityPub_Database())
}

func (factory *Factory) ActivityPub_Database() *federatingdb.Database {
	return federatingdb.NewDatabase(factory, factory.User(), factory.Inbox(), factory.Stream(), factory.Hostname())
}

func (factory *Factory) ActivityPub_Clock() activitypub.Clock {
	return activitypub.Clock{}
}

/*******************************************
 * Real-Time Update Channels
 *******************************************/

// RealtimeBroker returns a new RealtimeBroker that can push stream updates to connected clients.
func (factory *Factory) RealtimeBroker() *RealtimeBroker {
	return &factory.realtimeBroker
}

// StreamUpdateChannel initializes a background watcher and returns a channel containing any streams that have changed.
func (factory *Factory) StreamUpdateChannel() chan model.Stream {
	return factory.streamUpdateChannel
}

/*******************************************
 * Media Server
 *******************************************/

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
		panic(err)
	}

	// Return a filesystem pointing to the new subfolder.
	return afero.NewBasePathFs(base, path)
}

/*******************************************
 * Other Non-Model Services
 *******************************************/

// Content returns the Content transformation service
func (factory *Factory) Content() *service.Content {
	return factory.contentService
}

func (factory *Factory) Email() *service.DomainEmail {
	return &factory.emailService
}

// Key returns an instance of the Key Manager Service (KMS)
func (factory *Factory) Key() service.Key {
	return service.Key{}
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

// Publisher returns the Publisher service, which contains
// all of the business rules for publishing a stream to the federated Interwebs.
func (factory *Factory) Publisher() service.Publisher {
	return service.NewPublisher(factory.Stream(), factory.Follower(), factory.User())
}

// Queue returns the Queue service, which manages background jobs
func (factory *Factory) Queue() *queue.Queue {
	return factory.taskQueue
}

// Steranko returns a fully populated Steranko adapter for the User service.
func (factory *Factory) Steranko() *steranko.Steranko {

	return steranko.New(
		service.NewSterankoUserService(factory.User(), factory.Email()),
		factory.Key(),
		steranko.Config{
			PasswordSchema: schema.Schema{Element: schema.String{}},
		},
	)
}

// LookupProvider returns a fully populated LookupProvider service
func (factory *Factory) LookupProvider() form.LookupProvider {
	return service.NewLookupProvider(factory.Group())
}

/*******************************************
 * External APIs
 *******************************************/

// OAuth returns a fully populated OAuth service
func (factory *Factory) Provider() *service.Provider {
	return factory.providerService
}

// RSS returns a fully populated RSS service
func (factory *Factory) RSS() *service.RSS {
	return service.NewRSS(factory.Stream(), factory.Host())
}

// TODO: LOW: Move this to providers.Stripe
func (factory *Factory) StripeClient() (client.API, error) {

	const location = "domain.factory.StripeClient"

	domain := model.NewDomain()
	domainService := factory.Domain()

	// Load the domain from the database
	if err := domainService.Load(&domain); err != nil {
		return client.API{}, derp.Wrap(err, location, "Error loading domain record")
	}

	stripeClient, _ := domain.Clients.Get(providers.ProviderTypeStripe)

	// Confirm that stripe is active
	if !stripeClient.Active {
		return client.API{}, derp.NewBadRequestError(location, "Stripe is not active")
	}

	// Validate the stripe API key exists
	stripeKey := stripeClient.Data.GetString(providers.Stripe_APIKey)

	if stripeKey == "" {
		return client.API{}, derp.NewInternalError(location, "Stripe key must not be empty")
	}

	// Create a new client API and return
	result := client.API{}
	result.Init(stripeKey, nil)

	return result, nil
}

// Other libraries to make it here eventually...
// Service APIs (like Twitter? Slack? Discord?, The FB?)

/*******************************************
 * Helper Utilities
 *******************************************/

// collection returns a data.Collection that matches the requested name.
func (factory *Factory) collection(name string) data.Collection {

	if factory.Session == nil {
		return nil
	}
	return factory.Session.Collection(name)
}
