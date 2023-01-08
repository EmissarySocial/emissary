package gofed

import (
	"context"
	"net/url"
)

// ActorForInbox returns the actorIRI that is associated with the provided inboxIRI.
//
// This will only be called with inboxIRI whose actors are owned by this instance.
func (db *Database) ActorForInbox(c context.Context, inboxIRI *url.URL) (actorIRI *url.URL, err error) {
	return nil, nil
}
