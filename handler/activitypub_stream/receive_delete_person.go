package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	streamRouter.Add(vocab.ActivityTypeDelete, vocab.ActorTypePerson, func(context Context, activity streams.Document) error {

		const location = "handler.activityPub_stream.DeletePerson"

		// RULE: Actors can only delete themselves, not other actors
		if activity.Actor().ID() != activity.Object().ID() {
			return derp.ForbiddenError(location, "Actor and Object must be the same", activity.Actor().ID(), activity.Object().ID())
		}

		// Get an ActivityStream service for the Stream
		activityService := context.factory.ActivityStream(model.ActorTypeStream, context.stream.StreamID)

		// Delete from the cache
		if err := activityService.Delete(activity.Object().ID()); err != nil {
			return derp.Wrap(err, location, "Unable to delete stream", activity.Object().ID())
		}

		// Voila!
		return nil
	})
}
