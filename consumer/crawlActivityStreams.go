package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CrawlActivityStreams(factory *service.Factory, _ data.Session, args mapof.Any) queue.Result {

	const location = "consumer.CrawlActivityStreams"

	// Collect parameters
	actorType := args.GetString("actorType")
	actorToken := args.GetString("actorID")
	url := args.GetString("url")

	// Parse actorID
	actorID, err := primitive.ObjectIDFromHex(actorToken)

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid actorID"))
	}

	// Try to crawl the ActivityStream
	activityStreamCrawler := factory.ActivityStreamCrawler(actorType, actorID)

	if err := activityStreamCrawler.Crawl(url); err != nil {

		// If the ActivityStream no longer exists, then remove it from the cache
		if derp.IsNotFoundOrGone(err) {
			activityService := factory.ActivityStream(actorType, actorID)
			if err := activityService.Delete(url); err != nil {
				return queue.Error(derp.Wrap(err, location, "Unable to deleting ActivityStream", url))
			}
			return queue.Success()
		}

		// If it's "our fault" then we can't retry
		if derp.IsClientError(err) {
			return queue.Failure(derp.Wrap(err, location, "Client error when loading ActivityStream", url))
		}

		// Otherwise, it's "their fault" and it's worth retrying
		return queue.Error(derp.Wrap(err, location, "Unable to load ActivityStream"))
	}

	// No error => success!
	return queue.Success()
}
