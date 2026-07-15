package consumer

import (
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// RecycleDomain permanently deletes all records from this Domain's database that were
// "soft deleted" more than 30 days ago.
func RecycleDomain(factory *service.Factory, session data.Session, _ mapof.Any) queue.Result {

	const location = "consumer.RecycleDomain"

	log.Trace().Str("hostname", factory.Hostname()).Msg("Task: RecycleDomain")

	// RULE: Report and continue, rather than returning on the first error (as sibling purge tasks
	// do).  These collections are independent, so a single collection that fails every run -- a
	// query timeout, say -- must not permanently starve every collection after it in the list.
	errorCount := 0

	// Please try to enjoy each collection equally.
	for _, collection := range factory.Collections() {
		if err := queries.Recycle(session, collection); err != nil {
			derp.Report(derp.Wrap(err, location, "Error recycling collection", collection))
			errorCount = errorCount + 1
		}
	}

	// Every collection has had its turn.  Surface a failure now so the task is retried.
	if errorCount > 0 {
		return queue.Error(derp.Internal(location, "Unable to recycle one or more collections", errorCount))
	}

	// Congratulatory affirmation.
	return queue.Success()
}
