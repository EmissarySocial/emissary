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
	ownerID, location, _, err := ParseURL(inboxIRI)

	if err != nil {
		return nil, derp.Wrap(err, "gofed.Database", "Error parsing inbox IRI", inboxIRI)
	}

	// Validate basic assumptions about the URLs
	if location != "inbox" {
		return nil, derp.Wrap(err, "gofed.Database", "Invalid location for inbox IRI", inboxIRI)
	}

	// Generate the new outboxIRI
	result, _ := url.Parse(inboxIRI.String())
	result.Path = "/@" + ownerID.Hex() + "/outbox"

	return result, nil
}
