package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/turbine/queue"
)

func IndexAllStreams(factory *domain.Factory, args mapof.Any) queue.Result {

	const location = "consumer.IndexAllStreams"

	// Collect required services
	searchService := factory.SearchResult()
	streamService := factory.Stream()

	// Get a RangeFunc containing all Streams in the database
	streams, err := streamService.RangePublished()

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error retrieving Streams"))
	}

	// Index each Stream in the range
	for stream := range streams {

		// Recompute Hashtags
		originalHashtags := stream.Hashtags
		streamService.CalculateTags(&stream)

		// If necessary, re-save the Stream
		if !slice.Equal(stream.Hashtags, originalHashtags) {
			if err := streamService.Save(&stream, "Updating Hashtags"); err != nil {
				derp.Report(derp.Wrap(err, location, "Error saving Stream"))
			}
		}

		// Create a new SearchResult from the (updated?) Stream
		searchResult := streamService.SearchResult(&stream)

		if err := searchService.Sync(searchResult); err != nil {
			derp.Report(derp.Wrap(err, location, "Error saving SearchResult"))
		}
	}

	return queue.Success()
}
