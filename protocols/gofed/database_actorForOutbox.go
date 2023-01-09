package gofed

import (
	"context"
	"net/url"

	"github.com/benpate/derp"
)

// ActorForOutbox returns the associated actorIRI which is the Actor's id for the provided outbox IRI.
//
// This will only be called with outboxIRI whose actors are owned by this instance.
func (db Database) ActorForOutbox(c context.Context, outboxIRI *url.URL) (actorIRI *url.URL, err error) {

	ownerID, location, _, err := ParseURL(outboxIRI)

	if err != nil {
		return nil, derp.Wrap(err, "gofed.Database", "Error parsing outbox IRI", outboxIRI)
	}

	if location != "outbox" {
		return nil, derp.Wrap(err, "gofed.Database", "Invalid location for outbox IRI", outboxIRI)
	}

	result, _ := url.Parse(actorIRI.String())
	result.Path = "/@" + ownerID.Hex()

	return result, nil
}
