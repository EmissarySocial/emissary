package gofed

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

// This method accepts a modified vocab.ActivityStreamsOrderedCollectionPage that
// had been returned by GetInbox. Right now the library only prepends new items to
// the orderedItems property, so simple diffing can be done. This method should
// then modify the actual underlying inbox to reflect the change in this page.
func (db Database) SetInbox(c context.Context, inbox vocab.ActivityStreamsOrderedCollectionPage) error {

	// TODO: CRITICAL: Actually write this function
	spew.Dump("SetInbox")
	spew.Dump(streams.Serialize(inbox))
	return nil

	/*

		const location = "gofed.Database.SetInbox"

		items := inbox.GetActivityStreamsOrderedItems()

		for iterator := items.Begin(); iterator != items.End(); iterator = iterator.Next() {

			item := iterator.GetType()
			activityStream, err := ToModel(item, model.ActivityStreamContainerInbox)

			if err != nil {
				return derp.Wrap(err, location, "Error converting inbox item", item)
			}

			activityStream.Container = model.ActivityStreamContainerInbox

			if err := db.activityStreamService.Save(&activityStream, "Created"); err != nil {
				return derp.Wrap(err, location, "Error saving ActivityStream", activityStream)
			}
		}

		return nil
	*/
}
