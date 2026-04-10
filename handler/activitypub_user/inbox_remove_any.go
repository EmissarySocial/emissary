package activitypub_user

/*
func init() {
	inboxRouter.Add(vocab.ActivityTypeRemove, vocab.Any, inbox_RemoveAny)
}

// inbox_RemoveAny implements FEP-7888 Remove(Object, Collection) workflow to backfill
// discussions and preload the cache when we receive a Remove activity.
func inbox_RemoveAny(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_AddAny"

	// RULE: For now, no additional processing is required for non-public activities.
	if activity.NotPublic() {
		return nil
	}

	// Gonna need the followingService in a hot sec..
	followingService := context.factory.Following()
	following := model.NewFollowing()

	// RULE: Only process Add activities from Actors that we Follow.
	if err := followingService.LoadByURL(context.session, context.user.UserID, activity.Actor().ID(), &following); err != nil {
		return derp.Wrap(err, location, "Unable to locate `Following` record", context.user.UserID)
	}

	// Add a task to the queue to backfill the context of this activity
	queue := context.factory.Queue()
	queue.NewTask("Context-Reload", mapof.Any{
		"context": activity.Target().ID(),
	})

	return nil
}
*/
