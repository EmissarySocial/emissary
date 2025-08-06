package consumer

import (
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
	actorID, _ := primitive.ObjectIDFromHex(actorIDHex)

	// Get an ActivityStream service for the specified actor
	activityStreamService := factory.ActivityStream(actorType, actorID)

	// Send the ActivityPub message on the actor's behalf
	if err := activityStreamService.SendMessage(session, args); err != nil {
		return queue.Failure(derp.Wrap(err, location, "Error sending ActivityPub message"))
	}

	// There is only Woot.
	return queue.Success()
}
