package domain

import (
	"context"
	"fmt"
	"html/template"

	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/config"
	"github.com/benpate/ghost/content"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/ghost/vocabulary"
	"github.com/benpate/steranko"
)

// Factory knows how to create an populate all services
type Factory struct {
	Session data.Session
	domain  config.Domain

	// singletons (within this domain/factory)
	templateService *service.Template
	streamService   *service.Stream
	layoutService   *service.Layout
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

	// Template Service
	factory.Template()

	// Stream Service
	factory.Stream()

	return &factory, nil
}

///////////////////////////////////////
// Domain Data Accessors

func (factory *Factory) Hostname() string {
	return factory.domain.Hostname
}

///////////////////////////////////////
// Domain Model Services

// Attachment returns a fully populated Attachment service
func (factory *Factory) Attachment() *service.Attachment {
	return service.NewAttachment(factory.collection(CollectionAttachment))
}

// Stream returns a fully populated Stream service
func (factory *Factory) Stream() *service.Stream {

	if factory.streamService == nil {

		factory.streamService = service.NewStream(
			factory.collection(CollectionStream),
			factory.Template(),
			factory.FormLibrary(),
			factory.TemplateUpdateChannel(),
			factory.StreamUpdateChannel(),
		)
	}

	return factory.streamService
}

// StreamSource returns a fully populated StreamSource service
func (factory *Factory) StreamSource() *service.StreamSource {
	return service.NewStreamSource(factory.collection(CollectionStreamSource))
}

// Template returns a fully populated Template service
func (factory *Factory) Template() *service.Template {

	// Initialize service, if necessary
	if factory.templateService == nil {
		factory.templateService = service.NewTemplate(
			factory.domain.TemplatePaths,
			factory.Layout(),
			factory.LayoutUpdateChannel(),
			factory.TemplateUpdateChannel(),
		)
	}

	return factory.templateService
}

// User returns a fully populated User service
func (factory *Factory) User() *service.User {
	return service.NewUser(factory.collection(CollectionUser))
}

///////////////////////////////////////
// Render Library

// Layout service manages global website layouts
func (factory *Factory) Layout() *service.Layout {

	if factory.layoutService == nil {
		var err error
		factory.layoutService, err = service.NewLayout(factory.domain.LayoutPath, factory.LayoutUpdateChannel())
		derp.Report(err)
	}

	return factory.layoutService
}

// StreamRenderer generates a new stream renderer service.
func (factory *Factory) StreamRenderer(stream model.Stream, request *HTTPRequest) Renderer {
	return NewRenderer(factory.Stream(), factory.Editor(), request, stream)
}

// StreamViewer generates a new stream renderer service, pegged to a specific view.
func (factory *Factory) StreamViewer(stream model.Stream, request *HTTPRequest, viewID string) Renderer {
	renderer := NewRenderer(factory.Stream(), factory.Editor(), request, stream)
	renderer.viewID = viewID
	return renderer
}

// StreamTransitioner generates a new stream renderer service, pegged to a specific transition.
func (factory *Factory) StreamTransitioner(stream model.Stream, request *HTTPRequest, transitionID string) Renderer {
	renderer := NewRenderer(factory.Stream(), factory.Editor(), request, stream)
	renderer.transitionID = transitionID
	return renderer
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
				factory.streamUpdateChannel = service.NewStreamWatcher(collection.Mongo())
				fmt.Println("factory.StreamUpdateChannel: created mongodb stream watcher.")
			}
		}

		if factory.streamUpdateChannel == nil {
			// Fall through means failure.  Just return an "empty" channel for now
			factory.streamUpdateChannel = make(chan model.Stream)
			fmt.Println("factory.StreamUpdateChannel: created regular channel.")
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

// ContentLibrary returns our custom form widget library for
// use in the form.Form package
func (factory *Factory) ContentLibrary() content.Library {
	return content.ViewerLibrary()
}

func (factory *Factory) Editor() *service.Editor {
	return service.NewEditor()
}

// FormLibrary returns our custom form widget library for
// use in the form.Form package
func (factory *Factory) FormLibrary() form.Library {

	library := form.New(factory.OptionProvider())
	vocabulary.All(library)

	return library
}

// Key returns an instance of the Key Manager Service (KMS)
func (factory *Factory) Key() service.Key {
	return service.Key{}
}

// Steranko returns a fully populated Steranko adapter for the User service.
func (factory *Factory) Steranko() *steranko.Steranko {

	if factory.steranko == nil {

		sterankoUserService := service.NewSterankoUserService(factory.User())

		factory.steranko = steranko.New(sterankoUserService, factory.Key(), factory.domain.Steranko)
	}

	return factory.steranko
}

func (factory *Factory) OptionProvider() form.OptionProvider {
	return service.NewOptionProvider(factory.User())
}

///////////////////////////////////////
// External APIs

// RSS returns a fully populated RSS service
func (factory *Factory) RSS() *service.RSS {
	return service.NewRSS(factory.Stream())
}

// Other libraries to make it her, eventually...
// ActivityPub
// Service APIs (like Twitter? Slack? Discord?, The FB?)

///////////////////////////////////////
// Helper functions

// collection returns a data.Collection that matches the requested name.
func (factory *Factory) collection(name string) data.Collection {
	return factory.Session.Collection(name)
}
