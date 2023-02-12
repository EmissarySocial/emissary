package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/jsonld"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeAccept, vocab.Any, func(factory *domain.Factory, activity jsonld.Reader) error {

		// Look up the requested user account
		userService := factory.User()

		// Try to load the user's account
		user := model.NewUser()
		objectID := activity.Object().AsString()

		if err := userService.LoadByProfileURL(objectID, &user); err != nil {
			return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error loading user", objectID)
		}

		// TODO: HIGH: Enforce blocks here.
		// TODO: Are there limits on who we allow to follow?

		followerService := factory.Follower()
		if err := followerService.NewActivityPubFollower(&user, activity); err != nil {
			return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error creating new follower", user)
		}

		// Load the Actor for this user
		actor, err := userService.ActivityPubActor(&user)

		if err != nil {
			return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error loading actor", user)
		}

		if err := pub.PostAccept(actor, activity); err != nil {
			return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error sending Accept request", actor, activity)
		}

		return nil
	})
}
