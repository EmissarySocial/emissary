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
// TODO: This should be merged into Outbox:SendToSingleRecipient
func SendActivityPubMessage(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SendActivityPubMessage"

	// Get an ActivityStream service for the specified actor
	activityStreamService := factory.ActivityStream()

	// Send the ActivityPub message on the actor's behalf
	if err := activityStreamService.SendMessage(session, args); err != nil {
		return requeue(derp.Wrap(err, location, "Unable to send ActivityPub message"))
	}

	// There is only Woot.
	return queue.Success()
}
