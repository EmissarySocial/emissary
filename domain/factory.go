package domain

import (
	"context"
	"fmt"
	"html/template"

	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/form/vocabulary"
	"github.com/benpate/ghost/config"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/service"
	"github.com/benpate/mediaserver"
	"github.com/benpate/steranko"
	"github.com/spf13/afero"
)

// Factory knows how to create an populate all services
type Factory struct {
	Session data.Session
	domain  config.Domain

	// singletons (within this domain/factory)
	layoutService       service.Layout
	templateService     service.Template
	streamService       service.Stream
	userService         service.User
	subscriptionService *service.Subscription

	// real-time watchers
	realtimeBroker        *RealtimeBroker
	templateUpdateChannel chan string
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
		templateUpdateChannel: make(chan string),
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
		factory.RenderFunctions(),
	)

	go factory.layoutService.Watch()

	// Template Service
	factory.templateService = service.NewTemplate(
		factory.Domain(),
		factory.Layout(),
		factory.RenderFunctions(),
		factory.domain.TemplatePath,
		factory.TemplateUpdateChannel(),
	)

	go factory.templateService.Watch()

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
	)

	// Subscription Service
	factory.subscriptionService = service.NewSubscription(
		factory.collection(CollectionSubscription),
		factory.Stream(),
	)

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
	result := service.NewDomain(factory.collection(CollectionDomain), factory.RenderFunctions())
	return &result
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
	return &factory.templateService
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

// Layout service manages global website layouts
func (factory *Factory) Layout() *service.Layout {
	return &factory.layoutService
}

// RenderStep uses an Step object to create a new action
func (factory *Factory) RenderStep(stepInfo datatype.Map) (render.Step, error) {
	return render.NewStep(factory, stepInfo)
}

// RenderFunctions provides a map of generic functions (template.FuncMap) that can be used in html.Templates
func (factory *Factory) RenderFunctions() template.FuncMap {
	return render.FuncMap()
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
	return afero.NewBasePathFs(afero.NewOsFs(), "./uploads")
}

// AttachmentCache returns a reference to the Filesystem where cached/manipulated attachment files are stored.
func (factory *Factory) AttachmentCache() afero.Fs {
	return afero.NewBasePathFs(afero.NewOsFs(), "./uploads-cache")
}

/*******************************************
 * OTHER NON-MODEL SERVICES
 *******************************************/

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
	return factory.Session.Collection(name)
}
