package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
)

// init registers the handlers for inbound Delete and Undo activities
func init() {
	streamRouter.Add(vocab.ActivityTypeDelete, vocab.Any, DeleteAny)
	streamRouter.Add(vocab.ActivityTypeUndo, vocab.Any, DeleteAny)
}

// DeleteAny handles inbound Delete and Undo activities on a Stream: it removes any matching
// outbox messages and re-announces the removal to followers.
//
// NOTE: reaction (Like/Dislike/Announce) projection is owned SOLELY by the User inbox — reactions
// are delivered to the reacted-to object's author, never to a Stream's inbox (see
// COLLECTIONS-REDESIGN.md D8). This handler therefore does NOT touch response collections.
func DeleteAny(context Context, activity streams.Document) error {

	const location = "handler.activityPub_stream.DeleteAny"
	log.Trace().Str("activityType", activity.Type()).Msg(location)

	// Try to find the message in the cache
	outboxService := context.factory.Outbox()
	objectID := activity.Object().ID()

	// Find all activities that match the deleted object
	activities, err := outboxService.RangeByObjectID(context.session, model.FollowerTypeStream, context.stream.StreamID, objectID)

	if err != nil {
		return derp.Wrap(err, location, "Locating matching activities", objectID)
	}

	// Delete all outbox activities that match the deleted object
	for activity := range activities {

		if err := outboxService.Delete(context.session, &activity, "Removed via ActivityPub"); err != nil {
			return derp.Wrap(err, location, "Deleting message", activity)
		}
	}

	// Try to load the Actor for this user
	actor, err := context.ActivityPubActor()

	if err != nil {
		return derp.Wrap(err, "handler.activityPub_stream.DeleteAny", "Loading actor", context.stream)
	}

	// Announce the deletion to the stream's followers as a post-commit send (F3, W6 option B).
	announceID := activitypub.FakeActivityID(activity)
	followersURL := actor.ActorID() + "/pub/followers"
	outboxService.SendAnnounce(context.session, actor.ActorID(), announceID, activity, followersURL)

	// Voila!
	return nil
}
