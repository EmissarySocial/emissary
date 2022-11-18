package db

import (
	"context"

	"github.com/EmissarySocial/emissary/gofed/activityStreams"
	"github.com/EmissarySocial/emissary/gofed/common"
	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams/vocab"
)

/*
This method accepts a modified vocab.ActivityStreamsOrderedCollectionPage that had been returned by GetOutbox to update the underlying outbox.

It is similar in behavior to its SetInbox counterpart, but for the actor's Outbox instead.

See the similar documentation for SetInbox.

*/

func (db *Database) SetOutbox(ctx context.Context, outbox vocab.ActivityStreamsOrderedCollectionPage) error {

	for it := outbox.GetActivityStreamsOrderedItems().Begin(); it != nil; it = it.Next() {
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

			if itemType != activityStreams.ItemTypeOutbox {
				return derp.NewBadRequestError("activitypub.Database.SetOutbox", "Item is not an outbox", item)
			}

			if err := db.save(item, "SetOutbox via ActivityPub"); err != nil {
				return derp.Wrap(err, "activitypub.Database.SetInbox", "Error saving item", item)
			}
		}
	}

	return nil
}
