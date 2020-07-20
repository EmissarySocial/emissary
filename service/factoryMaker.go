package service

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/presto"
)

// FactoryMaker stores basic database connection information, and is able to
// make new factories for each user request.
type FactoryMaker struct {
	Server data.Server
}

// NewFactoryMaker returns a fully populated FactoryMaker object
func NewFactoryMaker(server data.Server) FactoryMaker {
	return FactoryMaker{
		Server: server,
	}
}

// Factory makes a new Factory object that is fully initialied (with a Context)
// and ready to generate new service objects.
func (fm FactoryMaker) Factory(ctx context.Context) Factory {
	return Factory{
		Context: ctx,
		Session: fm.Server.Session(ctx),
	}
}

// Actor returns a presto.Service for managing Actors
func (fm FactoryMaker) Actor(ctx context.Context) presto.Service {
	return fm.Factory(ctx).Actor()
}

// Attachment returns a presto.Service for managing Attachments
func (fm FactoryMaker) Attachment(ctx context.Context) presto.Service {
	return fm.Factory(ctx).Attachment()
}

// Comment returns a presto.Service for managing Comments
func (fm FactoryMaker) Comment(ctx context.Context) presto.Service {
	return fm.Factory(ctx).Comment()
}

// Domain returns a presto.Service for managing Domains
func (fm FactoryMaker) Domain(ctx context.Context) presto.Service {
	return fm.Factory(ctx).Domain()
}

// Key returns a presto.Service for managing Keys
func (fm FactoryMaker) Key(ctx context.Context) presto.Service {
	return fm.Factory(ctx).Key()
}

// Stream returns a presto.Service for managing Streams
func (fm FactoryMaker) Stream(ctx context.Context) presto.Service {
	return fm.Factory(ctx).Stream()
}

// User returns a presto.Service for managing Users
func (fm FactoryMaker) User(ctx context.Context) presto.Service {
	return fm.Factory(ctx).User()
}
