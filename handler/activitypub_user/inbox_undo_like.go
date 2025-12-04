package activitypub_user

import (
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeLike, undoResponse)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeLike, undoResponse)

	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeDislike, undoResponse)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeDislike, undoResponse)

	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeAnnounce, undoResponse)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeAnnounce, undoResponse)
}

// undoResponse handles the Undo/Delete actions on Like/Dislike/Announce records
func undoResponse(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.undoResponse"

	// Try to parse the original Activity from the JSON-LD
	originalActivity, err := activity.Object().Load() // The Object is the original Like/Dislike/Announce activity

	if err != nil {
		return nil
	}

	// RULE: ActivityPub type must match the received activity
	if activity.Actor().ID() != originalActivity.Actor().ID() {
		return derp.UnauthorizedError(location, "Actor undoing this activity must be the same as the original activity")
	}

	// Get/Generate the ID of the original activity
	originalActivityID := originalActivity.ID()

	if originalActivityID == "" {
		originalActivityID = activitypub.FakeActivityID(originalActivity)
	}

	// Get an ActivityStream service for the User
	activityService := context.factory.ActivityStream(model.ActorTypeUser, context.user.UserID)

	// Remove the original activity from the database.
	if err := activityService.Delete(originalActivityID); err != nil {
		return derp.Wrap(err, location, "Error deleting original activity", originalActivity)
	}

	return nil
}
