package activitypub_domain

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeFollow, undoFollow)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeFollow, undoFollow)
}

// undoFollow handles "Undo/Follow" and "Delete/Follow" activitites, which means
// that this code is called when a remote user unfollows an actor on this server.
func undoFollow(context Context, activity streams.Document) error {

	const location = "handler.activitypub_domain.undoFollow"

	// Try to load the existing follower record
	followerService := context.factory.Follower()
	follower := model.NewFollower()

	// Collect data from the original follow
	actorURL := activity.Actor().ID() // The "actor" is our follower.actor.ProfileURL

	if err := followerService.LoadByActivityPubFollower(context.session, model.FollowerTypeSearchDomain, primitive.NilObjectID, actorURL, &follower); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Error locating follower", activity.Value(), actorURL)
	}

	// Try to delete the existing follower record
	if err := followerService.Delete(context.session, &follower, "Removed by remote client"); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Error deleting follower", follower)
	}

	// Voila!
	return nil
}
