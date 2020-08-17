package service

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/form"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/vocabulary"
	"github.com/spf13/viper"
)

// Factory knows how to create an populate all services
type Factory struct {
	Context context.Context
	Session data.Session
}

// Actor returns a fully populated Actor service
func (factory Factory) Actor() Actor {
	return Actor{
		factory:    factory,
		collection: factory.Session.Collection(CollectionActor),
	}
}

// Attachment returns a fully populated Attachment service
func (factory Factory) Attachment() Attachment {
	return Attachment{
		factory:    factory,
		collection: factory.Session.Collection(CollectionAttachment),
	}
}

// Comment returns a fully populated Comment service
func (factory Factory) Comment() Comment {
	return Comment{
		factory:    factory,
		collection: factory.Session.Collection(CollectionComment),
	}
}

// Domain returns a fully populated Website service
func (factory Factory) Domain() Domain {
	return Domain{
		factory:    factory,
		collection: factory.Session.Collection(CollectionDomain),
	}
}

// Key returns a fully populated Contact service
func (factory Factory) Key() Key {
	return Key{
		factory:    factory,
		collection: factory.Session.Collection(CollectionKey),
	}
}

// StreamSource returns a fully populated StreamSource service
func (factory Factory) StreamSource() StreamSource {
	return StreamSource{
		factory:    factory,
		collection: factory.Session.Collection(CollectionStreamSource),
	}
}

// Stream returns a fully populated Stream service
func (factory Factory) Stream() Stream {
	return Stream{
		factory:    factory,
		collection: factory.Session.Collection(CollectionStream),
	}
}

// Template returns a fully populated Template service
func (factory Factory) Template() *Template {

	// Initialize service, if necessary
	if singletonTemplateService == nil {
		singletonTemplateService = &Template{
			Factory:   &factory,
			Sources:   make([]TemplateSource, 0),
			Templates: make(map[string]*model.Template),
			Updates:   make(chan *model.Template),
		}

		go singletonTemplateService.Start()
	}

	return singletonTemplateService
}

// User returns a fully populated User service
func (factory Factory) User() User {
	return User{
		factory:    factory,
		collection: factory.Session.Collection(CollectionUser),
	}
}

///////////////////////////////////////
// WATCHERS

func (factory Factory) StreamWatcher() chan model.Stream {
	return StreamWatcher(viper.GetString("dbserver"), viper.GetString("dbname"))
}

func (factory Factory) RealtimeBroker() *RealtimeBroker {

	if singletonRealtimeBroker == nil {
		singletonRealtimeBroker = NewRealtimeBroker(factory)
	}

	return singletonRealtimeBroker
}

/// NON MODEL SERVICES

func (factory Factory) Render() *Render {
	return &Render{
		factory: factory,
	}
}

func (factory Factory) PageService() *PageService {
	return &PageService{}
}

func (factory Factory) FormService() *FormService {
	return &FormService{}
}

func (factory Factory) FormLibrary() form.Library {

	library := form.New()
	vocabulary.All(library)

	return library
}

// RSS returns a fully populated RSS service
func (factory Factory) RSS() RSS {
	return RSS{
		factory: factory,
	}
}

// Close ends any connections opened by this Factory.
func (factory Factory) Close() {
	// DO NOT DO THIS OR YOU WILL PERMANENTLY DISCONNECT FROM THE DATABASE
	// factory.Session.Close()
}
