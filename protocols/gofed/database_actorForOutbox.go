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

	ownerID, _, err := ParseOutboxPath(outboxIRI)

	if err != nil {
		return nil, derp.Wrap(err, "gofed.Database", "Error parsing outbox IRI", outboxIRI)
	}

	result, _ := url.Parse(outboxIRI.String())
	result.Path = "/@" + ownerID.Hex() + "/pub"

	return result, nil
}
