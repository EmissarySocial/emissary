package activitypub_user

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.Any, func(context Context, document streams.Document) error {

		// Get an ActivityStream service for the User
		client := context.factory.ActivityStream().UserClient(context.user.UserID)

		// Force reload of the cache.  If the document is still there, then it will be refreshed.
		// If the document is gone, then it will be removed from the cache.
		_ = client.Delete(document.Object().ID())

		// Who let the dogs out?
		return nil
	})
}
