package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// SendActivityPubMessage sends an ActivityPub message to a single recipient/inboxURL
// `inboxURL` the URL to deliver the message to
// `actorType` the type of actor that is sending the message (User, Stream, Search)
// `message` the ActivityPub message to send
func SendActivityPubMessage(factory *domain.Factory, args mapof.Any) queue.Result {

	const location = "consumer.SendActivityPubMessage"

	activityStreamService := factory.ActivityStream()

	if err := activityStreamService.SendMessage(args); err != nil {
		return queue.Failure(derp.Wrap(err, location, "Error sending ActivityPub message"))
	}

	return queue.Success()
}
