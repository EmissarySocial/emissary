package gofed

import (
	"context"

	"github.com/go-fed/activity/streams/vocab"
)

// SetOutbox accepts a modified vocab.ActivityStreamsOrderedCollectionPage that had
// been returned by GetOutbox to update the underlying outbox.
//
// It is similar in behavior to its SetInbox counterpart, but for the actor's
// Outbox instead.  See the similar documentation for SetInbox.
func (db *Database) SetOutbox(c context.Context, outbox vocab.ActivityStreamsOrderedCollectionPage) error {
	return nil
}
