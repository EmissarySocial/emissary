package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeArticle, outbox_CreateArticle)
}

func outbox_CreateArticle(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_CreateArticle"

	// Find the ActivityStream's Document
	document := activity.Object()

	// Create a new Object using the incoming ActivityStream Document
	objectService := context.factory.Object()
	object := model.NewObject()

	// RULE: The server assigns the created object's canonical id, replacing any client-
	// provided value (ActivityPub §6). Recipients key incoming messages by this id, so it
	// must travel inside the delivered activity — not just the Location header.
	objectURL := objectService.ActivityPubURL(context.user.UserID, object.ObjectID)
	objectValue := document.Map()
	objectValue[vocab.PropertyID] = objectURL
	activity.SetProperty(vocab.PropertyObject, objectValue)

	object.Context = document.Context()
	object.UserID = context.user.UserID
	object.Permissions = document.Recipients()
	object.Value = objectValue

	// Save the new Object to the database
	if err := objectService.Save(context.session, &object, "Created via ActivityPub Outbox"); err != nil {
		return derp.Wrap(err, location, "Saving object", object)
	}

	// Put the activity into the User's outbox (which triggers delivery to all recipients)
	if err := putActivityIntoOutbox(context, activity); err != nil {
		return derp.Wrap(err, location, "Processing activity")
	}

	// Write the response to the client
	context.context.Response().Header().Set("Location", objectURL)
	return context.context.NoContent(http.StatusCreated)
}
