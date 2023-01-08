package gofed

import (
	"context"

	"github.com/go-fed/activity/streams/vocab"
)

// This method accepts a modified vocab.ActivityStreamsOrderedCollectionPage that
// had been returned by GetInbox. Right now the library only prepends new items to
// the orderedItems property, so simple diffing can be done. This method should
// then modify the actual underlying inbox to reflect the change in this page.
func (db *Database) SetInbox(c context.Context, inbox vocab.ActivityStreamsOrderedCollectionPage) error {
	return nil
}
