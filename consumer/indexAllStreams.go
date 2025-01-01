package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

func IndexAllStreams(factory *domain.Factory, args mapof.Any) queue.Result {

	const location = "consumer.IndexAllStreams"

	searchService := factory.Search()
	streamService := factory.Stream()

	streams, err := streamService.RangePublished()

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error retrieving Streams"))
	}

	for stream := range streams {

		searchResult, ok := streamService.SearchResult(&stream)

		if !ok {
			continue
		}

		if err := searchService.Upsert(searchResult); err != nil {
			derp.Report(derp.Wrap(err, location, "Error saving SearchResult"))
		}
	}

	return queue.Success()
}
