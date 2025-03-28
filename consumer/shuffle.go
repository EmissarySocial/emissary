package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// Shuffle task "shuffles" all stream and user records in the database,
// by updating the "shuffle" field to a unique random value.
func Shuffle(factory *domain.Factory, _ mapof.Any) queue.Result {

	const location = "consumer.Recycle"

	log.Trace().Str("host", factory.Hostname()).Msg("Task: Shuffle")

	// Shuffle all Stream records
	if err := factory.Stream().Shuffle(); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error shuffling Stream records"))
	}

	// Shuffle all SearchResult records
	if err := factory.SearchResult().Shuffle(); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error shuffling SearchResult records"))
	}

	return queue.Success()
}
