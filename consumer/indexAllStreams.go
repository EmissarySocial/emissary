package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

func IndexAllStreams(factory *domain.Factory, args mapof.Any) queue.Result {

	const location = "consumer.IndexAllStreams"

	streamService := factory.Stream()

	allStreams, err := streamService.RangeAll()

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error retrieving Streams"))
	}

	for summary := range allStreams {

		log.Debug().Str("url", summary.URL).Msg("Indexing Stream")
		transaction := remote.Post(summary.URL + "/search-index")

		if err := transaction.Send(); err != nil {
			if !derp.IsClientError(err) {
				return queue.Error(derp.Wrap(err, location, "Error sending request"))
			}
		}
	}

	return queue.Success()
}
