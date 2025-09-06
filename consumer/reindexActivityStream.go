package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ReindexActivityStream(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.ReindexActivityStream"

	url := args.GetString("url")

	log.Debug().Str("loc", location).Str("url", url).Msg("Reindexing ActivityStream")
	activityService := factory.ActivityStream(model.ActorTypeApplication, primitive.NilObjectID)

	// Configure crawler options to persist depth and history
	if _, err := activityService.Client().Load(url, ascache.WithForceReload()); err != nil {

		// If the ActivityStream no longer exists, then remove it from the cache
		if shouldDeleteActivityStream(err) {
			activityStreamService := factory.ActivityStream(model.ActorTypeApplication, primitive.NilObjectID)
			if err := activityStreamService.Delete(url); err != nil {
				return queue.Error(derp.Wrap(err, location, "Unable to deleting ActivityStream", url))
			}
		}

		if derp.IsClientError(err) {
			return queue.Failure(derp.Wrap(err, location, "Client error when loading ActivityStream"))
		}

		return queue.Error(derp.Wrap(err, location, "Unable to load ActivityStream"))
	}

	// Otherwise, success!
	return queue.Success()
}
