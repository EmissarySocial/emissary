package gofed

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

// SetOutbox accepts a modified vocab.ActivityStreamsOrderedCollectionPage that had
// been returned by GetOutbox to update the underlying outbox.
//
// It is similar in behavior to its SetInbox counterpart, but for the actor's
// Outbox instead.  See the similar documentation for SetInbox.
func (db Database) SetOutbox(c context.Context, outbox vocab.ActivityStreamsOrderedCollectionPage) error {

	// TODO: CRITICAL: Actually write this function
	spew.Dump("SetOutbox")
	spew.Dump(streams.Serialize(outbox))

	return nil
	/*
		const location = "gofed.Database.SetInbox"

		items := outbox.GetActivityStreamsOrderedItems()

		for iterator := items.Begin(); iterator != items.End(); iterator = iterator.Next() {
			item := iterator.GetType()

			spew.Dump(item)
			activityStream, err := ToModel(item, model.ActivityStreamContainerInbox)

			if err != nil {
				return derp.Wrap(err, location, "Error converting inbox item", item)
			}

			activityStream.Container = model.ActivityStreamContainerOutbox

			if err := db.activityStreamService.Save(&activityStream, "Created"); err != nil {
				return derp.Wrap(err, location, "Error saving ActivityStream", activityStream)
			}
		}

		return nil
	*/
}
