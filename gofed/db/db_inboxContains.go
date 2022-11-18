package db

import (
	"context"
	"net/url"
	"strings"

	"github.com/benpate/derp"
)

// Given the IRI of an inbox, the implemented method should set contains to true if the ActivityStreams object with the id
// is contained within that Inbox's OrderedCollection.
//
// A naive implementation may just do a linear search through the OrderedCollection, while certain databases may permit
// better lookup performance with a proper query.

func (db *Database) InboxContains(_ context.Context, inboxURL *url.URL, itemURL *url.URL) (contains bool, err error) {

	// Guarantee that the item's URL is contained within the inbox URL.
	if !strings.HasPrefix(itemURL.String(), inboxURL.String()) {
		return false, derp.NewBadRequestError("activitypub.Database.InboxContains", "Item URL does not match inbox URL", itemURL.String(), inboxURL.String())
	}

	return db.Exists(nil, itemURL)
}
