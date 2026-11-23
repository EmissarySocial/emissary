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

	// RULE: An actor removes only what its own server published, checked before any other work
	// so a forgery costs no lookups. Without it, any signed actor could remove a Stream's Announce
	// of another server's post, or of a child Stream here, and have the Stream rebroadcast it.
	if err := requireOwnTarget(activity); err != nil {
		return derp.Wrap(err, location, "Validating object origin", activity.ActorID())
	}

	// Try to find the message in the cache
	outboxService := context.factory.Outbox()
	objectID := activity.Object().ID()

	// Find all activities that match the deleted object
	activities, err := outboxService.RangeByObjectID(context.session, model.FollowerTypeStream, context.stream.StreamID, objectID)

	if err != nil {
		return derp.Wrap(err, location, "Locating matching activities", objectID)
	}

	// Delete all outbox activities that match the deleted object
	for message := range activities {

		if err := outboxService.Delete(context.session, &message, "Removed via ActivityPub"); err != nil {
			return derp.Wrap(err, location, "Deleting message", "outboxMessageId: "+message.OutboxMessageID.Hex())
		}
	}

	// Remove the Stream's own Announces of the object. Nothing is sent for them, because the
	// Announce of the Delete below already tells every follower.
	if err := context.factory.Outbox2().RemoveByObjectID(context.session, model.ActorTypeStream, context.stream.StreamID, objectID); err != nil {
		return derp.Wrap(err, location, "Removing announces", objectID)
	}

	// Try to load the Actor for this user
	actor, err := context.ActivityPubActor()

	if err != nil {
		return derp.Wrap(err, location, "Loading actor", "streamId: "+context.stream.StreamID.Hex())
	}

	// Announce the deletion to the stream's followers through the delivery pipeline, after this transaction commits
	announceID := activitypub.FakeActivityID(activity)
	followersURL := actor.ActorID() + "/pub/followers"
	if err := context.factory.Outbox2().SendAnnounce(context.session, actor.ActorID(), announceID, activity, followersURL); err != nil {
		return derp.Wrap(err, location, "Announcing deletion", context.stream.StreamID)
	}

	// Voila!
	return nil
}

// requireOwnTarget returns a Forbidden error when a Delete or Undo names an object whose id does not
// share its actor's origin, including an object with no id or a urn:uuid
func requireOwnTarget(activity streams.Document) error {

	const location = "handler.activitypub_stream.requireOwnTarget"

	if object := activity.Object(); !object.IsSameOrigin(activity.ActorID()) {
		return derp.Forbidden(location, "Object must share the actor's origin", activity.ActorID(), object.ID())
	}

	return nil
}
