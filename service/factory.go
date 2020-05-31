package service

import (
	"context"

	"github.com/benpate/data"
)

/// SINGLETON VALUES
var templateCache *TemplateCache

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

// Post returns a fully populated Contact service
func (factory Factory) Post() Post {
	return Post{
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

// Source returns a fully populated Source service
func (factory Factory) Source() Source {
	return Source{
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
func (factory Factory) Template() Template {
	return Template{
		factory: factory,
		session: factory.Session,
	}
}

// User returns a fully populated User service
func (factory Factory) User() User {
	return User{
		factory: factory,
		session: factory.Session,
	}
}

/// NON MODEL SERVICES

// TemplateCache returns a fully populated TemplateCache service
func (factory Factory) TemplateCache() *TemplateCache {

	// Initialize
	if templateCache == nil {
		templateCache, _ = NewTemplateCache(factory)
		// TODO: Should USE the errors that NewTemplateCache returns, eventually.
	}

	return templateCache
}

// RSS returns a fully populated RSS service
func (factory Factory) RSS() RSS {
	return RSS{
		factory: factory,
	}
}
