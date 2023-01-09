package gofed

import (
	"context"
	"net/url"

	"github.com/benpate/derp"
)

// ActorForInbox returns the actorIRI that is associated with the provided inboxIRI.
//
// This will only be called with inboxIRI whose actors are owned by this instance.
func (db Database) ActorForInbox(c context.Context, inboxIRI *url.URL) (actorIRI *url.URL, err error) {

	ownerID, location, _, err := ParseURL(inboxIRI)

	if err != nil {
		return nil, derp.Wrap(err, "gofed.Database", "Error parsing outbox IRI", inboxIRI)
	}

	if location != "inbox" {
		return nil, derp.Wrap(err, "gofed.Database", "Invalid location for outbox IRI", inboxIRI)
	}

	result, _ := url.Parse(actorIRI.String())
	result.Path = "/@" + ownerID.Hex()

	return result, nil
}
