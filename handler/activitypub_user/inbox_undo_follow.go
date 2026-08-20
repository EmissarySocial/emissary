package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the handlers for inbound Undo/Follow and Delete/Follow activities
func init() {
	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeFollow, inbox_UndoFollow)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeFollow, inbox_UndoFollow)
}

// inbox_UndoFollow handles "Undo/Follow" and "Delete/Follow" activitites, which means
// that this code is called when a remote user unfollows an actor on this server.
func inbox_UndoFollow(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_UndoFollow"

	// Try to load the existing follower record
	followerService := context.factory.Follower()
	follower := model.NewFollower()

	// Collect data from the original follow
	actorURL := activity.ActorID() // The "actor" is our follower.actor.ProfileURL

	if err := followerService.LoadByActivityPubFollower(context.session, model.FollowerTypeUser, context.user.UserID, actorURL, &follower); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Locating follower", activity.Value(), context.user.UserID, actorURL)
	}

	// Try to delete the existing follower record
	if err := followerService.Delete(context.session, &follower, "Removed by remote client"); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Deleting follower", follower)
	}

	// Remove the corresponding FOLLOW notification (if any), matched by the unfollowing actor.
	// The central NotifyFromActivity hook already attempts this for embedded Undo/Follow objects;
	// this repeats it (idempotently) to also cover any Undo whose object had to be dereferenced to
	// route here.  A cleanup failure must not fail the unfollow.
	if err := context.factory.Notification().DeleteFollowByActor(context.session, context.user.UserID, actorURL, "unfollow"); err != nil {
		derp.Report(derp.Wrap(err, location, "Deleting follow notification", context.user.UserID, actorURL))
	}

	// Voila!
	return nil
}
