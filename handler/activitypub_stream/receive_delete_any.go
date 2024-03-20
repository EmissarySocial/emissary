package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	streamRouter.Add(vocab.ActivityTypeDelete, vocab.Any, func(context Context, activity streams.Document) error {

		const location = "handler.activityPub_stream.DeleteAny"

		// Try to find the message in the cache
		outboxService := context.factory.Outbox()
		message := model.NewOutboxMessage()
		objectID := activity.Object().ID()

		if err := outboxService.LoadByURL(model.FollowerTypeStream, context.stream.StreamID, objectID, &message); err != nil {
			if derp.NotFound(err) {
				return nil
			}
			return derp.Wrap(err, location, "Error loading message", objectID)
		}

		// If Found, delete the message
		if err := outboxService.Delete(&message, "Removed via ActivityPub"); err != nil {
			return derp.Wrap(err, location, "Error deleting message", message)
		}

		// Try to load the Actor for this user
		actor, err := context.ActivityPubActor(true)

		if err != nil {
			return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error loading actor", context.stream)
		}

		// Announce the deleted object
		announceID := activitypub.FakeActivityID(activity)
		actor.SendAnnounce(announceID, activity)

		// Voila!
		return nil
	})
}
