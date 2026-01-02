package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// SendActivityPubMessage sends an ActivityPub message to a single recipient/inboxURL
// `inboxURL` the URL to deliver the message to
// `actorType` the type of actor that is sending the message (User, Stream, Search)
// `message` the ActivityPub message to send
func SendActivityPubMessage(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SendActivityPubMessage"

	// Get an ActivityStream service for the specified actor
	activityStreamService := factory.ActivityStream()

	// Send the ActivityPub message on the actor's behalf
	if err := activityStreamService.SendMessage(session, args); err != nil {

		// Retry HTTP 429 (Too Many Requests) errors
		if tooManyRequests, retryDuration := derp.IsTooManyRequests(err); tooManyRequests {
			return queue.Requeue(retryDuration)
		}

		// If this is our fault then it can't be retried. Fail accordingly.
		if derp.IsClientError(err) {
			return queue.Failure(derp.Wrap(err, location, "Unable to deliver ActivityPub message (Client Error cannot be retried)"))
		}

		// Otherwise, it's the Server's fault, and we can retry
		return queue.Error(derp.Wrap(err, location, "Unable to deliver ActivityPub message (Server Error can be retried)"))
	}

	// There is only Woot.
	return queue.Success()
}
