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
		ruleService := context.factory.Rule()

		// RULE: Require that the search query in the document matches the search query inbox.
		if actorURL := searchDomainService.ActivityPubURL(); actorURL != activity.Object().ID() {
			return derp.Internal(location, "Invalid Search Query ID", actorURL, activity.Object().ID())
		}

		// RULE: Do not allow new "Follows" of any blocked Actors
		if filter := ruleService.Filter(primitive.NilObjectID, service.WithBlocksOnly()); filter.Disallow(context.session, &activity) {
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
		if err := followerService.NewActivityPubFollower(context.session, model.FollowerTypeSearchDomain, primitive.NilObjectID, document, &follower); err != nil {
			return derp.Wrap(err, location, "Creating new follower")
		}

		// Try to load the Actor for this user
		actor, err := searchDomainService.ActivityPubActor(context.session)

		if err != nil {
			return derp.Wrap(err, location, "Loading domain search actor")
		}

		// Send the "Accept" message to the Requester as a post-commit queue task (F3): the signed
		// delivery happens after this transaction commits and is independently retryable.
		acceptID := followerService.ActivityPubID(&follower)
		context.factory.Outbox().SendAccept(context.session, actor.ActorID(), acceptID, activity)

		// Voila!
		return nil
	})
}
