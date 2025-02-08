package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

func Geocode(factory *domain.Factory, streamService *service.Stream, stream *model.Stream, args mapof.Any) queue.Result {

	const location = "consumer.Geocode"

	// Try to geocode the Places in this Stream
	geocodeService := factory.Geocode()

	if err := geocodeService.Geocode(stream); err != nil {
		return queue.Error(derp.Wrap(err, location, "Cannot geocode stream", stream))
	}

	// Try to save the Stream
	if err := streamService.Save(stream, "Updated Geocodes"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Cannot save stream", stream))
	}

	// Yuss!
	return queue.Success()
}
