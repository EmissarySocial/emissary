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

func DeleteAny(context Context, activity streams.Document) error {

	const location = "handler.activityPub_stream.DeleteAny"
	log.Trace().Str("activityType", activity.Type()).Msg(location)

	// Try to find the message in the cache
	outboxService := context.factory.Outbox()
	objectID := activity.Object().ID()

	// Find all activities that match the deleted object
	activities, err := outboxService.RangeByObjectID(model.FollowerTypeStream, context.stream.StreamID, objectID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to locate matching activities", objectID)
	}

	// Delete all outbox activities that match the deleted object
	for activity := range activities {

		if err := outboxService.Delete(&activity, "Removed via ActivityPub"); err != nil {
			return derp.Wrap(err, location, "Error deleting message", activity)
		}
	}

	// Try to load the Actor for this user
	actor, err := context.ActivityPubActor()

	if err != nil {
		return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error loading actor", context.stream)
	}

	// Announce the deleted object
	announceID := activitypub.FakeActivityID(activity)
	actor.SendAnnounce(announceID, activity)

	// Voila!
	return nil
}
