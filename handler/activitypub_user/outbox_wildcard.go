package activitypub_user

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.Any, send_Wildcard)
}

// send_Wildcard adds a new activity to the User's outbox without any additional processing.
// This activity will not show up in the user's profile web page because we don't understand what it is.
func send_Wildcard(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.send_Wildcard"

	// Save the activity in the user's outbox
	if err := context.factory.Outbox2().AddUserActivity(context.session, context.user.UserID, activity); err != nil {
		return derp.Wrap(err, location, "Unable to save activity")
	}

	// Get a service for the "Locator" interface
	sendLocator := context.factory.SendLocator(context.session)

	// Send ActivityPub notifications to participants
	sender := sender.New(sendLocator, context.factory.Queue())

	if err := sender.Send(activity.Map()); err != nil {
		return derp.Wrap(err, location, "Unable to send activity")
	}

	// Done.
	return context.context.NoContent(http.StatusOK)
}
