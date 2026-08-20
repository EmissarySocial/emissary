package activitypub_search

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// init registers the handler for inbound Follow activities
func init() {
	inboxRouter.Add(vocab.ActivityTypeFollow, vocab.Any, func(context Context, activity streams.Document) error {

		const location = "handler.activityPub_search.ReceiveFollow"

		// Look up the requested search query
		searchQueryService := context.factory.SearchQuery()

		// RULE: Require that the search query in the document matches the search query inbox.
		actorURL := searchQueryService.ActivityPubURL(context.searchQuery.SearchQueryID) // nolint:scopeguard (readability)

		if activity.Object().ID() != actorURL {
			return derp.Internal(location, "Invalid Search Query ID", actorURL, activity.Object().ID())
		}

		// RULE: A blocked actor may not Follow. Verify first (D5 exception set), then reject loudly --
		// the check lives HERE because the wire gate deliberately defers Follows; see
		// activitypub.IsWireGateException (handler/activitypub/ruleValidator.go) for why Follow
		// rejects loudly while other activity types are silently discarded.
		// Evaluated against admin-tier rules (NilObjectID) -- NOT searchQuery.SearchQueryID, which is a
		// SearchQuery id, not a UserID; passing it would query a nonexistent user's rules and never block.
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
		if err := followerService.NewActivityPubFollower(context.session, model.FollowerTypeSearch, context.searchQuery.SearchQueryID, document, &follower); err != nil {
			return derp.Wrap(err, location, "Creating new follower", context.searchQuery)
		}

		// Try to load the Actor for this user
		actor, err := searchQueryService.ActivityPubActor(context.session, context.searchQuery.SearchQueryID)

		if err != nil {
			return derp.Wrap(err, location, "Loading actor", context.searchQuery)
		}

		// Send the "Accept" message to the Requester as a post-commit queue task (F3): the signed
		// delivery happens after this transaction commits and is independently retryable.
		acceptID := followerService.ActivityPubID(&follower)
		context.factory.Outbox().SendAccept(context.session, actor.ActorID(), acceptID, activity)

		// Voila!
		return nil
	})
}
