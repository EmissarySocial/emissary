package activitypub_user

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// outbox_Wildcard handles any ActivityPub activity that doesn't have a specific handler
func init() {
	outboxRouter.Add(vocab.Any, vocab.Any, outbox_Wildcard)
}

// outbox_Wildcard accepts any unrecognized activity, and simply forwards it to the User's outbox without any further processing.
func outbox_Wildcard(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_Wildcard"

	// Put the activity into the User's outbox (which triggers delivery to all recipients)
	if err := putActivityIntoOutbox(context, activity); err != nil {
		return derp.Wrap(err, location, "Processing activity")
	}

	// Send response to caller
	return context.context.NoContent(http.StatusAccepted)
}

// putActivityIntoOutbox delivers an activity to every recipient it names
func putActivityIntoOutbox(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.putActivityIntoOutbox"

	// RULE: An Undo may only reverse the actor's own activities (ActivityPub §6.10)
	if err := requireUndoActorMatches(activity); err != nil {
		return derp.Wrap(err, location, "Validating Undo")
	}

	activityURL, err := context.factory.Outbox2().PutUserActivity(context.session, context.user.UserID, activity)

	if err != nil {
		return derp.Wrap(err, location, "Putting activity into outbox")
	}

	// Report the assigned id to the C2S client. The Location header stays reserved for the
	// created OBJECT's URL, so the ACTIVITY id travels in its own header. Clients use it to
	// seed inbox cursors with an id the server will actually serve back.
	context.context.Response().Header().Set("X-Activity-Id", activityURL)

	return nil
}

// requireUndoActorMatches returns an error when an Undo names an activity performed by another actor
func requireUndoActorMatches(activity streams.Document) error {

	const location = "handler.activitypub_user.requireUndoActorMatches"

	// Only an Undo reverses another activity
	if activity.Type() != vocab.ActivityTypeUndo {
		return nil
	}

	// RULE: Every embedded activity must name the same actor. A bare URL object, or one with no
	// actor, is the receiver's to resolve, so only a stated mismatch fails.
	actorID := activity.ActorID()

	for object := range activity.Object().Range() {

		// RULE: Never read a property of a bare URL, because Document.Get loads it from the network
		if object.IsString() {
			continue
		}

		for innerActorID := range object.Get(vocab.PropertyActor).RangeIDs() {
			if innerActorID != actorID {
				return derp.Forbidden(location, "Undo actor must match the actor of the undone activity", "actorID: "+innerActorID)
			}
		}
	}

	// You can only take back what's yours
	return nil
}
