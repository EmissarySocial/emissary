package gofed

import (
	"context"
	"net/url"
)

// OutboxForInbox returns the outboxIRI for the Actor who owns the provided inboxIRI
//
// This will only be called with inboxIRI whose actors are owned by this instance.
func (db Database) OutboxForInbox(c context.Context, inboxIRI *url.URL) (outboxIRI *url.URL, err error) {
	return nil, nil
}
