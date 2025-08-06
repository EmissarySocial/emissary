package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// RecycleDomain deletes all records from the database that were
// "soft deleted" more than 30 days ago.
func RecycleDomain(factory *service.Factory, session data.Session, _ mapof.Any) queue.Result {

	/* DISABLING RECYCLE FOR NOW BECAUSE I'M TERRIFIED OF BREAKING PRODUCTION.

	const location = "consumer.Recycle"

	log.Trace().Str("host", factory.Hostname()).Msg("Task: Recycle")

	// Please try to enjoy each collection equally.
	for _, collection := range factory.Collections() {
		if err := queries.Recycle(factory.Session, collection); err != nil {
			return queue.Error(derp.Wrap(err, location, "Error recycling collection", collection))
		}
	}
	*/

	// Congratulatory affirmation.
	return queue.Success()
}
