package gofed

import (
	"context"
	"net/url"
)

// InboxContains return s TRUE if the ActivityStreams object with the id is contained
// within that Inbox's OrderedCollection.
//
// A naive implementation may just do a linear search through the OrderedCollection,
// while certain databases may permit better lookup performance with a proper query.
func (db Database) InboxContains(c context.Context, inbox, id *url.URL) (contains bool, err error) {
	return false, nil
}
