package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// Shuffle task "shuffles" all stream and user records in the database,
// by updating the "shuffle" field to a unique random value.
func Shuffle(factory *service.Factory, session data.Session, _ mapof.Any) queue.Result {

	const location = "consumer.Shuffle"

	log.Trace().Str("host", factory.Hostname()).Msg("Task: Shuffle")

	// Shuffle all Stream records
	if err := factory.Stream().Shuffle(session); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to shuffle Stream records"))
	}

	// Shuffle all SearchResult records
	if err := factory.SearchResult().Shuffle(session); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to shuffle SearchResult records"))
	}

	return queue.Success()
}
