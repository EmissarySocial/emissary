package activitypub_user

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeNote, send_CreateNote)
}

// send_CreateNote creates a new note (now, just a private message) in the user's outbox
func send_CreateNote(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.send_CreateNote"

	// For now, we don't support public notes, so return an error
	// In the future, we'll add more rules that map public-facing posts to Streams.
	if activity.Recipients().Contains(vocab.NamespaceASPublic) {
		return derp.NotImplemented(location, "Public notes are not supported at this time.")
	}

	// Save the activity in the user's outbox
	if err := context.factory.Outbox2().AddUserActivity(context.session, context.user.UserID, activity); err != nil {
		return derp.Wrap(err, location, "Unable to save outbox activity")
	}

	// Done.
	return context.context.NoContent(http.StatusOK)
}
