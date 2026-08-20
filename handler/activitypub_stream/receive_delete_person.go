package activitypub_stream

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the handler for inbound Delete/Person activities
func init() {
	streamRouter.Add(vocab.ActivityTypeDelete, vocab.ActorTypePerson, func(context Context, activity streams.Document) error {

		const location = "handler.activityPub_stream.DeletePerson"

		// RULE: Actors can only delete themselves, not other actors
		if activity.ActorID() != activity.Object().ID() {
			return derp.Forbidden(location, "Actor and Object must be the same", activity.ActorID(), activity.Object().ID())
		}

		// Delete from the cache
		if err := context.factory.ActivityStream().Delete(activity.Object().ID()); err != nil {
			return derp.Wrap(err, location, "Deleting stream", activity.Object().ID())
		}

		// Voila!
		return nil
	})
}
