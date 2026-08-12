package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Unfollow guarantees that the user is no longer following the specified URL.  If the user is already following this actor,
// then the Following record is disconnected. Otherwise, no action is taken.
func (service *Following) Unfollow(session data.Session, userID primitive.ObjectID, actorID string) error {

	const location = "service.Following.Unfollow"

	// If the actor ID is not a valid URL, it's probably a username/handle,
	// so try to resolve it into a URL using Sherlock/WebFinger.
	if uri.NotValidURL(actorID) {

		// Look up the Actor from the Activity service
		actor, err := service.activityService.GetActor(actorID)

		if err != nil {
			return derp.Wrap(err, location, "Finding Actor for URL", actorID)
		}

		actorID = actor.ID()
	}

	// Try to load the existing Following record for this user and URL
	following := model.NewFollowing()
	if err := service.LoadByURL(session, userID, actorID, &following); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading following for user and URL", userID, actorID)
	}

	// Disconnect the Following record
	return service.Delete(session, &following, "")
}

// Disconnect notifies the external service that a Following has ended (e.g. an ActivityPub
// Undo/Follow). It only ENQUEUES the notification (no blocking HTTP), so it is safe to call
// synchronously inside the deleting transaction; the actual send happens post-commit.
func (service *Following) Disconnect(session data.Session, following *model.Following) {

	switch following.Method {

	case model.FollowingMethodActivityPub:
		service.disconnect_ActivityPub(session, following)
	}
}

// disconnect_ActivityPub queues an ActivityPub "Undo" of the original "Follow" request, delivered
// post-commit to the followed actor per spec (https://www.w3.org/TR/activitypub/#undo-activity-outbox).
// It carries the Follow as payload (Outbox.SendUndoFollow), NOT a row reference, because the caller
// (Following.deleteNoStats) has already deleted the Following row. See POST-COMMIT-FEDERATION.md F4.
func (service *Following) disconnect_ActivityPub(session data.Session, following *model.Following) {
	followMap := service.AsJSONLD(following)
	actorURL := service.ActivityPubActorID(following)
	service.outboxService.SendUndoFollow(session, actorURL, followMap)
}
