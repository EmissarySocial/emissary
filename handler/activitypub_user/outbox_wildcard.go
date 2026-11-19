package activitypub_user

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers outbox_Wildcard for every activity that has no handler of its own
func init() {
	outboxRouter.Add(vocab.Any, vocab.Any, outbox_Wildcard)
}

// outbox_Wildcard stores an activity that no other handler claims in the User's outbox, and
// delivers it to the recipients it names. An activity that names no one is refused.
func outbox_Wildcard(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_Wildcard"

	// RULE: An activity that names no one would be stored and delivered to no one, so the client is told
	// instead. PutUserActivity does not check, because its other callers have already saved a record.
	if err := requireRecipients(activity); err != nil {
		return derp.Wrap(err, location, "Validating recipients")
	}

	// Put the activity into the User's outbox (which triggers delivery to all recipients)
	if err := putActivityIntoOutbox(context, activity); err != nil {
		return derp.Wrap(err, location, "Processing activity")
	}

	// Send response to caller
	return respondCreated(context, "")
}

// requireRecipients returns a BadRequest when an activity names no recipient in `to`, `cc`, `bto`
// or `bcc`, which are the only properties that delivery reads
func requireRecipients(activity streams.Document) error {

	const location = "handler.activitypub_user.requireRecipients"

	if len(activity.Recipients()) == 0 {
		return derp.BadRequest(location, "Activity must name at least one recipient in to, cc, bto, or bcc", "activityType: "+activity.Type())
	}

	return nil
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
