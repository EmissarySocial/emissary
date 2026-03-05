package activitypub_user

import (
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeLike, inboxUndoLike)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeLike, inboxUndoLike)

	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeDislike, inboxUndoLike)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeDislike, inboxUndoLike)

	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeAnnounce, inboxUndoLike)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeAnnounce, inboxUndoLike)
}

// inboxUndoLike handles the Undo/Delete actions on Like/Dislike/Announce records
func inboxUndoLike(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inboxUndoLike"

	// Try to parse the original Activity from the JSON-LD
	originalActivity, err := activity.Object().Load() // The Object is the original Like/Dislike/Announce activity

	if err != nil {
		return nil
	}

	// RULE: ActivityPub type must match the received activity
	if activity.Actor().ID() != originalActivity.Actor().ID() {
		return derp.Unauthorized(location, "Actor undoing this activity must be the same as the original activity")
	}

	// Get/Generate the ID of the original activity
	originalActivityID := originalActivity.ID()

	if originalActivityID == "" {
		originalActivityID = activitypub.FakeActivityID(originalActivity)
	}

	// Remove the original activity from the database.
	if err := context.factory.ActivityStream().Delete(originalActivityID); err != nil {
		return derp.Wrap(err, location, "Unable to delete original activity", originalActivity)
	}

	return nil
}
