package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
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

func putActivityIntoOutbox(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.putActivityIntoOutbox"

	// Calculate all recipients of this Activity
	recipients := activity.Recipients()

	// For now, we don't support public notes, so return an error
	// In the future, we'll add more rules that map public-facing posts to Streams.
	if activity.IsPublic() {
		return derp.NotImplemented(location, "Public notes are not supported at this time.")
	}

	// Collect services
	factory := context.factory
	locatorService := factory.Locator()
	outbox2Service := factory.Outbox2()

	// Confirm that the actor matches the authenticated user
	userID, err := locatorService.ParseUser(activity.ActorID())

	if err != nil {
		return derp.Wrap(err, location, "Parsing userID from actorID", "actorID: "+activity.ActorID(), activity.Map())
	}

	if userID != context.user.UserID {
		return derp.Forbidden(location, "Actor does not match authenticated user", "actorID: "+activity.ActorID(), activity.Map())
	}

	// Add an activity record to the Outbox2
	outboxItem := model.NewOutboxItem()
	outboxItem.URL = locatorService.ActivityURL(model.ActorTypeUser, context.user.UserID, outboxItem.ActivityID)

	// RULE: The server assigns the activity's canonical id, replacing any client-provided
	// value (ActivityPub §6.1). Client-generated ids (e.g. "urn:uuid:...") are not
	// dereferenceable URLs, and remote servers reject activities that carry them.
	activity.SetProperty(vocab.PropertyID, outboxItem.URL)

	// Report the assigned id to the C2S client. The Location header stays reserved for the
	// created OBJECT's URL, so the ACTIVITY id travels in its own header. Clients use it to
	// seed inbox cursors with an id the server will actually serve back.
	context.context.Response().Header().Set("X-Activity-Id", outboxItem.URL)

	outboxItem.ActorType = model.ActorTypeUser
	outboxItem.ActorID = context.user.UserID
	outboxItem.Activity = activity.Map()
	outboxItem.Recipients = recipients

	// Save the activity in the user's outbox
	if err := outbox2Service.Save(context.session, &outboxItem, "Created via ActivityPub Outbox2"); err != nil {
		return derp.Wrap(err, location, "Saving Outbox2 activity")
	}

	return nil
}
