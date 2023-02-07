package gofed

import (
	"context"
	"net/url"

	"github.com/benpate/derp"
)

// OutboxForInbox returns the outboxIRI for the Actor who owns the provided inboxIRI
//
// This will only be called with inboxIRI whose actors are owned by this instance.
func (db Database) OutboxForInbox(c context.Context, inboxIRI *url.URL) (outboxIRI *url.URL, err error) {

	// Parse the inboxIRI
	ownerID, _, err := ParseInboxPath(inboxIRI)

	if err != nil {
		return nil, derp.Wrap(err, "gofed.Database", "Error parsing inbox IRI", inboxIRI)
	}

	// Generate the new outboxIRI
	result, _ := url.Parse(inboxIRI.String())
	result.Path = "/@" + ownerID.Hex() + "/pub/outbox"

	return result, nil
}
