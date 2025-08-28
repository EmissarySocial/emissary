package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	streamRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeFollow, undoFollow)
	streamRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeFollow, undoFollow)
}

// undoFollow handles "Undo/Follow" and "Delete/Follow" activitites, which means
// that this code is called when a remote stream unfollows an actor on this server.
func undoFollow(context Context, activity streams.Document) error {

	const location = "handler.activitypub_stream.undoFollow"

	// Try to load the existing follower record
	followerService := context.factory.Follower()
	follower := model.NewFollower()

	// Load the original follow
	originalFollow, err := activity.Object().Load()

	if err != nil {
		if derp.IsNotFound(err) {
			return nil // If there is no follower record, then there's nothing to delete.
		}

		// All other errors are bad, tho.
		return derp.Wrap(err, location, "Error retrieving original follow request", activity.Value())
	}

	// Collect data from the original follow
	actorURL := originalFollow.Actor().ID()   // The "actor" of the original follow is our follower.actor.ProfileURL
	streamURL := originalFollow.Object().ID() // The "object" of the original follow is our local StreamURL
	streamID, err := context.factory.Stream().ParseURL(context.session, streamURL)

	if err != nil {
		return derp.Wrap(err, location, "Invalid User URL", streamURL)
	}

	if err := followerService.LoadByActivityPubFollower(context.session, model.FollowerTypeStream, streamID, actorURL, &follower); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Error loading Follower", activity.Value(), streamID, actorURL)
	}

	// Try to delete the existing follower record
	if err := followerService.Delete(context.session, &follower, "Removed by remote client"); err != nil {
		return derp.Wrap(err, location, "Error deleting follower", follower)
	}

	// Voila!
	return nil
}
