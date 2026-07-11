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

	// Read the wrapped original activity with LoadLink, NOT Load. LoadLink returns an INLINE object
	// as-is (no fetch) but dereferences a bare-URL reference:
	//   - OUR Undos embed the original activity inline (D7). Load() would still HTTP-fetch object.id,
	//     but that Like/Dislike/Announce was usually hard-deleted by the sender, so the fetch 404s.
	//     LoadLink sees the inline map and skips the fetch.
	//   - OTHER servers may send Undo with object as a bare URL reference; LoadLink fetches those.
	originalActivity := activity.Object().LoadLink()

	// A wrapped object we still can't resolve (bare reference that no longer exists, or a fetch
	// failure) has no type — nothing to un-project. No-op and let the rest of the Undo/Delete flow
	// (outbox cleanup) proceed.
	if originalActivity.Type() == "" {
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
