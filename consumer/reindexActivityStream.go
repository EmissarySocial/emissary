package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/EmissarySocial/emissary/tools/ascrawler"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

func ReindexActivityStream(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.ReindexActivityStream"

	url := args.GetString("url")

	log.Debug().Str("loc", location).Str("url", url).Msg("Reindexing ActivityStream")
	activityService := factory.ActivityStream()

	// Try to load the ActivityStream. Skip the cache, and to not re-trigger the crawler.
	if _, err := activityService.AppClient().Load(url, ascache.WithWriteOnly(), ascrawler.WithoutCrawler()); err != nil {

		// If the ActivityStream no longer exists, then remove it from the cache
		if derp.IsNotFoundOrGone(err) {
			if inner := activityService.Delete(url); inner != nil {
				return queue.Error(derp.Wrap(inner, location, "Unable to delete ActivityStream", url))
			}
			return queue.Success()
		}

		// Retry HTTP 429 (Too Many Requests) errors
		if tooManyRequests, retryDuration := derp.IsTooManyRequests(err); tooManyRequests {
			return queue.Requeue(retryDuration)
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
