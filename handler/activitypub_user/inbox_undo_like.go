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

	// Parse the original Like/Dislike/Announce being undone. Our outgoing Undos EMBED the original
	// activity inline (D7), so this resolves WITHOUT a network fetch; a reference-only Undo from
	// another server may require a fetch, and if that fails we silently no-op (nothing to un-project).
	originalActivity, err := activity.Object().Load()

	if err != nil {
		return nil
	}

	// NOTE: no acceptance carve-out is needed here (unlike the add path). The actor-match guard below
	// ensures only the original reactor can undo, and removing an item that was never added (because
	// the original was dropped by D8) is a harmless no-op. See COLLECTIONS-REDESIGN.md D8.

	// RULE: ActivityPub type must match the received activity
	if activity.ActorID() != originalActivity.ActorID() {
		return derp.Unauthorized(location, "Actor undoing this activity must be the same as the original activity")
	}

	// Remove this Like/Dislike/Announce from the target Stream's response collection (if the target
	// is a local Stream). Keyed by the original activity's own ID, matching what the add path stored.
	if err := context.factory.Stream().RemoveResponseCollectionItem(context.session, originalActivity.Object().ID(), originalActivity.Type(), originalActivity.ID()); err != nil {
		return derp.Wrap(err, location, "Unable to remove response from collection", originalActivity.ID())
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
