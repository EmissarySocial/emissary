package gofed

import (
	"context"
	"net/url"
)

// Owns returns TRUE when the id is an IRI owned by this running instance of the server.
// That is, the data represented by the id did not come from a federated peer.
func (db Database) Owns(c context.Context, id *url.URL) (owns bool, err error) {
	return false, nil
}
