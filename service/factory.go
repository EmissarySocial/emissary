package service

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/presto"
)

// Factory knows how to create an populate all services
type Factory struct {
	ds data.Datastore
	// domains map[string]data.Datastore
}

// NewFactory generates a new, fully populated Factory object that is ready to use.
func NewFactory(ds data.Datastore) *Factory {

	return &Factory{
		ds: ds,
	}
}

// Session returns a new mongodb database session
func (factory *Factory) Session() data.Session {
	return factory.ds.Session(context.TODO())
}

// Actor returns a fully populated Actor service
func (factory *Factory) Actor() Actor {
	return Actor{
		factory: factory,
		session: factory.Session(),
	}
}

// Attachment returns a fully populated Attachment service
func (factory *Factory) Attachment() Attachment {
	return Attachment{
		factory: factory,
		session: factory.Session(),
	}
}

// Comment returns a fully populated Comment service
func (factory *Factory) Comment() Comment {
	return Comment{
		factory: factory,
		session: factory.Session(),
	}
}

// Domain returns a fully populated Website service
func (factory *Factory) Domain() Domain {
	return Domain{
		factory: factory,
		session: factory.Session(),
	}
}

// Key returns a fully populated Contact service
func (factory *Factory) Key() Key {
	return Key{
		factory: factory,
		session: factory.Session(),
	}
}

// Page returns a fully populated Contact service
func (factory *Factory) Page() Page {
	return Page{
		factory: factory,
		session: factory.Session(),
	}
}

// Stream returns a fully populated Section service
func (factory *Factory) Stream() Stream {
	return Stream{
		factory: factory,
		session: factory.Session(),
	}
}

// User returns a fully populated User service
func (factory *Factory) User() User {
	return User{
		factory: factory,
		session: factory.Session(),
	}
}

// Presto is an adapter that maps service functions into presto.ServiceFunc's
func (factory *Factory) Presto(service string) presto.ServiceFunc {

	return func() presto.Service {

		switch service {

		case "Actor":
			return factory.Actor()

		case "Attachment":
			return factory.Attachment()

		case "Comment":
			return factory.Comment()

		case "Domain":
			return factory.Domain()

		case "Key":
			return factory.Key()

		case "Page":
			return factory.Page()

		case "Stream":
			return factory.Stream()

		case "User":
			return factory.User()

		}

		panic("Invalid service name: " + service)
	}
}
