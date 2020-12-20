package domain

import (
	"context"
	"fmt"
	"html/template"
	"net/url"

	"github.com/benpate/data"
	"github.com/benpate/data/mongodb"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/config"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/ghost/vocabulary"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
)

// Factory knows how to create an populate all services
type Factory struct {
	Session data.Session
	domain  config.Domain

	// singletons (within this domain/factory)
	templateService *service.Template
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

	// TemplateSources
	factory.Template()

	return &factory, nil
}

///////////////////////////////////////
// Domain Model Services

// Attachment returns a fully populated Attachment service
func (factory *Factory) Attachment() *service.Attachment {
	return service.NewAttachment(factory.collection(CollectionAttachment))
}

// Stream returns a fully populated Stream service
func (factory *Factory) Stream() *service.Stream {
	return service.NewStream(
		factory.Template(),
		factory.collection(CollectionStream),
		factory.StreamUpdateChannel(),
	)
}

// StreamSource returns a fully populated StreamSource service
func (factory *Factory) StreamSource() *service.StreamSource {
	return service.NewStreamSource(factory.collection(CollectionStreamSource))
}

// Template returns a fully populated Template service
func (factory *Factory) Template() *service.Template {

	spew.Dump("factory.Template() -----------")
	spew.Dump(factory.templateService == nil)

	// Initialize service, if necessary
	if factory.templateService == nil {
		factory.templateService = service.NewTemplate(
			factory.domain.TemplatePaths,
			factory.Layout(),
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
func (factory *Factory) StreamRenderer(stream *model.Stream, query url.Values) *Renderer {

	return &Renderer{
		factory: factory,
		stream:  stream,
		query:   query,
	}
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

		sterankoUserService := service.NewSterankoUserService(factory.User())

		factory.steranko = steranko.New(sterankoUserService, factory.domain.Steranko)
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
