package db

import (
	"context"

	"github.com/EmissarySocial/emissary/protocols/gofed/activityStreams"
	"github.com/EmissarySocial/emissary/protocols/gofed/common"
	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams/vocab"
)

/*
This method accepts a modified vocab.ActivityStreamsOrderedCollectionPage that had been returned by GetInbox.
Right now the library only prepends new items to the orderedItems property, so simple diffing can be done.
This method should then modify the actual underlying inbox to reflect the change in this page.
*/

func (db *Database) SetInbox(ctx context.Context, inbox vocab.ActivityStreamsOrderedCollectionPage) error {

	for it := inbox.GetActivityStreamsOrderedItems().Begin(); it != nil; it = it.Next() {
		select {

		// check if context was cancelled
		case <-ctx.Done():
			return ctx.Err()

		// otherwise, save the next item
		default:

			item := it.GetType()
			_, itemType, _, err := common.ParseItem(item)

			if err != nil {
				return derp.Wrap(err, "activitypub.Database.SetOutbox", "Error parsing item", item)
			}

			if itemType != activityStreams.ItemTypeInbox {
				return derp.NewBadRequestError("activitypub.Database.SetOutbox", "Item is not an outbox", item)
			}

			if err := db.save(item, "SetInbox via ActivityPub"); err != nil {
				return derp.Wrap(err, "activitypub.Database.SetInbox", "Error saving item", item)
			}
		}
	}

	return nil
}
