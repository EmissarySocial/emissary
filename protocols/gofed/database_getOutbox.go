package gofed

import (
	"context"
	"net/url"

	"github.com/go-fed/activity/streams/vocab"
)

// GetOutbox returns the latest page of the inbox corresponding to the outboxIRI.
//
// It is similar in behavior to its GetInbox counterpart, but for the actor's Outbox
// instead. See the similar documentation for GetInbox.
func (db *Database) GetOutbox(c context.Context, outboxIRI *url.URL) (inbox vocab.ActivityStreamsOrderedCollectionPage, err error) {
	return nil, nil
}
