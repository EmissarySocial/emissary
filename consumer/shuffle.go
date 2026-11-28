package consumer

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// shuffleTimeout bounds one Domain's shuffle, which is a single statement over a whole collection
const shuffleTimeout = 5 * time.Minute

// Shuffle gives every SearchResult on a Domain a new random "shuffle" value, which is the
// order that shuffled search pages are drawn in
func Shuffle(factory *service.Factory, _ mapof.Any) queue.Result {

	const location = "consumer.Shuffle"

	log.Trace().Str("host", factory.Hostname()).Msg("Task: Shuffle")

	// RULE: No transaction.  One would hold every write uncommitted until the last (see AGENTS.md)
	ctx, cancel := context.WithTimeout(context.Background(), shuffleTimeout)
	defer cancel()

	session, err := factory.Server().Session(ctx)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Opening database session", factory.Hostname()))
	}

	defer session.Close()

	// Shuffle all SearchResult records
	if err := factory.SearchResult().Shuffle(session); err != nil {
		return queue.Error(derp.Wrap(err, location, "Shuffling SearchResult records", factory.Hostname()))
	}

	return queue.Success()
}
