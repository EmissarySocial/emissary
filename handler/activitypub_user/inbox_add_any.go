package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeAdd, vocab.Any, inbox_AddAny)
}

// inbox_AddAny implements FEP-7888 Add(Object, Collection) workflow to backfill
// discussions and preload the cache when we receive an Add activity.
func inbox_AddAny(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_AddAny"

	// RULE: For now, no additional processing is required for non-public activities.
	if activity.NotPublic() {
		return nil
	}

	// Gonna need the followingService in a hot sec..
	followingService := context.factory.Following()
	following := model.NewFollowing()

	// RULE: Only process Add activities from Actors that we Follow.
	if err := followingService.LoadByURL(context.session, context.user.UserID, activity.ActorID(), &following); err != nil {
		return derp.Wrap(err, location, "Locating `Following` record", context.user.UserID)
	}

	// Add a task to the queue (post-commit) to backfill the context of this activity
	postcommit.Publish(context.session, context.factory.Queue(), "ReceiveActivityPub-Add", mapof.Any{
		"actor":  activity.ActorID(),
		"object": activity.Object().ID(),
		"target": activity.Target().ID(),
	})

	return nil
}
