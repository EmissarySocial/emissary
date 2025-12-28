package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.Any, func(context Context, document streams.Document) error {

		// Get an ActivityStream service for the User
		activityService := context.factory.ActivityStream(model.ActorTypeUser, context.user.UserID)

		// Force reload of the cache.  If the document is still there, then it will be refreshed.
		// If the document is gone, then it will be removed from the cache.
		_ = activityService.Client().Delete(document.Object().ID())

		// Who let the dogs out?
		return nil
	})
}
