package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/davecgh/go-spew/spew"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeFollow, undoFollow)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeFollow, undoFollow)
}

// undoFollow handles "Undo/Follow" and "Delete/Follow" activitites, which means
// that this code is called when a remote user unfollows an actor on this server.
func undoFollow(factory *domain.Factory, activity streams.Document) error {

	const location = "handler.activityPub_HandleRequest_Undo_Follow"
	spew.Dump("undoFollow", activity.Value())

	// Try to load the existing follower record
	followerService := factory.Follower()
	follower := model.NewFollower()

	// Load the original follow
	originalFollow, err := activity.Object().AsObject()

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving original follow request", activity.Value())
	}

	// Collect data from the original follow
	actorURL := originalFollow.ActorID() // The "actor" of the original follow is our follower.actor.ProfileURL
	userURL := originalFollow.ObjectID() // The "object" of the original follow is our local UserURL
	userService := factory.User()
	userID, err := userService.ParseProfileURL(userURL)

	if err != nil {
		return derp.Wrap(err, location, "Invalid User URL", userURL)
	}

	if err := followerService.LoadByActivityPubFollower(userID, actorURL, &follower); err != nil {
		return derp.Wrap(err, location, "Error locating follower", activity.Value(), userID, actorURL)
	}

	// Try to delete the existing follower record
	if err := followerService.Delete(&follower, "Removed by remote client"); err != nil {
		return derp.Wrap(err, location, "Error deleting follower", follower)
	}

	// Voila!
	return nil
}
