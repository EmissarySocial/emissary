package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the handler for outbound Update/Article activities
func init() {
	outboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeArticle, outbox_UpdateArticle)
}

// outbox_UpdateArticle publishes a revised Article from the User's outbox
func outbox_UpdateArticle(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_UpdateArticle"

	document := activity.Object()

	// Parse the UserID and ObjectID from the Document's URL
	locatorService := context.factory.Locator()
	userID, objectID, err := locatorService.ParseObject(document.ID())

	if err != nil {
		return derp.Wrap(err, location, "Parsing object ID", document.ID())
	}

	// Verify that the UserID matches the current user
	if userID != context.user.UserID {
		return derp.Forbidden(location, "Current User must own the Object being Updated", "objectID: "+document.ID())
	}

	// Load the existing Object from the database
	objectService := context.factory.Object()
	object := model.NewObject()

	if err := objectService.LoadByID(context.session, userID, objectID, &object); err != nil {
		return derp.Wrap(err, location, "Loading object", "objectID", objectID)
	}

	// Update the Object's value
	object.Value = document.Map()
	object.Permissions = document.Recipients()

	// Save the updated Object to the database
	if err := objectService.Save(context.session, &object, "Updated via ActivityPub Outbox"); err != nil {
		return derp.Wrap(err, location, "Saving object", object)
	}

	// Put the activity into the User's outbox (which triggers delivery to all recipients)
	if err := putActivityIntoOutbox(context, activity); err != nil {
		return derp.Wrap(err, location, "Processing activity")
	}

	// Send response to caller
	return context.context.NoContent(http.StatusAccepted)
}
