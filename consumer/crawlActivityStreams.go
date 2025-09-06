package consumer

import (
	"net/http"

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
	history := args.GetSliceOfString("history")

	// Parse actorID
	actorID, err := primitive.ObjectIDFromHex(actorToken)

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid actorID"))
	}

	// Ok fam, it's about to get real.
	activityStreamCrawler := factory.ActivityStreamCrawler(actorType, actorID)
	err = activityStreamCrawler.Crawl(url, history)

	if err == nil {
		return queue.Success()
	}

	// If the ActivityStream no longer exists, then remove it from the cache
	if shouldDeleteActivityStream(err) {
		activityStreamService := factory.ActivityStream(actorType, actorID)
		if err := activityStreamService.Delete(url); err != nil {
			return queue.Error(derp.Wrap(err, location, "Unable to deleting ActivityStream", url))
		}
	}

	// Client errors should not be retried.
	if derp.IsClientError(err) {
		return queue.Failure(derp.Wrap(err, location, "Client error when loading ActivityStream", history))
	}

	// Server errors should be retried
	return queue.Error(derp.Wrap(err, location, "Unable to load ActivityStream"))
}

func shouldDeleteActivityStream(err error) bool {

	if derp.IsNotFound(err) {
		return true
	}

	if derp.ErrorCode(err) == http.StatusGone {
		return true
	}

	return false
}
