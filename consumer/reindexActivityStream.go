package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// ReindexActivityStream refreshes a single cached ActivityStream document from its origin server.
func ReindexActivityStream(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.ReindexActivityStream"

	url := args.GetString("url")

	log.Debug().Str("loc", location).Str("url", url).Msg("Reindexing ActivityStream")
	activityService := factory.ActivityStream()

	// Try to load the ActivityStream. Skip the cache, and waive the default cooldown.
	if _, err := activityService.AppClient().Load(url, ascache.WithWriteOnly(), ascache.WithMinAge(0)); err != nil {

		// If there is any error, then remove the item from the cache
		if inner := activityService.Delete(url); inner != nil {
			return queue.Error(derp.Wrap(inner, location, "Deleting ActivityStream", url))
		}

		// If the ActivityStream no longer exists then this is a success
		if derp.IsNotFoundOrGone(err) {
			return queue.Success()
		}

		// Any other errors are failures. Do not retry.
		return queue.Failure(err)
	}

	// No error => success!
	return queue.Success()
}
