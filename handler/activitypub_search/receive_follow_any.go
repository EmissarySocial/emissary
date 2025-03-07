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
		actorURL := searchQueryService.ActivityPubURL(context.searchQuery)

		if activity.Object().ID() != actorURL {
			return derp.NewInternalError(location, "Invalid Search Query ID", actorURL, activity.Object().ID())
		}

		// RULE: Do not allow new "Follows" of any blocked Actors
		ruleFilter := context.factory.Rule().Filter(context.searchQuery.SearchQueryID, service.WithBlocksOnly())
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
		if err := followerService.NewActivityPubFollower(model.FollowerTypeSearch, context.searchQuery.SearchQueryID, document, &follower); err != nil {
			return derp.Wrap(err, location, "Error creating new follower", context.searchQuery)
		}

		// Try to load the Actor for this user
		actor, err := searchQueryService.ActivityPubActor(context.searchQuery, false)

		if err != nil {
			return derp.Wrap(err, location, "Error loading actor", context.searchQuery)
		}

		// Sen the "Accept" message to the Requester
		acceptID := followerService.ActivityPubID(&follower)
		actor.SendAccept(acceptID, activity)

		// Voila!
		return nil
	})
}
