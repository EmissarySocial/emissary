package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeFollow, vocab.Any, func(factory *domain.Factory, user *model.User, activity streams.Document) error {

		// Look up the requested user account
		userService := factory.User()

		// Try to verify the User
		userID, err := service.ParseProfileURL_UserID(activity.ObjectID())

		if err != nil {
			return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Invalid User URL", activity.ObjectID())
		}

		if userID != user.UserID {
			return derp.New(500, "handler.activityPub_HandleRequest_Follow", "Invalid User ID", userID, user.UserID)
		}

		// TODO: CRITICAL: Enforce blocks here.
		// Are there other limits on who we allow to follow?
		// What about manual accepts?

		// Try to look up the complete actor record from the activity
		follower, err := activity.Actor().Load()

		if err != nil {
			return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error parsing actor", activity)
		}

		// Try to create a new follower record
		followerService := factory.Follower()
		if err := followerService.NewActivityPubFollower(user, follower); err != nil {
			return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error creating new follower", user)
		}

		// Try to load the Actor for this user
		actor, err := userService.ActivityPubActor(user.UserID)

		if err != nil {
			return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error loading actor", user)
		}

		// Send an "Accept" to the requester (queued)
		queue := factory.Queue()
		queue.Run(pub.SendAcceptQueueTask(actor, activity))

		// Voila!
		return nil
	})
}
