package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
)

// ImportStartup begins an account migration: it loads the source Actor, tells the source server where
// the data is going, and queues an ImportItem for every document in the import plan.
func ImportStartup(factory *service.Factory, session data.Session, user *model.User, record *model.Import, args mapof.Any) queue.Result {

	const location = "consumer.ImportStartup"

	importService := factory.Import()

	// We'll need to authenticated using BEARER tokens (not HTTP signatures)
	// WithMinAge(0) waives the default cooldown on both loads below. The cache is keyed by URL alone,
	// so a cooldown here could answer an AUTHENTICATED request with the public copy that some earlier
	// unauthenticated load left behind. (BUG-104)
	client := factory.ActivityStream().AppClient()
	withBearerAuth := sherlock.WithRemoteOptions(options.BearerAuth(record.OAuthToken.AccessToken))
	withoutCooldown := ascache.WithMinAge(0)

	// Load the actor so we can make an import plan
	actor, err := client.Load(record.SourceID, sherlock.AsActor(), ascache.WithWriteOnly(), withBearerAuth, withoutCooldown)

	// We should have already loaded the actor when starting the Import process.
	// If we cannot load the actor now, then just abandon the whole damned thing.
	if err != nil {

		record.StateID = model.ImportStateImportError
		record.Message = "Unable to load ActivityPub Actor: " + err.Error()

		if inner := importService.Save(session, record, "Import Error"); inner != nil {
			return queue.Failure(derp.Wrap(inner, location, "Saving import failure", record))
		}

		return queue.Failure(derp.Wrap(err, location, "Loading ActivityPub actor", record.SourceID))
	}

	// Call the Actor's "startMigration" endpoint to tell the source server where we're importing the data to.
	if startEndpoint := actor.Endpoints().Get(vocab.EndpointStartMigration).String(); startEndpoint != "" {

		txn := remote.Post(startEndpoint).
			Form("actor", user.ActivityPubURL()).
			Form("oracle", factory.Host()+"/.imported").
			With(options.BearerAuth(record.OAuthToken.AccessToken))

		if err := txn.Send(); err != nil {
			return queue.Error(derp.Wrap(err, location, "Calling startMigration endpoint", startEndpoint))
		}
	}

	// Import plan contains all of the collections that we can import from this actor
	plan := importService.CalcImportPlan(actor)
	importItemService := factory.ImportItem()
	totalItems := 0

	// For each collection in the plan...
	for _, planItem := range plan {

		// Load the collection
		collection, err := client.Load(planItem.Href, ascache.WithWriteOnly(), withBearerAuth, withoutCooldown)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Loading import collection", planItem))
			continue
		}

		// Create an ImportItem for each same-origin document in the collection
		totalItems = totalItems + createImportItems(session, importItemService, record, actor, planItem.Value, collection)
	}

	// Update the Import record with new expectations
	record.TotalItems = totalItems
	record.CompleteItems = 0
	record.SourceURL = actor.ID()

	if err := importService.Save(session, record, "Updating item count"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Updating import record", record))
	}

	// Start a task (post-commit) to import all of the items for this source
	postcommit.Publish(session, factory.Queue(), "ImportItems", args)

	// Let's get this party started.
	return queue.Success()
}

// createImportItems creates one ImportItem for each same-origin document in a loaded
// collection and returns the number of items created.
func createImportItems(session data.Session, importItemService *service.ImportItem, record *model.Import, actor streams.Document, itemType string, collection streams.Document) int {

	const location = "consumer.createImportItems"

	count := 0

	// For each document in this collection...
	for document := range collections.RangeDocuments(collection) {

		// RULE: Only import documents hosted by the source account's own origin.
		// Migration collections can legitimately reference third-party hosts (e.g. a
		// following/blocked list), and a hostile actor can plant arbitrary URLs. Fetching
		// those would (a) forward the user's source-scoped OAuth Bearer token off-origin and
		// (b) save attacker-chosen content as the user's record. So we skip them entirely.
		if uri.NotSameOrigin(document.ID(), actor.ID()) {
			derp.Report(derp.Forbidden(location, "Skipping cross-origin import document", document.ID(), actor.ID()))
			continue
		}

		// Create a new ImportItem task to import this document
		importItem := model.NewImportItem()
		importItem.ImportID = record.ImportID
		importItem.UserID = record.UserID
		importItem.Type = itemType
		importItem.ImportURL = document.ID()
		importItem.StateID = model.ImportItemStateNew

		// Save the ImportItem to the task list
		if err := importItemService.Save(session, &importItem, "Created"); err != nil {
			derp.Report(derp.Wrap(err, location, "Creating import item"))
			continue
		}

		// Increment the counter of items created
		count = count + 1
	}

	return count
}
