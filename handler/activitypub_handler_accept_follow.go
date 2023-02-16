package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/first"
	"github.com/davecgh/go-spew/spew"
)

func init() {

	// This funciton handles ActivityPub "Accept/Follow" activities, meaning that
	// it is called with a remote server accepts our follow request.
	inboxRouter.Add(vocab.ActivityTypeAccept, vocab.ActivityTypeFollow, func(factory *domain.Factory, activity streams.Document) error {

		spew.Dump("Received >>>>>>>>>>>>> Accept/Follow", activity.Value())

		followingService := factory.Following()

		// Parse the Object.ID of the activity, which should be our original "Follow" activity
		userID, followingID, err := service.ParseProfileURL_AsFollowing(activity.ObjectID())

		if err != nil {
			return derp.Wrap(err, "handler.inboxRouter.Accept.Follow", "Error parsing followingID", activity.ObjectID())
		}

		// Try to load the original "Following" record.
		// If it doesn't already exist, then this message is invalid.
		following := model.NewFollowing()
		if err := followingService.LoadByID(userID, followingID, &following); err != nil {
			return derp.Wrap(err, "handler.inboxRouter.Accept.Follow", "Error loading following record", userID, followingID)
		}

		// Populate our "Following" record with the NAME and AVATAR of the remote Actor
		remoteActor := activity.Actor()

		spew.Dump(remoteActor.Value())
		spew.Dump(remoteActor.IconURL(), remoteActor.ImageURL())

		// Upgrade the "Following" record to ActivityPub
		following.Label = remoteActor.Name()
		following.ImageURL = first.String(remoteActor.IconURL(), remoteActor.ImageURL())
		following.Method = model.FollowMethodActivityPub
		following.Secret = ""
		following.PollDuration = 30

		// Save the "Following" record to the database
		if err := followingService.SetStatus(&following, model.FollowingStatusSuccess, ""); err != nil {
			return derp.Wrap(err, "handler.inboxRouter.Accept.Follow", "Error saving following", following)
		}

		return nil
	})
}
