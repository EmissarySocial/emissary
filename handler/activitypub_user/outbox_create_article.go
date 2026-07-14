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
	object.Context = document.Context()
	object.UserID = context.user.UserID
	object.Permissions = document.Recipients()
	object.Value = document.Map()

	// Save the new Object to the database
	if err := objectService.Save(context.session, &object, "Created via ActivityPub Outbox"); err != nil {
		return derp.Wrap(err, location, "Unable to save object", object)
	}

	// Put the activity into the User's outbox (which triggers delivery to all recipients)
	if err := putActivityIntoOutbox(context, activity); err != nil {
		return derp.Wrap(err, location, "Unable to process activity")
	}

	// Write the response to the client
	context.context.Response().Header().Set("Location", objectService.ActivityPubURL(object.UserID, object.ObjectID))
	return context.context.NoContent(http.StatusCreated)
}
