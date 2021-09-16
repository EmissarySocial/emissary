package mockdb

import (
	"context"

	"github.com/benpate/data"
)

// Server is a mock database
type Server map[string][]data.Object

// New returns a fully initialized Database object
func New() data.Server {
	return Server{}
}

// Session returns a session that can be used as a mock database.
func (server Server) Session(ctx context.Context) (data.Session, error) {

	return Session{
		Server:  &server,
		Context: ctx,
	}, nil
}

// hasCollection returns TRUE if the designated collection already exists in the Server
func (server Server) hasCollection(collection string) bool {

	_, ok := server[collection]

	return ok
}

// getCollection loads (and creates, if necessary) the named collection in this datastore
func (server Server) getCollection(collection string) []data.Object {

	if !server.hasCollection(collection) {
		server[collection] = []data.Object{}
	}

	return server[collection]
}
