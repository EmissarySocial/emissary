package service

import (
	"context"

	"github.com/benpate/data"
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
