package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// init registers the handler for inbound Follow activities
func init() {
	streamRouter.Add(vocab.ActivityTypeFollow, vocab.Any, func(context Context, activity streams.Document) error {

		const location = "handler.activityPub_stream.FollowAny"

		// RULE: A Follow must be delivered to the inbox of the actor it names.  This is a 422, not a
		// 500: the request is well-formed and the server is healthy -- the sender simply addressed a
		// Follow for one actor to a different actor's inbox, and no retry will change that.
		if context.stream.ActivityPubURL() != activity.Object().ID() {
			return derp.Validation("Follow object does not match the Stream that owns this inbox", context.stream.ActivityPubURL(), activity.Object().ID(), derp.WithLocation(location))
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
