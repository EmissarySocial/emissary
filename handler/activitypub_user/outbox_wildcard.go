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
