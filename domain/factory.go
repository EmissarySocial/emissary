package domain

import (
	"context"
	"fmt"
	"html/template"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/action"
	"github.com/benpate/ghost/config"
	mongodb "github.com/benpate/ghost/data-mongo"
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
	templateService     *service.Template
	streamService       *service.Stream
	layoutService       *service.Layout
	subscriptionService *service.Subscription
	steranko            *steranko.Steranko

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
		Session:               session,
		domain:                domain,
		templateUpdateChannel: make(chan model.Template),
		layoutUpdateChannel:   make(chan *template.Template),
	}

	// Initialize Communication Channels

	if session, ok := factory.Session.(*mongodb.Session); ok {

		if collection, ok := session.Collection("Stream").(*mongodb.Collection); ok {
			factory.streamUpdateChannel = service.NewStreamWatcher(collection.Mongo())
		}
	}

	if factory.streamUpdateChannel == nil {
		// Fall through means failure.  Just return an "empty" channel for now
		factory.streamUpdateChannel = make(chan model.Stream)
	}

	factory.realtimeBroker = NewRealtimeBroker(&factory, factory.StreamUpdateChannel())

	// Initialize Background Services

	// This loads the web page layout (real-time updates to wait until later)
	factory.layoutService = service.NewLayout(
		factory.domain.LayoutPath,
		factory.LayoutUpdateChannel(),
	)

	// Template Service
	factory.templateService = service.NewTemplate(
		factory.domain.TemplatePaths,
		factory.Layout(),
		factory.LayoutUpdateChannel(),
		factory.TemplateUpdateChannel(),
	)

	// Stream Service
	factory.streamService = service.NewStream(
		factory.collection(CollectionStream),
		factory.Template(),
		factory.FormLibrary(),
		factory.TemplateUpdateChannel(),
		factory.StreamUpdateChannel(),
	)

	// Subscription Service
	factory.subscriptionService = service.NewSubscription(
		factory.collection(CollectionSubscription),
		factory.Stream(),
	)

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
	result := service.NewAttachment(factory.collection(CollectionAttachment))
	return &result
}

// Stream returns a fully populated Stream service
func (factory *Factory) Stream() *service.Stream {
	return factory.streamService
}

func (factory *Factory) StreamDraft() *service.StreamDraft {

	result := service.NewStreamDraft(
		factory.collection(CollectionStreamDraft),
		factory.Stream(),
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

// Template returns a fully populated Template service
func (factory *Factory) Template() *service.Template {
	return factory.templateService
}

// User returns a fully populated User service
func (factory *Factory) User() service.User {
	return service.NewUser(factory.collection(CollectionUser))
}

///////////////////////////////////////
// Render Library

// Layout service manages global website layouts
func (factory *Factory) Layout() *service.Layout {
	return factory.layoutService
}

// StreamViewer generates a new stream renderer service, pegged to a specific view.
func (factory *Factory) Renderer(ctx *steranko.Context, stream model.Stream, actionID string) (Renderer, error) {

	// Try to retrieve the action from the template
	action, err := factory.getAction(stream.TemplateID, actionID)

	if err != nil {
		return Renderer{}, derp.Wrap(err, "ghost.factory.Renderer", "Can't locate action", stream)
	}

	// Create and return the new Renderer
	renderer := NewRenderer(ctx, factory.Stream(), stream, action)
	return renderer, nil
}

// getActions locates and populates the action.Action for a specific template and actionID
func (factory *Factory) getAction(templateID string, actionID string) (action.Action, error) {

	// Load the template and action from the templateService
	templateService := factory.Template()

	config, err := templateService.Action(templateID, actionID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.factory.getAction", "Invalid actionID", templateID, actionID)
	}

	// Populate the action with the data from
	switch config.Method {

	case "create-stream":
		return action.NewAction_CreateStream(config, factory.Stream()), nil

	case "create-top-stream":
		return action.NewAction_CreateTopStream(config, factory.Stream()), nil

	case "delete-stream":
		return action.NewAction_DeleteStream(config, factory.Stream()), nil

	case "publish-content":
		return action.NewAction_PublishContent(config, factory.Stream()), nil

	case "update-content":
		return action.NewAction_UpdateContent(config, factory.Stream()), nil

	case "update-data":
		return action.NewAction_UpdateData(config, factory.Template(), factory.Stream(), factory.FormLibrary()), nil

	case "update-state":
		return action.NewAction_UpdateState(config, factory.Template(), factory.Stream(), factory.FormLibrary()), nil

	case "view-stream":
		return action.NewAction_ViewStream(config, factory.Layout()), nil
	}

	// Fall through means we have an unrecognized action
	return nil, derp.New(derp.CodeInternalError, "ghost.factory.getAction", "Invalid action configuration", config)
}

///////////////////////////////////////
// Real-Time UpdateChannels

// RealtimeBroker returns a new RealtimeBroker that can push stream updates to connected clients.
func (factory *Factory) RealtimeBroker() *RealtimeBroker {
	return factory.realtimeBroker
}

// StreamUpdateChannel initializes a background watcher and returns a channel containing any streams that have changed.
func (factory *Factory) StreamUpdateChannel() chan model.Stream {
	return factory.streamUpdateChannel
}

// TemplateUpdateChannel returns a channel for transmitting templates that have changed.
func (factory *Factory) TemplateUpdateChannel() chan model.Template {
	return factory.templateUpdateChannel
}

// LayoutUpdateChannel returns a channel for transmitting the global layout when it has changed.
func (factory *Factory) LayoutUpdateChannel() chan *template.Template {
	return factory.layoutUpdateChannel
}

///////////////////////////////////////
// NON MODEL SERVICES

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

	return steranko.New(
		service.NewSterankoUserService(factory.User()),
		factory.Key(),
		factory.domain.Steranko,
	)
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
