package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// ImportItems is a queue consumer that processes individual ImportItem records for a given Import.
// When this task completes, it re-queues itself to process the next ImportItem in the database.
// If there are no more ImportItem records to process, then the Import is marked as "Complete".
func ImportItems(factory *service.Factory, session data.Session, user *model.User, importRecord *model.Import, args mapof.Any) queue.Result {

	const location = "consumer.ImportItems"

	importService := factory.Import()
	importItemService := factory.ImportItem()
	importItem := model.NewImportItem()

	// BUT FIRST... a helper function to make the rest of this task read easier.
	// HELPER that processes errors encountered on individual records
	closeTask := func(err error) queue.Result {

		if err == nil {
			importItem.StateID = model.ImportItemStateDone
			importItem.Message = "Import successful"

		} else {
			importItem.StateID = model.ImportItemStateError
			importItem.Message = derp.Message(err)
			derp.Report(derp.Wrap(err, location, "Error importing item", importItem))
		}

		// Update the ImportItem record in the database
		if err := importItemService.Save(session, &importItem, ""); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to update Import status message"))
		}

		// Increment the CompleteItems counter and save the Import
		importRecord.CompleteItems = importRecord.CompleteItems + 1

		if err := importService.Save(session, importRecord, "Increment counter"); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to increment item counter"))
		}

		// Requeue this task to locate the next importRecord in the chain
		return queue.Requeue(0)
	}

	// Okay. Let's do this.
	// Load the next importable item from the database
	err := importItemService.LoadNext(session, importRecord.UserID, importRecord.ImportID, &importItem)

	// If no "next" importRecord is found, then the import is complete
	if derp.IsNotFound(err) {
		if inner := importService.SetState(session, importRecord, model.ImportStateReviewing); inner != nil {
			return queue.Error(derp.Wrap(inner, location, "Unable to update import importRecord"))
		}
		return queue.Success()
	}

	// All other errors should be retried
	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to load next importable item"))
	}

	// -----------------------------------------------
	// Success! Let's process the next importable item

	// Update the display to show the URL that we're currently working on
	if err := importService.SetMessage(session, importRecord, importItem.ImportURL); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to update Import status message", importRecord))
	}

	// -----------------------------------------------
	// From here forward, all errors can be handled by the closeTask() helper function

	// Get the importable service that can handle this type of item
	locator := factory.ImportableLocator()
	importable, err := locator(importItem.Type)

	if err != nil {
		return closeTask(derp.Wrap(err, location, "Unrecognized collection type: "+importItem.Type))
	}

	// Retrieve the document to be imported from the remote server
	document := make([]byte, 0)
	txn := remote.Get(importItem.ImportURL).
		With(options.BearerAuth(importRecord.OAuthToken.AccessToken)).
		With(options.Debug()).
		Result(&document)

	if err := txn.Send(); err != nil {
		return closeTask(derp.Wrap(err, location, "Unable to retrieve document from source server"))
	}

	// Save the document to the local database
	if err := importable.Import(session, importRecord, &importItem, user, document); err != nil {
		return closeTask(err)
	}

	// Success! Increment the complete items counter and exit the task
	return closeTask(nil)
}
