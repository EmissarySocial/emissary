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

func ImportItems(factory *service.Factory, session data.Session, user *model.User, record *model.Import, args mapof.Any) queue.Result {

	const location = "consumer.ImportItems"

	importService := factory.Import()
	importItemService := factory.ImportItem()
	importItem := model.NewImportItem()

	// BUT FIRST... a helper function to make the rest of this task read easier.
	// HELPER that processes errors encountered on individual records
	closeTask := func(stateID string, message string) queue.Result {

		// Update the ImportItem and save it to the database
		importItem.StateID = stateID
		importItem.Message = message

		if err := importItemService.Save(session, &importItem, ""); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to update Import status message"))
		}

		// Increment the CompleteItems counter and save the Import
		record.CompleteItems = record.CompleteItems + 1

		if err := importService.Save(session, record, "Increment counter"); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to increment item counter"))
		}

		// Requeue this task to locate the next record in the chain
		return queue.Requeue(0)
	}

	// Okay. Let's do this.
	// Load the next importable item from the database
	err := importItemService.LoadNext(session, record.UserID, record.ImportID, &importItem)

	// If no "next" record is found, then the import is complete
	if derp.IsNotFound(err) {
		if err := importService.SetState(session, record, model.ImportStateReviewing); err != nil {
			return queue.Error(derp.Wrap(err, location, "Unable to update import record"))
		}
		return queue.Success()
	}

	// All other errors should be retried
	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to load next importable item"))
	}

	// Success! Let's process the next importable item

	// Update the display to show the URL that we're currently working on
	if err := importService.SetMessage(session, record, importItem.URL); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to update Import status message", record))
	}

	// Get the importable service that can handle this type of item
	locator := factory.ImportableLocator()
	importable, err := locator(importItem.Type)

	if err != nil {
		return closeTask(model.ImportItemStateError, "Unrecognized collection type: "+importItem.Type)
	}

	// Retrieve the document to be imported from the remote server
	document := make([]byte, 0)
	txn := remote.Get(importItem.URL).
		With(options.BearerAuth(record.OAuthToken.AccessToken)).
		With(options.Debug()).
		Result(&document)

	if err := txn.Send(); err != nil {
		return closeTask(model.ImportItemStateError, "Unable to retrieve document from source server")
	}

	// Save the document to the local database
	if err := importable.Import(session, &importItem, user, document); err != nil {
		return closeTask(model.ImportItemStateError, "Unable to process document: "+err.Error())
	}

	// Success! Increment the complete items counter and exit the task
	return closeTask(model.ImportItemStateDone, "Processed Successfully")
}
