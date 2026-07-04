package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// AddToCollection adds a message to a collection / reply chain managed by this server.
func AddToCollection(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.AddToCollection"

	// Collect parameters
	objectID := args.GetString("url")

	// Parse the Collection from the URL
	locatorService := factory.Locator()
	userID, collectionID, err := locatorService.ParseCollection(objectID)

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Unable to parse collection", "url: "+objectID))
	}

	// Load the document being added to the collection
	activityService := factory.ActivityStream()
	client := activityService.UserClient(userID)
	document, err := client.Load(objectID)

	if err != nil {
		return requeue(derp.Wrap(err, location, "Unable to load document", "url: "+objectID))
	}

	// Create a new CollectionItem record
	collectionItem := model.NewCollectionItem()
	collectionItem.CollectionID = collectionID
	collectionItem.UserID = userID
	collectionItem.InReplyTo = document.InReplyTo().ID()

	// Save the unique ObjectLink record to the database
	if err := factory.CollectionItem().SaveUnique(session, &collectionItem, "Created"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to save collection item", "collectionID: "+collectionID.Hex(), "userID: "+userID.Hex(), "objectID: "+objectID))
	}

	// Woot.
	return queue.Success()
}
