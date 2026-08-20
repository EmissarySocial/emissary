package activitypub_user

import (
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the handler for inbound Delete activities
func init() {
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.Any, func(context Context, activity streams.Document) error {

		const location = "handler.activitypub_user.inbox_DeleteAny"

		// RULE: No further processing required for non-public activities
		if activity.NotPublic() {
			return nil
		}

		// RULE: Actors can only delete objects from their own origin, not evict arbitrary cache entries (D19)
		if !activitypub.IsSameOrigin(activity.ActorID(), activity.Object().ID()) {
			return derp.Forbidden(location, "Actor and Object must share the same origin", activity.ActorID(), activity.Object().ID())
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
