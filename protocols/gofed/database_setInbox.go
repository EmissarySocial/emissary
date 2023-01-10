package gofed

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-fed/activity/streams/vocab"
)

// This method accepts a modified vocab.ActivityStreamsOrderedCollectionPage that
// had been returned by GetInbox. Right now the library only prepends new items to
// the orderedItems property, so simple diffing can be done. This method should
// then modify the actual underlying inbox to reflect the change in this page.
func (db Database) SetInbox(c context.Context, inbox vocab.ActivityStreamsOrderedCollectionPage) error {

	const location = "gofed.Database.SetInbox"

	items := inbox.GetActivityStreamsOrderedItems()

	for iterator := items.Begin(); iterator != items.End(); iterator = iterator.Next() {
		item := iterator.GetType()
		activity, err := ToModel(item, model.ActivityPlaceInbox)

		if err != nil {
			return derp.Wrap(err, location, "Error converting inbox item", item)
		}

		spew.Dump("I should eventually insert this..", activity)

		// TODO: CRITICAL: How to identify duplicates?
	}

	return nil
}
