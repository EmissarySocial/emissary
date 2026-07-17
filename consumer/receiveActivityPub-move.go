package consumer

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// ReceiveActivityPubMove processes an incoming ActivityPub Move activity
// This only works for an `Actor` who is moving themselves to a new URL
func ReceiveActivityPubMove(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.ReceiveActivityPubMove"

	// Collect arguments
	actorURL := args.GetString("actor")
	objectURL := args.GetString("object") // nolint:scopeguard
	targetURL := args.GetString("target")

	// RULE: The Actor and Object must be the same URL
	if actorURL != objectURL {
		return queue.Failure(derp.BadRequest(location, "Actors can only `Move` themselves.", "actor: "+actorURL, "object: "+objectURL))
	}

	// Load and validate the Target actor
	client := factory.ActivityStream().AppClient()
	target, err := client.Load(targetURL, ascache.WithWriteOnly())

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Loading Target document", "target: "+targetURL))
	}

	// RULE: The targe document must be an Actor
	if !target.IsActor() {
		return queue.Failure(derp.BadRequest(location, "Target document must be an Actor", "target", target.Value()))
	}

	// RULE: the target must back-reference the moving actor via `alsoKnownAs`. Without this, ANY actor
	// could redirect all of another actor's local followers to an arbitrary destination -- a
	// follower-hijack primitive. The Move only proceeds if the target claims the origin as an alias.
	if !targetClaimsActor(target, actorURL) {
		return queue.Failure(derp.Forbidden(location, "Target actor does not claim the moving actor in alsoKnownAs", "actor: "+actorURL, "target: "+targetURL))
	}

	// Try to update all "Following" records that point to the old Actor URL
	followingService := factory.Following()
	ruleService := factory.Rule()

	followings, err := followingService.RangeByActorID(session, actorURL)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Loading Following records", "actor: "+actorURL))
	}

	for following := range followings {
		if err := migrateOrBlockFollowing(session, ruleService, followingService, following, actorURL, targetURL); err != nil {
			return queue.Error(derp.Wrap(err, location, "Migrating Following record", "followingID", following.FollowingID))
		}
	}

	// Try to remove the original Actor from the cache
	if err := factory.ActivityStream().Delete(actorURL); err != nil {
		if !derp.IsNotFound(err) {
			return queue.Error(derp.Wrap(err, location, "Removing Actor from cache", "actor: "+actorURL))
		}
	}

	// Ohio.
	return queue.Success()
}

// migrateOrBlockFollowing moves a single Following record to the Move target, UNLESS the following's
// User blocks the origin actor -- in which case it copies the block to the target instead of following
// the actor to its new home (R20).
func migrateOrBlockFollowing(session data.Session, ruleService *service.Rule, followingService *service.Following, following model.Following, actorURL string, targetURL string) error {

	const location = "consumer.migrateOrBlockFollowing"

	disposition, err := ruleService.DispositionForKeys(session, following.UserID, model.ActorMatchKeys(actorURL), time.Now().Unix())

	if err != nil {
		return derp.Wrap(err, location, "Checking block rules", "userID", following.UserID)
	}

	if disposition.IsBlocked() {
		return ruleService.BlockActor(session, following.UserID, targetURL, "Blocked actor moved to this account")
	}

	return followingService.Move(session, &following, targetURL)
}

// targetClaimsActor reports whether the target actor's `alsoKnownAs` back-references the moving actor,
// which is the target's proof of consent to receive the Move. `alsoKnownAs` may be a single value or a
// list; a bare-string alias is read from its String(), an object alias from its ID().
func targetClaimsActor(target streams.Document, actorURL string) bool {

	for alias := range target.Get("alsoKnownAs").Range() {

		value := alias.ID()

		if value == "" {
			value = alias.String()
		}

		if value == actorURL {
			return true
		}
	}

	return false
}
