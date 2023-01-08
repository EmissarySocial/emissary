package db

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/protocols/gofed/activityStreams"
	"github.com/go-fed/activity/streams/vocab"
)

/*
This method returns the latest page of the inbox corresponding to the outboxIRI.

It is similar in behavior to its GetInbox counterpart, but for the actor's Outbox instead.

See the similar documentation for GetInbox.
*/

func (db *Database) GetOutbox(ctx context.Context, outboxURL *url.URL) (outbox vocab.ActivityStreamsOrderedCollectionPage, err error) {
	return db.getOrderedCollectionPage(ctx, outboxURL, activityStreams.ItemTypeOutbox)
}
