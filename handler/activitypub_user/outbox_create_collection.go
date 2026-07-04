package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.CoreTypeOrderedCollection, outbox_CreateOrderedCollection)
}

func outbox_CreateOrderedCollection(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_CreateOrderedCollection"

	// Find the ActivityStream's Document
	document := activity.Object()

	// Create a new Object using the incoming ActivityStream Document
	collectionService := context.factory.Collection()
	collection := model.NewCollection()
	collection.UserID = context.user.UserID
	collection.To = document.To().SliceOfString()
	collection.Cc = document.CC().SliceOfString()

	// Save the new Object to the database
	if err := collectionService.Save(context.session, &collection, "Created via ActivityPub Outbox"); err != nil {
		return derp.Wrap(err, location, "Unable to save collection", collection)
	}

	// RULE: Unlike other Create handlers (Article, Note, KeyPackage), a Collection is NOT
	// pushed through the outbox via putActivityIntoOutbox.  A Collection is a pull resource:
	// it is served from GetCollection at /pub/collections/{id} and read access is gated by
	// its To/Cc participant lists (see Collection.HasParticipant / canViewCollection).
	// Participants fetch it on demand; it is not federated as a standalone activity.

	// Write the response to the client
	context.context.Response().Header().Set("Location", collectionService.ActivityPubURL(collection.UserID, collection.CollectionID))
	return context.context.NoContent(http.StatusCreated)
}
