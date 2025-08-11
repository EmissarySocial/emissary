package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascrawler"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CrawlActivityStreams(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.CrawlActivityStreams"

	url := args.GetString("url")
	depth := args.GetInt("depth")

	log.Debug().Str("loc", location).Str("url", url).Int("depth", depth).Msg("Loading ActivityStream")

	activityService := factory.ActivityStream(model.ActorTypeApplication, primitive.NilObjectID)

	if _, err := activityService.Client().Load(url, ascrawler.AtDepth(depth)); err != nil {

		if derp.IsClientError(err) {
			return queue.Failure(derp.Wrap(err, location, "Client error when loading ActivityStream"))
		}

		return queue.Error(derp.Wrap(err, location, "Unable to load ActivityStream"))
	}

	// Otherwise, success!
	return queue.Success()
}
