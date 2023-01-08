package gofed

import (
	"context"
	"net/url"
)

// ActorForOutbox returns the associated actorIRI which is the Actor's id for the provided outbox IRI.
//
// This will only be called with outboxIRI whose actors are owned by this instance.
func (db *Database) ActorForOutbox(c context.Context, outboxIRI *url.URL) (actorIRI *url.URL, err error) {
	return nil, nil
}
