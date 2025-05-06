package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeFollow, undoFollow)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeFollow, undoFollow)
}

// undoFollow handles "Undo/Follow" and "Delete/Follow" activitites, which means
// that this code is called when a remote user unfollows an actor on this server.
func undoFollow(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.undoFollow"

	// Try to load the existing follower record
	followerService := context.factory.Follower()
	follower := model.NewFollower()

	// Collect data from the original follow
	actorURL := activity.Actor().ID() // The "actor" is our follower.actor.ProfileURL

	if err := followerService.LoadByActivityPubFollower(model.FollowerTypeUser, context.user.UserID, actorURL, &follower); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Error locating follower", activity.Value(), context.user.UserID, actorURL)
	}

	// Try to delete the existing follower record
	if err := followerService.Delete(&follower, "Removed by remote client"); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Error deleting follower", follower)
	}

	// Voila!
	return nil
}
