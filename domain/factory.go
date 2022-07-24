package domain

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	formlib "github.com/benpate/form/vocabulary"
	"github.com/benpate/mediaserver"
	"github.com/benpate/nebula"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"github.com/spf13/afero"

	"github.com/stripe/stripe-go/v72/client"
)

// Factory knows how to create an populate all services
type Factory struct {
	Session data.Session
	config  config.Domain

	// services (from server)
	layoutService   *service.Layout
	templateService *service.Template
	contentLibrary  *nebula.Library
	formLibrary     form.Library // TODO: this should be cached in the server factory after OptionCodes refactor.

	// Upload Directories (from server)
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs

	// services (within this domain/factory)
	domainService       *service.Domain
	streamService       service.Stream
	subscriptionService *service.Subscription
	realtimeBroker      *RealtimeBroker
	userService         service.User

	// real-time watchers
	streamUpdateChannel chan model.Stream

	MarkForDeletion bool
}

// NewFactory creates a new factory tied to a MongoDB database
func NewFactory(domain config.Domain, layoutService *service.Layout, templateService *service.Template, contentLibrary *nebula.Library, attachmentOriginals afero.Fs, attachmentCache afero.Fs) (*Factory, error) {

	fmt.Println("Starting domain: " + domain.Hostname + "...")

	// Base Factory object
	factory := Factory{
		layoutService:   layoutService,
		templateService: templateService,
		contentLibrary:  contentLibrary,

		attachmentOriginals: attachmentOriginals,
		attachmentCache:     attachmentCache,
		streamUpdateChannel: make(chan model.Stream),
	}

	factory.realtimeBroker = NewRealtimeBroker(&factory, factory.StreamUpdateChannel())

	// Create form library
	factory.formLibrary = form.NewLibrary(factory.OptionProvider())
	formlib.All(&factory.formLibrary)

	// Start the Domain Service
	factory.domainService = service.NewDomain(
		factory.collection(CollectionDomain),
		render.FuncMap(),
	)

	// Start the Stream Service
	factory.streamService = service.NewStream(
		factory.collection(CollectionStream),
		factory.Template(),
		factory.StreamDraft(),
		factory.Attachment(),
		factory.FormLibrary(),
		factory.ContentLibrary(),
		factory.StreamUpdateChannel(),
	)

	// Start the User Service
	factory.userService = service.NewUser(
		factory.collection(CollectionUser),
		factory.Stream(),
	)

	// Start the Subscription Service
	factory.subscriptionService = service.NewSubscription(
		factory.collection(CollectionSubscription),
		factory.Stream(),
		factory.ContentLibrary(),
	)

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

		// If we already have a database connection, then close it
		if factory.Session != nil {
			factory.Session.Close()
		}

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

		factory.Session = session

		// Refresh cached services
		factory.domainService.Refresh(factory.collection(CollectionDomain))
		factory.realtimeBroker.Refresh()
		factory.streamService.Refresh(factory.collection(CollectionStream))
		factory.subscriptionService.Refresh(factory.collection(CollectionSubscription))
		factory.userService.Refresh(factory.collection(CollectionUser))

		// Watch for updates to streams
		if session, ok := factory.Session.(*mongodb.Session); ok {
			if collection, ok := session.Collection("Stream").(*mongodb.Collection); ok {
				go service.WatchStreams(collection.Mongo(), factory.streamUpdateChannel)
			}
		}

	}

	factory.config = domain
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
	factory.subscriptionService.Close()
	factory.userService.Close()
}

/*******************************************
 * DOMAIN DATA ACCESSORS
 *******************************************/

// ID implements the set.Set interface.  (Domains are indexed by their hostname)
func (factory *Factory) ID() string {
	return factory.config.Hostname
}

func (factory *Factory) Host() string {

	if factory.config.Hostname == "localhost" {
		return "http://localhost"
	}

	return "https://" + factory.config.Hostname
}

func (factory *Factory) Hostname() string {
	return factory.config.Hostname
}

func (factory *Factory) Config() config.Domain {
	return factory.config
}

/*******************************************
 * DOMAIN MODEL SERVICES
 *******************************************/

// Attachment returns a fully populated Attachment service
func (factory *Factory) Attachment() *service.Attachment {
	result := service.NewAttachment(factory.collection(CollectionAttachment), factory.MediaServer())
	return &result
}

// Domain returns a fully populated Domain service
func (factory *Factory) Domain() *service.Domain {
	return factory.domainService
}

// Mention returns a fully populated Mention service
func (factory *Factory) Mention() service.Mention {
	return service.NewMention(factory.collection(CollectionMention))
}

// Stream returns a fully populated Stream service
func (factory *Factory) Stream() *service.Stream {
	return &factory.streamService
}

// StreamDraft returns a fully populated StreamDraft service.
func (factory *Factory) StreamDraft() *service.StreamDraft {

	result := service.NewStreamDraft(
		factory.collection(CollectionStreamDraft),
		factory.Stream(),
		factory.ContentLibrary(),
	)

	return &result
}

// StreamSource returns a fully populated StreamSource service
func (factory *Factory) StreamSource() *service.StreamSource {
	return service.NewStreamSource(factory.collection(CollectionStreamSource))
}

// Subscription returns a fully populated Subscription service
func (factory *Factory) Subscription() *service.Subscription {
	return factory.subscriptionService
}

// User returns a fully populated User service
func (factory *Factory) User() *service.User {
	return &factory.userService
}

// Group returns a fully populated Group service
func (factory *Factory) Group() *service.Group {
	result := service.NewGroup(factory.collection(CollectionGroup))
	return &result
}

/*******************************************
 * RENDER OBJECTS
 *******************************************/

// Layout service manages global website layouts (managed globally by the server.Factory)
func (factory *Factory) Layout() *service.Layout {
	return factory.layoutService
}

// Template returns a fully populated Template service (managed globally by the server.Factory)
func (factory *Factory) Template() *service.Template {
	return factory.templateService
}

// Content returns a content.Widget that can view content
func (factory *Factory) ContentLibrary() *nebula.Library {
	return factory.contentLibrary
}

/*******************************************
 * REAL-TIME UPDATE CHANNELS
 *******************************************/

// RealtimeBroker returns a new RealtimeBroker that can push stream updates to connected clients.
func (factory *Factory) RealtimeBroker() *RealtimeBroker {
	return factory.realtimeBroker
}

// StreamUpdateChannel initializes a background watcher and returns a channel containing any streams that have changed.
func (factory *Factory) StreamUpdateChannel() chan model.Stream {
	return factory.streamUpdateChannel
}

/*******************************************
 * MEDIA SERVER
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
 * OTHER NON-MODEL SERVICES
 *******************************************/

// FormLibrary returns our custom form widget library for
// use in the form.Form package
func (factory *Factory) FormLibrary() *form.Library {
	return &factory.formLibrary
}

// Key returns an instance of the Key Manager Service (KMS)
func (factory *Factory) Key() service.Key {
	return service.Key{}
}

// Steranko returns a fully populated Steranko adapter for the User service.
func (factory *Factory) Steranko() *steranko.Steranko {

	return steranko.New(
		service.NewSterankoUserService(factory.User()),
		factory.Key(),
		steranko.Config{
			PasswordSchema: schema.Schema{Element: schema.String{}},
		},
	)
}

func (factory *Factory) OptionProvider() form.OptionProvider {
	return service.NewOptionProvider(factory.Group(), factory.User())
}

/*******************************************
 * EXTERNAL APIs
 *******************************************/

// RSS returns a fully populated RSS service
func (factory *Factory) RSS() *service.RSS {
	return service.NewRSS(factory.Stream())
}

func (factory *Factory) StripeClient() (client.API, error) {

	const location = "domain.factory.StripeClient"

	var domain model.Domain
	domainService := factory.Domain()

	// Load the domain from the database
	if err := domainService.Load(&domain); err != nil {
		return client.API{}, derp.Wrap(err, location, "Error loading domain record")
	}

	// Confirm that stripe is active
	if !domain.Connections.GetBool("stripe_isActive") {
		return client.API{}, derp.NewBadRequestError(location, "Stripe is not active")
	}

	// Validate the stripe API key exists
	stripeKey := domain.Connections.GetString("stripe_apiKey")

	if stripeKey == "" {
		return client.API{}, derp.NewInternalError(location, "Stripe key must not be empty")
	}

	// Create a new client API and return
	result := client.API{}
	result.Init(stripeKey, nil)

	return result, nil
}

// Other libraries to make it her, eventually...
// ActivityPub
// Service APIs (like Twitter? Slack? Discord?, The FB?)

/*******************************************
 * HELPER UTILITIES
 *******************************************/

// collection returns a data.Collection that matches the requested name.
func (factory *Factory) collection(name string) data.Collection {
	if factory.Session == nil {
		return nil
	}
	return factory.Session.Collection(name)
}
