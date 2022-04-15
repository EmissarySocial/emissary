package domain

import (
	"context"
	"fmt"

	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	formlib "github.com/benpate/form/vocabulary"
	"github.com/benpate/nebula"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
	"github.com/spf13/afero"
	"github.com/whisperverse/mediaserver"
	"github.com/whisperverse/whisperverse/config"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/service"
)

// Factory knows how to create an populate all services
type Factory struct {
	Session data.Session
	domain  config.Domain

	// services (from server)
	layoutService   *service.Layout
	templateService *service.Template

	// services (within this domain/factory)
	streamService       service.Stream
	userService         service.User
	subscriptionService *service.Subscription

	// Widget Libraries
	formLibrary    form.Library
	contentLibrary nebula.Library

	// Upload Directories
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs

	// real-time watchers
	realtimeBroker        *RealtimeBroker
	templateUpdateChannel chan string
	streamUpdateChannel   chan model.Stream
}

// NewFactory creates a new factory tied to a MongoDB database
func NewFactory(domain config.Domain, layoutService *service.Layout, templateService *service.Template, attachmentOriginals afero.Fs, attachmentCache afero.Fs) (*Factory, error) {

	fmt.Println("Starting Hostname: " + domain.Hostname + "...")

	// Base Factory object
	factory := Factory{
		domain:                domain,
		layoutService:         layoutService,
		templateService:       templateService,
		templateUpdateChannel: make(chan string),
		attachmentOriginals:   attachmentOriginals,
		attachmentCache:       attachmentCache,
	}

	// If there is a database then set it up now.
	if domain.ConnectString != "" {

		server, err := mongodb.New(domain.ConnectString, domain.DatabaseName)

		if err != nil {
			return nil, derp.Wrap(err, "service.NewFactory", "Error connecting to MongoDB (Server)", domain)
		}

		session, err := server.Session(context.Background())

		if err != nil {
			return nil, derp.Wrap(err, "service.NewFactory", "Error connecting to MongoDB (Session)", domain)
		}

		factory.Session = session

		/** REAL TIME COMMUNICATION CHANNELS *********************/

		// Create Stream Update Channel
		if session, ok := factory.Session.(*mongodb.Session); ok {

			if collection, ok := session.Collection("Stream").(*mongodb.Collection); ok {
				factory.streamUpdateChannel = service.NewStreamWatcher(collection.Mongo())
			}
		}

		// Fall through means we're not running on MongoDB.  Just return an "empty" channel for now
		if factory.streamUpdateChannel == nil {
			factory.streamUpdateChannel = make(chan model.Stream)
		}

		// Create Realtime Broker
		factory.realtimeBroker = NewRealtimeBroker(&factory, factory.StreamUpdateChannel())

		/** SINGLETON SERVICES *********************/

		// Stream Service
		factory.streamService = service.NewStream(
			factory.collection(CollectionStream),
			factory.Template(),
			factory.StreamDraft(),
			factory.Attachment(),
			factory.FormLibrary(),
			factory.TemplateUpdateChannel(),
			factory.StreamUpdateChannel(),
		)

		go factory.streamService.Watch()

		factory.userService = service.NewUser(
			factory.collection(CollectionUser),
			factory.Stream(),
		)

		// Subscription Service
		factory.subscriptionService = service.NewSubscription(
			factory.collection(CollectionSubscription),
			factory.Stream(),
			factory.ContentLibrary(),
		)
	}

	/** WIDGET LIBRARIES *********************/

	// Crate content library (for now, only using defaults)
	factory.contentLibrary = nebula.NewLibrary()

	// Create form library
	factory.formLibrary = form.NewLibrary(factory.OptionProvider())
	formlib.All(&factory.formLibrary)

	return &factory, nil
}

/*******************************************
 * DOMAIN DATA ACCESSORS
 *******************************************/

func (factory *Factory) Hostname() string {
	return factory.domain.Hostname
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
	result := service.NewDomain(factory.collection(CollectionDomain), render.FuncMap())
	return &result
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
	return &factory.contentLibrary
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

// TemplateUpdateChannel returns a channel for transmitting templates that have changed.
func (factory *Factory) TemplateUpdateChannel() chan string {
	return factory.templateUpdateChannel
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

func (factory *Factory) Outbox() service.Outbox {
	return service.NewOutbox()
}

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
