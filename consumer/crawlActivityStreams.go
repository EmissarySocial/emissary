package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascrawler"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CrawlActivityStreams(factory *service.Factory, _ data.Session, args mapof.Any) queue.Result {

	const location = "consumer.CrawlActivityStreams"

	history := args.GetSliceOfString("history")
	url := args.GetString("url")
	actorType := args.GetString("actorType")
	actorToken := args.GetString("actorID")
	actorID, err := primitive.ObjectIDFromHex(actorToken)

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid actorID"))
	}

	log.Debug().Str("loc", location).Str("url", url).Int("depth", len(history)).Msg("Crawling ActivityStream")
	activityService := factory.ActivityStream(actorType, actorID)

	// Configure crawler options to persist depth and history
	if _, err := activityService.Client().Load(url, ascrawler.WithHistory(history...)); err != nil {

		if derp.IsClientError(err) {
			return queue.Failure(derp.Wrap(err, location, "Client error when loading ActivityStream", history))
		}

		return queue.Error(derp.Wrap(err, location, "Unable to load ActivityStream"))
	}

	// Otherwise, success!
	return queue.Success()
}
