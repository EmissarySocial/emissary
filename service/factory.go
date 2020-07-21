package service

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service/templatesource"
)

// Factory knows how to create an populate all services
type Factory struct {
	Context context.Context
	Session data.Session
}

// Actor returns a fully populated Actor service
func (factory Factory) Actor() Actor {
	return Actor{
		factory: factory,
		session: factory.Session,
	}
}

// Attachment returns a fully populated Attachment service
func (factory Factory) Attachment() Attachment {
	return Attachment{
		factory: factory,
		session: factory.Session,
	}
}

// Comment returns a fully populated Comment service
func (factory Factory) Comment() Comment {
	return Comment{
		factory: factory,
		session: factory.Session,
	}
}

// Domain returns a fully populated Website service
func (factory Factory) Domain() Domain {
	return Domain{
		factory: factory,
		session: factory.Session,
	}
}

// Key returns a fully populated Contact service
func (factory Factory) Key() Key {
	return Key{
		factory: factory,
		session: factory.Session,
	}
}

// Publisher returns a fully populated Publisher service
func (factory Factory) Publisher() Publisher {
	return Publisher{
		factory: factory,
		session: factory.Session,
	}
}

// StreamSource returns a fully populated StreamSource service
func (factory Factory) StreamSource() StreamSource {
	return StreamSource{
		factory: factory,
		session: factory.Session,
	}
}

// Stream returns a fully populated Stream service
func (factory Factory) Stream() Stream {
	return Stream{
		factory: factory,
		session: factory.Session,
	}
}

// Template returns a fully populated Template service
func (factory Factory) Template() *Template {

	// Initialize service, if necessary
	if singletonTemplateService == nil {
		singletonTemplateService = &Template{
			Sources:   []templatesource.TemplateSource{},
			Templates: map[string]model.Template{},
		}
	}

	return singletonTemplateService
}

// User returns a fully populated User service
func (factory Factory) User() User {
	return User{
		factory: factory,
		session: factory.Session,
	}
}

/// NON MODEL SERVICES

// RSS returns a fully populated RSS service
func (factory Factory) RSS() RSS {
	return RSS{
		factory: factory,
	}
}
