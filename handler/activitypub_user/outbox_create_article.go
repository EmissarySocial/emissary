package activitypub_user

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the handler for outbound Create/Article activities
func init() {
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeArticle, outbox_CreateArticle)
}

// outbox_CreateArticle publishes a new Article (or Note) from the User's outbox, readable only by its recipients
func outbox_CreateArticle(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_CreateArticle"

	// Save the Object and put the activity into the User's outbox (which delivers it to every recipient)
	objectURL, activityURL, err := context.factory.Object().CreateAddressed(context.session, context.user.UserID, activity)

	if err != nil {
		return derp.Wrap(err, location, "Creating addressed object")
	}

	// Write the response to the client: the new object's URL, and the activity's in its own header
	context.context.Response().Header().Set("X-Activity-Id", activityURL)
	context.context.Response().Header().Set("Location", objectURL)
	return context.context.NoContent(http.StatusCreated)
}
