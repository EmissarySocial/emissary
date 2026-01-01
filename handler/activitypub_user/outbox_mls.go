package activitypub_user

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeGroupInfo, outbox_MLS)
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypePrivateMessage, outbox_MLS)
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypePublicMessage, outbox_MLS)
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeWelcome, outbox_MLS)
}

// Create an mls:Welcome via the ActivityPub API
func outbox_MLS(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_MLS"

	// Save the activity in the user's outbox
	if err := context.factory.Outbox2().AddUserActivity(context.session, context.user.UserID, activity.Map()); err != nil {
		return derp.Wrap(err, location, "Unable to save outbox activity")
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
