package activitypub_search

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

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

		// RULE: Do not allow new "Follows" of any blocked Actors
		ruleFilter := context.factory.Rule().Filter(context.searchQuery.SearchQueryID, service.WithBlocksOnly()) // nolint:scopeguard
		if ruleFilter.Disallow(context.session, &activity) {
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
