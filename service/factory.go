package service

import (
	"context"
	"fmt"
	"html/template"

	"github.com/benpate/data"
	"github.com/benpate/data/mongodb"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/config"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/vocabulary"
	"github.com/benpate/steranko"
)

// Factory knows how to create an populate all services
type Factory struct {
	Session data.Session
	domain  config.Domain

	// singletons (within this domain/factory)
	templateService *Template
	layoutService   *Layout
	steranko        *steranko.Steranko

	// real-time watchers
	realtimeBroker        *RealtimeBroker
	layoutUpdateChannel   chan *template.Template
	templateUpdateChannel chan model.Template
	streamUpdateChannel   chan model.Stream
}

// NewFactory creates a new factory tied to a MongoDB database
func NewFactory(domain config.Domain) (*Factory, error) {

	fmt.Println("Starting Hostname: " + domain.Hostname)

	server, err := mongodb.New(domain.ConnectString, domain.DatabaseName)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.NewFactory", "Error connecting to MongoDB (Server)", domain)
	}

	session, err := server.Session(context.Background())

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.NewFactory", "Error connecting to MongoDB (Session)", domain)
	}

	factory := Factory{
		Session: session,
		domain:  domain,
	}

	// Initialize Background Services

	// This loads the web page layout (real-time updates to wait until later)
	factory.Layout()

	// TemplateSources
	factory.Template()

	return &factory, nil
}

///////////////////////////////////////
// Domain Model Services

// Attachment returns a fully populated Attachment service
func (factory *Factory) Attachment() Attachment {
	return Attachment{
		factory:    factory,
		collection: factory.Session.Collection(CollectionAttachment),
	}
}

// Folder returns a fully populated Folder service
func (factory *Factory) Folder() Folder {
	return Folder{
		factory:    factory,
		collection: factory.Session.Collection(CollectionFolder),
	}
}

// Key returns a fully populated Key service
func (factory *Factory) Key() Key {
	return Key{
		factory:    factory,
		collection: factory.Session.Collection(CollectionKey),
	}
}

// Stream returns a fully populated Stream service
func (factory *Factory) Stream() Stream {
	return Stream{
		factory:             factory,
		collection:          factory.Session.Collection(CollectionStream),
		streamUpdateChannel: factory.StreamUpdateChannel(),
	}
}

// StreamSource returns a fully populated StreamSource service
func (factory *Factory) StreamSource() StreamSource {
	return StreamSource{
		factory:    factory,
		collection: factory.Session.Collection(CollectionStreamSource),
	}
}

// Template returns a fully populated Template service
func (factory *Factory) Template() *Template {

	// Initialize service, if necessary
	if factory.templateService == nil {
		factory.templateService = NewTemplate(
			factory.domain.TemplatePaths,
			factory.LayoutUpdateChannel(),
			factory.TemplateUpdateChannel(),
			factory.StreamUpdateChannel(),
			factory.Layout(),
			factory.Stream(),
		)
	}

	return factory.templateService
}

// User returns a fully populated User service
func (factory *Factory) User() User {
	return User{
		factory:    factory,
		collection: factory.Session.Collection(CollectionUser),
	}
}

///////////////////////////////////////
// Render Library

// Layout service manages global website layouts
func (factory *Factory) Layout() *Layout {

	if factory.layoutService == nil {
		var err error
		factory.layoutService, err = NewLayout(factory.domain.LayoutPath, factory.LayoutUpdateChannel())
		derp.Report(err)
	}

	return factory.layoutService
}

// StreamRenderer service returns a fully populated render.Stream object
func (factory *Factory) StreamRenderer(stream model.Stream, view string) render.Stream {
	return render.NewStream(factory.Layout(), factory.Folder(), factory.Template(), factory.Stream(), stream, view)
}

// FormRenderer service returns a fully populated render.Form object
func (factory *Factory) FormRenderer(stream model.Stream, layout string, transition string) render.Form {
	return render.NewForm(factory.Layout(), factory.Folder(), factory.Template(), factory.FormLibrary(), stream, layout, transition)
}

// FolderRenderer service returns a fully populated render.Folder object
func (factory *Factory) FolderRenderer(folder model.Folder, layout string) render.Folder {
	return render.NewFolder(factory.Layout(), factory.Folder(), factory.Template(), factory.Stream(), folder, layout)
}

///////////////////////////////////////
// Real-Time UpdateChannels

// RealtimeBroker returns a new RealtimeBroker that can push stream updates to connected clients.
func (factory *Factory) RealtimeBroker() *RealtimeBroker {

	if factory.realtimeBroker == nil {
		factory.realtimeBroker = NewRealtimeBroker(factory, factory.StreamUpdateChannel())
	}

	return factory.realtimeBroker
}

// StreamUpdateChannel initializes a background watcher and returns a channel containing any streams that have changed.
func (factory *Factory) StreamUpdateChannel() chan model.Stream {

	if factory.streamUpdateChannel == nil {

		if session, ok := factory.Session.(*mongodb.Session); ok {

			if collection, ok := session.Collection("Stream").(*mongodb.Collection); ok {
				factory.streamUpdateChannel = StreamWatcher(collection.Mongo())
			}
		}

		if factory.streamUpdateChannel == nil {
			// Fall through means failure.  Just return an "empty" channel for now
			factory.streamUpdateChannel = make(chan model.Stream)
		}
	}

	return factory.streamUpdateChannel
}

// TemplateUpdateChannel returns a channel for transmitting templates that have changed.
func (factory *Factory) TemplateUpdateChannel() chan model.Template {

	if factory.templateUpdateChannel == nil {
		factory.templateUpdateChannel = make(chan model.Template)
	}

	return factory.templateUpdateChannel
}

// LayoutUpdateChannel returns a channel for transmitting the global layout when it has changed.
func (factory *Factory) LayoutUpdateChannel() chan *template.Template {

	if factory.layoutUpdateChannel == nil {
		factory.layoutUpdateChannel = make(chan *template.Template)
	}

	return factory.layoutUpdateChannel
}

///////////////////////////////////////
// NON MODEL SERVICES

// Steranko returns a fully populated Steranko adapter for the User service.
func (factory *Factory) Steranko() *steranko.Steranko {

	if factory.steranko == nil {

		userService := SterankoUserService{
			userService: factory.User(),
		}

		config := steranko.Config{}

		factory.steranko = steranko.New(userService, config)
	}

	return factory.steranko
}

// FormLibrary returns our custom form widget library for
// use in the form.Form package
func (factory *Factory) FormLibrary() form.Library {

	library := form.New()
	vocabulary.All(library)

	return library
}

///////////////////////////////////////
// External APIs

// RSS returns a fully populated RSS service
func (factory *Factory) RSS() RSS {
	return RSS{
		factory: factory,
	}
}

// Close ends any connections opened by this Factory.
func (factory *Factory) Close() {
	// DO NOT DO THIS OR YOU WILL PERMANENTLY DISCONNECT FROM THE DATABASE
	// factory.Session.Close()
}

// Other libraries to make it her, eventually...
// ActivityPub
// Service APIs (like Twitter? Slack? Discord?, The FB?)
