package activitypub_search

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the handlers for inbound Undo/Follow and Delete/Follow activities
func init() {
	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeFollow, undoFollow)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeFollow, undoFollow)
}

// undoFollow handles "Undo/Follow" and "Delete/Follow" activitites, which means
// that this code is called when a remote user unfollows an actor on this server.
func undoFollow(context Context, activity streams.Document) error {

	const location = "handler.activitypub_search.undoFollow"

	// Try to load the existing follower record
	followerService := context.factory.Follower()
	follower := model.NewFollower()

	// Collect data from the original follow
	actorURL := activity.ActorID() // The "actor" is our follower.actor.ProfileURL

	if err := followerService.LoadByActivityPubFollower(context.session, model.FollowerTypeSearch, context.searchQuery.SearchQueryID, actorURL, &follower); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Locating follower", activity.Value(), context.searchQuery.SearchQueryID, actorURL)
	}

	// Try to delete the existing follower record
	if err := followerService.Delete(context.session, &follower, "Removed by remote client"); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Deleting follower", follower)
	}

	// Voila!
	return nil
}
