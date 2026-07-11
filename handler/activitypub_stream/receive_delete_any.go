package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
)

func init() {
	streamRouter.Add(vocab.ActivityTypeDelete, vocab.Any, DeleteAny)
	streamRouter.Add(vocab.ActivityTypeUndo, vocab.Any, DeleteAny)
}

// DeleteAny handles inbound Delete and Undo activities on a Stream: it removes any matching
// outbox messages, un-projects an undone Like/Dislike/Announce from the Stream's response
// collection, and re-announces the removal to followers.
func DeleteAny(context Context, activity streams.Document) error {

	const location = "handler.activityPub_stream.DeleteAny"
	log.Trace().Str("activityType", activity.Type()).Msg(location)

	// Try to find the message in the cache
	outboxService := context.factory.Outbox()
	objectID := activity.Object().ID()

	// If this Undo/Delete targets a Like/Dislike/Announce on one of our Streams, remove the
	// corresponding item from that Stream's response collection (symmetric with BoostAny's add).
	if err := removeResponseCollectionItem(context, activity); err != nil {
		return derp.Wrap(err, location, "Unable to remove response from collection", objectID)
	}

	// Find all activities that match the deleted object
	activities, err := outboxService.RangeByObjectID(context.session, model.FollowerTypeStream, context.stream.StreamID, objectID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to locate matching activities", objectID)
	}

	// Delete all outbox activities that match the deleted object
	for activity := range activities {

		if err := outboxService.Delete(context.session, &activity, "Removed via ActivityPub"); err != nil {
			return derp.Wrap(err, location, "Unable to delete message", activity)
		}
	}

	// Try to load the Actor for this user
	actor, err := context.ActivityPubActor()

	if err != nil {
		return derp.Wrap(err, "handler.activityPub_stream.DeleteAny", "Unable to load actor", context.stream)
	}

	// Announce the deleted object
	announceID := activitypub.FakeActivityID(activity)
	actor.SendAnnounce(announceID, activity)

	// Voila!
	return nil
}

// removeResponseCollectionItem handles the response-collection side of an inbound Undo/Delete.
// When the wrapped object is a Like/Dislike/Announce that targets one of our Streams, it removes
// the matching item from that Stream's collection and refreshes the count. It is a quiet no-op for
// any other wrapped type (the service method filters by type), so it is safe to call on every Undo/Delete.
func removeResponseCollectionItem(context Context, activity streams.Document) error {

	// The wrapped object is the original Like/Dislike/Announce activity being undone.
	originalActivity, err := activity.Object().Load()

	if err != nil {
		// A wrapped object we can't resolve is not one we can un-project; leave the rest of the
		// Undo/Delete flow (outbox cleanup) to proceed.
		return nil
	}

	// activityURL is the original activity's ID (the collection item key); targetURL is the Stream it responded to.
	return context.factory.Stream().RemoveResponseCollectionItem(
		context.session,
		originalActivity.Object().ID(),
		originalActivity.Type(),
		originalActivity.ID(),
	)
}
