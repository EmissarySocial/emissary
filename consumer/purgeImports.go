package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// PurgeImports removes import records that have been kept past their "purgeDate"
// (which is set to 6 months after the import is completed).
func PurgeImports(factory *service.Factory, session data.Session, _ mapof.Any) queue.Result {

	const location = "consumer.PurgeImports"

	log.Trace().Msg("Task: PurgeImports")

	// Collect required services
	importService := factory.Import()
	importItemService := factory.ImportItem()

	// Query all imports that are past their purge date
	purgableImports := importService.RangePurgable(session)

	for record := range purgableImports {

		// Hard Delete import items
		if err := importItemService.DeleteByImportID(session, record.UserID, record.ImportID); err != nil {
			return queue.Error(derp.Wrap(err, location, "Unable to delete ImportItems", record))
		}

		// Delete the Import record
		if err := importService.HardDeleteByID(session, record.UserID, record.ImportID); err != nil {
			return queue.Error(derp.Wrap(err, location, "Unable to delete Import", record))
		}
	}

	// Glorious success
	return queue.Success()
}
