package activitypub_domain

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeFollow, vocab.Any, func(context Context, activity streams.Document) error {

		const location = "handler.activityPub_domain.ReceiveFollow"

		// Look up the requested search query
		searchDomainService := context.factory.SearchDomain()

		// RULE: Require that the search query in the document matches the search query inbox.
		actorURL := searchDomainService.ActivityPubURL()

		if activity.Object().ID() != actorURL {
			return derp.InternalError(location, "Invalid Search Query ID", actorURL, activity.Object().ID())
		}

		// RULE: Do not allow new "Follows" of any blocked Actors
		ruleFilter := context.factory.Rule().Filter(primitive.NilObjectID, service.WithBlocksOnly())
		if ruleFilter.Disallow(&activity) {
			return derp.ForbiddenError(location, "Blocked by rule", activity.Object().ID())
		}

		// Try to look up the complete actor record from the activity
		document, err := activity.Actor().Load()

		if err != nil {
			return derp.Wrap(err, location, "Error parsing actor", activity)
		}

		// Try to create a new follower record
		followerService := context.factory.Follower()
		follower := model.NewFollower()
		if err := followerService.NewActivityPubFollower(model.FollowerTypeSearchDomain, primitive.NilObjectID, document, &follower); err != nil {
			return derp.Wrap(err, location, "Error creating new follower")
		}

		// Try to load the Actor for this user
		actor, err := searchDomainService.ActivityPubActor()

		if err != nil {
			return derp.Wrap(err, location, "Error loading domain search actor")
		}

		// Sen the "Accept" message to the Requester
		acceptID := followerService.ActivityPubID(&follower)
		actor.SendAccept(acceptID, activity)

		// Voila!
		return nil
	})
}
