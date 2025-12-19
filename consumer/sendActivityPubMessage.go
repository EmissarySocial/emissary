package consumer

import (
	"time"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SendActivityPubMessage sends an ActivityPub message to a single recipient/inboxURL
// `inboxURL` the URL to deliver the message to
// `actorType` the type of actor that is sending the message (User, Stream, Search)
// `message` the ActivityPub message to send
func SendActivityPubMessage(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SendActivityPubMessage"

	// Parse arguments
	actorType := args.GetString("actorType")
	actorIDHex := args.GetString("actorID")
	actorID, err := primitive.ObjectIDFromHex(actorIDHex)

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid ActorID", actorIDHex))
	}

	// Get an ActivityStream service for the specified actor
	activityStreamService := factory.ActivityStream(actorType, actorID)

	// Send the ActivityPub message on the actor's behalf
	if err := activityStreamService.SendMessage(session, args); err != nil {

		// If it is a certain kind of HTTP error, then maybe retry it...
		if httpError := derp.UnwrapHTTPError(err); httpError != nil {

			// Retry HTTP 429 (Too Many Requests) errors
			if derp.IsTooManyRequests(err) {

				// See how long we should wait before retrying
				// +2 minutes because everyone needs two extra minutes. UwU
				if retryAfter := httpError.RetryAfter(); retryAfter > 0 {
					retryDuration := time.Duration(retryAfter+120) * time.Second
					return queue.Requeue(retryDuration)
				}

				// If not retry-after duration is given, use 15 minutes as a default
				retryDuration := 15 * time.Minute
				return queue.Requeue(retryDuration)
			}
		}

		// If this is our fault then it can't be retried. Fail accordingly.
		if derp.IsClientError(err) {
			return queue.Failure(derp.Wrap(err, location, "Unable to deliver ActivityPub message (Client Error cannot be retried)"))
		} else {
			return queue.Error(derp.Wrap(err, location, "Unable to deliver ActivityPub message (Server Error can be retried)"))
		}
	}

	// There is only Woot.
	return queue.Success()
}
