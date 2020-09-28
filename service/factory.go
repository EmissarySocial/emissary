package service

import (
	"context"
	"fmt"

	"github.com/benpate/data"
	"github.com/benpate/data/mongodb"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/config"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/service/templatesource"
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
	templateWatcher chan model.Template
	realtimeBroker  *RealtimeBroker
	steranko        *steranko.Steranko
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
	fmt.Println(" - Website Layout.")
	factory.Layout()

	// TemplateSources
	templateService := factory.Template()

	for _, path := range domain.TemplatePaths {
		fmt.Println(" - Template Directory: " + path)
		fileSource := templatesource.NewFile(path)
		templateService.AddSource(fileSource)
	}

	return &factory, nil
}

// Attachment returns a fully populated Attachment service
func (factory *Factory) Attachment() Attachment {
	return Attachment{
		factory:    factory,
		collection: factory.Session.Collection(CollectionAttachment),
	}
}

// Key returns a fully populated Contact service
func (factory *Factory) Key() Key {
	return Key{
		factory:    factory,
		collection: factory.Session.Collection(CollectionKey),
	}
}

// StreamSource returns a fully populated StreamSource service
func (factory *Factory) StreamSource() StreamSource {
	return StreamSource{
		factory:    factory,
		collection: factory.Session.Collection(CollectionStreamSource),
	}
}

// Stream returns a fully populated Stream service
func (factory *Factory) Stream() Stream {
	return Stream{
		factory:    factory,
		collection: factory.Session.Collection(CollectionStream),
	}
}

// Template returns a fully populated Template service
func (factory *Factory) Template() *Template {

	// Initialize service, if necessary
	if factory.templateService == nil {

		factory.templateService = &Template{
			Factory:   factory,
			Sources:   make([]TemplateSource, 0),
			Templates: make(map[string]*model.Template),
			Updates:   factory.TemplateWatcher(),
		}

		go factory.templateService.Start()
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
// WATCHERS

// Layout service manages layouts
func (factory *Factory) Layout() *Layout {

	if factory.layoutService == nil {
		layout, err := NewLayout(factory.domain.LayoutPath)
		factory.layoutService = &layout
		derp.Report(err)
	}

	return factory.layoutService
}

// StreamWatcher initializes a background watcher and returns a channel containing any streams that have changed.
func (factory *Factory) StreamWatcher() chan model.Stream {

	if session, ok := factory.Session.(*mongodb.Session); ok {

		if collection, ok := session.Collection("Stream").(*mongodb.Collection); ok {
			return StreamWatcher(collection.Mongo())
		}
	}

	// Fall through means failure.  Just return an "empty" channel for now
	return make(chan model.Stream)
}

// TemplateWatcher returns a channel for transmitting templates that have changed.
func (factory *Factory) TemplateWatcher() chan model.Template {

	if factory.templateWatcher == nil {
		factory.templateWatcher = make(chan model.Template)
	}

	return factory.templateWatcher
}

// RealtimeBroker returns a new RealtimeBroker that can push stream updates to connected clients.
func (factory *Factory) RealtimeBroker() *RealtimeBroker {

	if factory.realtimeBroker == nil {
		factory.realtimeBroker = NewRealtimeBroker(factory)
	}

	return factory.realtimeBroker
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

func (factory *Factory) FormLibrary() form.Library {

	library := form.New()
	vocabulary.All(library)

	return library
}

/////////////////////// Render Library

// StreamRenderer returns a fully populated render.Stream object
func (factory *Factory) StreamRenderer(stream *model.Stream, wrapper string, view string) render.Stream {

	layout := factory.Layout()
	return render.NewStream(layout.Template, factory.Template(), factory.Stream(), stream, wrapper, view)
}

// FormRenderer returns a fully populated render.Form object
func (factory *Factory) FormRenderer(stream *model.Stream, wrapper string, transition string) render.Form {

	layout := factory.Layout()
	return render.NewForm(layout.Template, factory.Template(), factory.FormLibrary(), stream, wrapper, transition)
}

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
