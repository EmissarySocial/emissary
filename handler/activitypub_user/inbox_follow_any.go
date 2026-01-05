package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeFollow, vocab.Any, func(context Context, activity streams.Document) error {

		const location = "handler.activitypub_user.inbox_FollowAny"

		// Look up the requested user account
		userService := context.factory.User()

		// Try to verify the User
		userID, err := service.ParseProfileURL_UserID(activity.Object().ID())

		if err != nil {
			return derp.Wrap(err, location, "Invalid User URL", activity.Object().ID())
		}

		if userID != context.user.UserID {
			return derp.Internal(location, "Invalid User ID", userID, context.user.UserID)
		}

		// RULE: Do not allow new "Follows" of any blocked Actors
		ruleFilter := context.factory.Rule().Filter(context.user.UserID, service.WithBlocksOnly()) // nolint:scopeguard
		if ruleFilter.Disallow(context.session, &activity) {
			return derp.Forbidden(location, "Blocked by rule", activity.Object().ID())
		}

		// Try to look up the complete actor record from the activity
		document, err := activity.Actor().Load()

		if err != nil {
			return derp.Wrap(err, location, "Error parsing actor", activity)
		}

		// Try to create a new follower record
		followerService := context.factory.Follower()
		follower := model.NewFollower()
		if err := followerService.NewActivityPubFollower(context.session, model.FollowerTypeUser, context.user.UserID, document, &follower); err != nil {
			return derp.Wrap(err, location, "Unable to create new follower", context.user)
		}

		// Try to load the Actor for this user
		actor, err := userService.ActivityPubActor(context.session, context.user.UserID)

		if err != nil {
			return derp.Wrap(err, location, "Unable to load actor", context.user)
		}

		// Sen the "Accept" message to the Requester
		acceptID := followerService.ActivityPubID(&follower)
		actor.SendAccept(acceptID, activity)

		// Voila!
		return nil
	})
}
