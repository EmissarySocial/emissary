package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ReceiveActivityPubMove processes an incoming ActivityPub Move activity
// This only works for an `Actor` who is moving themselves to a new URL
func ReceiveActivityPubMove(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.ReceiveActivityPubMove"

	// Collect arguments
	actorURL := args.GetString("actor")
	objectURL := args.GetString("object")
	targetURL := args.GetString("target")

	// RULE: The Actor and Object must be the same URL
	if actorURL != objectURL {
		return queue.Failure(derp.BadRequest(location, "Actors can only `Move` themselves.", "actor: "+actorURL, "object: "+objectURL))
	}

	// Load and validate the Target actor
	activityService := factory.ActivityStream(model.ActorTypeApplication, primitive.NilObjectID)
	target, err := activityService.Client().Load(targetURL, ascache.WithWriteOnly())

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to load Target document", "target: "+targetURL))
	}

	// RULE: The targe document must be an Actor
	if !target.IsActor() {
		return queue.Failure(derp.BadRequest(location, "Target document must be an Actor", "target", target.Value()))
	}

	// Try to update all "Following" records that point to the old Actor URL
	followingService := factory.Following()
	followings, err := followingService.RangeByActorID(session, actorURL)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to load Following records", "actor: "+actorURL))
	}

	for following := range followings {

		// Move the Following record to the new target actor
		if err := followingService.Move(session, &following, targetURL); err != nil {
			return queue.Error(derp.Wrap(err, location, "Unable to update Following record", "followingID", following.FollowingID))
		}
	}

	// Try to remove the original Actor from the cache
	if err := activityService.CacheClient().Delete(actorURL); err != nil {
		if !derp.IsNotFound(err) {
			return queue.Error(derp.Wrap(err, location, "Unable to remove Actor from cache", "actor: "+actorURL))
		}
	}

	// Ohio.
	return queue.Success()
}
