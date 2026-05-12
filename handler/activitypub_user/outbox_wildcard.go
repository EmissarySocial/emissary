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
		return derp.Wrap(err, location, "Unable to process activity")
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
	if recipients.Contains(vocab.NamespaceASPublic) {
		return derp.NotImplemented(location, "Public notes are not supported at this time.")
	}

	// Collect services
	factory := context.factory
	locatorService := factory.Locator()
	outbox2Service := factory.Outbox2()

	// Confirm that the actor matches the authenticated user
	userID, err := locatorService.ParseUser(activity.Actor().ID())

	if err != nil {
		return derp.Wrap(err, location, "Unable to parse userID from actorID", "actorID: "+activity.Actor().ID(), activity.Map())
	}

	if userID != context.user.UserID {
		return derp.Forbidden(location, "Actor does not match authenticated user", "actorID: "+activity.Actor().ID(), activity.Map())
	}

	// Add an activity record to the Outbox2
	dbActivity := model.NewActivity()
	dbActivity.URL = locatorService.ActivityURL(model.ActorTypeUser, context.user.UserID, dbActivity.ActivityID)
	dbActivity.ActorType = model.ActorTypeUser
	dbActivity.ActorID = context.user.UserID
	dbActivity.Object = activity.Map()
	dbActivity.Recipients = recipients

	// Save the activity in the user's outbox
	if err := outbox2Service.Save(context.session, &dbActivity, "Created via ActivityPub Outbox2"); err != nil {
		return derp.Wrap(err, location, "Unable to save Outbox2 activity")
	}

	return nil
}
