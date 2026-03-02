package activitypub_user

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.Any, func(context Context, activity streams.Document) error {

		// RULE: No further processing required for non-public activities
		if activity.NotPublic() {
			return nil
		}

		// Get an ActivityStream service for the User
		client := context.factory.ActivityStream().UserClient(context.user.UserID)

		// Force reload of the cache.  If the activity is still there, then it will be refreshed.
		// If the activity is gone, then it will be removed from the cache.
		_ = client.Delete(activity.Object().ID())

		// Who let the dogs out?
		return nil
	})
}
