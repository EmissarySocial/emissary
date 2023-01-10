package gofed

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// Owns returns TRUE when the id is an IRI owned by this running instance of the server.
// That is, the data represented by the id did not come from a federated peer.
func (db Database) Owns(c context.Context, id *url.URL) (owns bool, err error) {

	// Eliminate external URLs
	if !IsLocalURL(db.hostname, id) {
		return false, nil
	}

	// Parse the URL
	_, location, _, err := ParsePath(id)

	if err != nil {
		return false, derp.Wrap(err, "gofed.Database.Owns", "Error parsing URL", id)
	}

	return (location != model.ActivityPlaceInbox), nil
}
