package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ImportStartup(factory *service.Factory, session data.Session, user *model.User, record *model.Import, args mapof.Any) queue.Result {

	const location = "consumer.ImportStartup"

	importService := factory.Import()

	// We'll need to authenticated using BEARER tokens (not HTTP signatures)
	activityService := factory.ActivityStream(model.ActorTypeApplication, primitive.NilObjectID)
	withBearerAuth := sherlock.WithRemoteOptions(options.BearerAuth(record.OAuthToken.AccessToken))
	client := activityService.Client()

	// Load the actor so we can make an import plan
	actor, err := client.Load(record.SourceID, sherlock.AsActor(), ascache.WithForceReload(), withBearerAuth)

	// We should have already loaded the actor when starting the Import process.
	// If we cannot load the actor now, then just abandon the whole damned thing.
	if err != nil {

		record.StateID = model.ImportStateImportError
		record.Message = "Unable to load ActivityPub Actor: " + err.Error()

		if err := importService.Save(session, record, "Import Error"); err != nil {
			return queue.Failure(derp.Wrap(err, location, "Unable to save import failure", record))
		}

		return queue.Failure(derp.Wrap(err, location, "Unable to load ActivityPub actor", record.SourceID))
	}

	// Call the Actor's "startMigration" endpoint to tell the source server where we're importing the data to.
	if startEndpoint := actor.Endpoints().Get(vocab.EndpointStartMigration).String(); startEndpoint != "" {

		txn := remote.Post(startEndpoint).
			Form("actor", user.ActivityPubURL()).
			Form("oracle", factory.Host()+"/.imported").
			With(options.BearerAuth(record.OAuthToken.AccessToken))

		if err := txn.Send(); err != nil {
			return queue.Error(derp.Wrap(err, location, "Unable to call startMigration endpoint", startEndpoint))
		}
	}

	// Import plan contains all of the collections that we can import from this actor
	plan := importService.CalcImportPlan(actor)
	importItemService := factory.ImportItem()
	totalItems := 0

	// For each collection in the plan...
	for _, planItem := range plan {

		// Load the collection
		collection, err := client.Load(planItem.Href, ascache.WithForceReload(), withBearerAuth)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to load import collection", planItem))
			continue
		}

		// For each document in this collection...
		for document := range collections.RangeDocuments(collection) {

			// Create a new ImportItem task to import this document
			importItem := model.NewImportItem()
			importItem.ImportID = record.ImportID
			importItem.UserID = record.UserID
			importItem.Type = planItem.Value
			importItem.ImportURL = document.ID()
			importItem.StateID = model.ImportItemStateNew

			// Save the ImportItem to the task list
			if err := importItemService.Save(session, &importItem, "Created"); err != nil {
				derp.Report(derp.Wrap(err, location, "Unable to create import item"))
				continue
			}

			// Increment the TotalItems counter
			totalItems = totalItems + 1
		}
	}

	// Update the Import record with new expectations
	record.TotalItems = totalItems
	record.CompleteItems = 0
	record.SourceURL = actor.ID()

	if err := importService.Save(session, record, "Updating item count"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to update import record", record))
	}

	// Start a task to import all of the items for this source
	factory.Queue().NewTask("ImportItems", args)

	// Let's get this party started.
	return queue.Success()
}
