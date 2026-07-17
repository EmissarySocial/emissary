package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
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
			return derp.Internal(location, "Invalid User ID", context.stream.ActivityPubURL(), activity.Object().ID())
		}

		// RULE: A blocked actor may not Follow. Verify first (D5 exception set), then reject loudly --
		// the check lives HERE because the wire gate deliberately defers Follows; see
		// activitypub.IsWireGateException (handler/activitypub/ruleValidator.go) for why Follow
		// rejects loudly while other activity types are silently discarded.
		blocked, err := context.factory.Rule().IsActorBlocked(context.session, primitive.NilObjectID, activity)

		if err != nil {
			return derp.Wrap(err, location, "Checking block rules", activity.ActorID())
		}

		if blocked {
			return derp.Forbidden(location, "Blocked by rule", activity.Object().ID())
		}

		// Try to look up the complete actor record from the activity
		document, err := activity.Actor().Load()

		if err != nil {
			return derp.Wrap(err, location, "Parsing actor", activity)
		}

		// Try to create a new follower record
		followerService := context.factory.Follower()
		follower := model.NewFollower()
		if err := followerService.NewActivityPubFollower(context.session, model.FollowerTypeStream, context.stream.StreamID, document, &follower); err != nil {
			return derp.Wrap(err, location, "Creating new follower", context.stream)
		}

		// Send an "Accept" message to the Requester
		actor, err := context.ActivityPubActor()

		if err != nil {
			return derp.Wrap(err, location, "Loading actor", context.stream)
		}

		// Send the "Accept" as a post-commit queue task (F3): the signed delivery happens after this
		// transaction commits and is independently retryable.
		acceptID := followerService.ActivityPubID(&follower)
		context.factory.Outbox().SendAccept(context.session, actor.ActorID(), acceptID, activity)

		// Voila!
		return nil
	})
}
