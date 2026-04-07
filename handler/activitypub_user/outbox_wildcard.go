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
func outbox_Wildcard(context Context, document streams.Document) error {

	const location = "handler.activitypub_user.outbox_Wildcard"

	// Calculate all recipients of this Activity
	recipients := document.Recipients()

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
	userID, err := locatorService.ParseUser(document.Actor().ID())

	if err != nil {
		return derp.Wrap(err, location, "Unable to parse userID from actorID", "actorID: "+document.Actor().ID())
	}

	if userID != context.user.UserID {
		return derp.Forbidden(location, "Actor does not match authenticated user", "actorID: "+document.Actor().ID())
	}

	// Add an activity record to the Outbox2
	activity := model.NewActivity()
	activity.URL = locatorService.ActivityURL(model.ActorTypeUser, context.user.UserID, activity.ActivityID)
	activity.ActorType = model.ActorTypeUser
	activity.ActorID = context.user.UserID
	activity.Object = document.Map()
	activity.Recipients = recipients

	// Save the activity in the user's outbox
	if err := outbox2Service.Save(context.session, &activity, "Created via ActivityPub Outbox2"); err != nil {
		return derp.Wrap(err, location, "Unable to save Outbox2 activity")
	}

	// Done.
	return context.context.NoContent(http.StatusAccepted)
}
