package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/EmissarySocial/emissary/tools/ascrawler"
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

	// Try to load the ActivityStream. Skip the cache, and to not re-trigger the crawler.
	if _, err := activityService.Client().Load(url, ascache.WithForceReload(), ascrawler.WithoutCrawler()); err != nil {

		// If the ActivityStream no longer exists, then remove it from the cache
		if derp.IsNotFoundOrGone(err) {
			if err := activityService.Delete(url); err != nil {
				return queue.Error(derp.Wrap(err, location, "Unable to delete ActivityStream", url))
			}
			return queue.Success()
		}

		// If it's "our fault" then we can't retry
		if derp.IsClientError(err) {
			return queue.Failure(derp.Wrap(err, location, "Unable to load ActivityStream. No retry", url))
		}

		// Otherwise, it's "their fault" and it's worth retrying
		return queue.Error(derp.Wrap(err, location, "Unable to load ActivityStream. Will retry later", url))
	}

	// No error => success!
	return queue.Success()
}
