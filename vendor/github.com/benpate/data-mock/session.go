package mockdb

import (
	"context"

	"github.com/benpate/data"
)

// Session is a mock database session
type Session struct {
	Server  *Server
	Context context.Context
}

// Collection returns a reference to a collection of records
func (session Session) Collection(collection string) data.Collection {

	return Collection{
		Server:  session.Server,
		Context: session.Context,
		Name:    collection,
	}
}

// Close cleans up any remaining data created by the mock session.
func (session Session) Close() {

}
