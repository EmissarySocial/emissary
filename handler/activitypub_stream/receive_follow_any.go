package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	streamRouter.Add(vocab.ActivityTypeFollow, vocab.Any, func(context Context, activity streams.Document) error {

		const location = "handler.activityPub_stream.FollowAny"

		// Validate that the receiving Stream matches the Actor ID in the Activity
		if context.stream.ActivityPubURL() != activity.Object().ID() {
			return derp.NewInternalError(location, "Invalid User ID", context.stream.ActivityPubURL(), activity.Object().ID())
		}

		// Apply rules to filter out unwanted follow activities
		ruleFilter := context.factory.Rule().Filter(primitive.NilObjectID, service.WithBlocksOnly())
		if ruleFilter.Disallow(&activity) {
			return derp.NewForbiddenError(location, "Blocked by rule", activity.Object().ID())
		}

		// Try to look up the complete actor record from the activity
		document, err := activity.Actor().Load()

		if err != nil {
			return derp.Wrap(err, location, "Error parsing actor", activity)
		}

		// Try to create a new follower record
		followerService := context.factory.Follower()
		follower := model.NewFollower()
		if err := followerService.NewActivityPubFollower(model.FollowerTypeStream, context.stream.StreamID, document, &follower); err != nil {
			return derp.Wrap(err, location, "Error creating new follower", context.stream)
		}

		// Send an "Accept" message to the Requester
		actor, err := context.ActivityPubActor(false)

		if err != nil {
			return derp.Wrap(err, location, "Error loading actor", context.stream)
		}

		acceptID := followerService.ActivityPubID(&follower)
		actor.SendAccept(acceptID, activity)

		// Voila!
		return nil
	})
}
