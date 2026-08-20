package activitypub_user

import (
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the handlers that undo Like, Dislike, and Announce activities
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

	// Read the original Like/Dislike/Announce being undone. Use LoadLink, NOT Load: LoadLink returns
	// an INLINE object as-is (no fetch) but dereferences a bare-URL reference. This matters both ways:
	//   - OUR Undos embed the original activity inline (D7). Load() would still HTTP-fetch object.id,
	//     but that /pub/liked/<id> Response was already hard-deleted by the time the Undo loops back,
	//     so the fetch 404s. LoadLink sees the inline map and skips the fetch entirely.
	//   - OTHER servers (e.g. Mastodon) send Undo with object as a bare URL reference. LoadLink
	//     fetches those so we can still resolve and un-project them.
	originalActivity := activity.Object().LoadLink()

	// If we still can't resolve the original activity (bare reference that no longer exists, or a
	// fetch failure), there's nothing to un-project. A missing type is the tell. No-op.
	if originalActivity.Type() == "" {
		return nil
	}

	// NOTE: no acceptance carve-out is needed here (unlike the add path). The actor-match guard below
	// ensures only the original reactor can undo, and removing an item that was never added (because
	// the original was dropped by D8) is a harmless no-op. See COLLECTIONS-REDESIGN.md D8.

	// RULE: ActivityPub type must match the received activity
	if activity.ActorID() != originalActivity.ActorID() {
		return derp.Unauthorized(location, "Actor undoing this activity must be the same as the original activity")
	}

	// RULE: The undone activity must share the actor's origin (D19). originalActivity is resolved from a
	// link the sender controls, so a bare-URL id could name a victim's activity; the actor-match guard
	// above is satisfied trivially by attacker-supplied content. Binding the id host to the verified
	// actor's host stops an attacker from evicting arbitrary cache entries by naming someone else's URL.
	if originalActivityID := originalActivity.ID(); originalActivityID != "" {
		if !activitypub.IsSameOrigin(activity.ActorID(), originalActivityID) {
			return derp.Forbidden(location, "Undone activity must share the same origin as the actor", activity.ActorID(), originalActivityID)
		}
	}

	// Remove this Like/Dislike/Announce from the target Stream's response collection (if the target
	// is a local Stream). Keyed by the original activity's own ID, matching what the add path stored.
	if err := context.factory.Stream().RemoveResponseCollectionItem(context.session, originalActivity.Object().ID(), originalActivity.Type(), originalActivity.ID()); err != nil {
		return derp.Wrap(err, location, "Removing response from collection", originalActivity.ID())
	}

	// Get/Generate the ID of the original activity
	originalActivityID := originalActivity.ID()

	if originalActivityID == "" {
		originalActivityID = activitypub.FakeActivityID(originalActivity)
	}

	// Remove the original activity from the database.
	if err := context.factory.ActivityStream().Delete(originalActivityID); err != nil {
		return derp.Wrap(err, location, "Deleting original activity", originalActivity)
	}

	return nil
}
