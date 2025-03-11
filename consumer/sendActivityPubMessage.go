package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SendActivityPubMessage sends an ActivityPub message to a single recipient/inboxURL
// `inboxURL` the URL to deliver the message to
// `actorType` the type of actor that is sending the message (User, Stream, Search)
// `actorID` unique ID of the actor (zero value for Search Actor)
// `message` the ActivityPub message to send
func SendActivityPubMessage(factory *domain.Factory, args mapof.Any) queue.Result {

	const location = "consumer.SendActivityPubMessage"

	// Collect arguments
	message := args.GetMap("message")
	inboxURL := args.GetString("inboxURL")

	// Find ActivityPub Actor
	actor, err := getActivityPubActor(factory, args)

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Error finding ActivityPub Actor"))
	}

	// Send the message to the inboxURL
	if err := actor.SendOne(inboxURL, message); err != nil {

		spew.Dump("Error sending ActivityPub message", message, err)

		// If the error is "our fault" we won't be able to correct it, so Fail now
		if derp.IsClientError(err) {
			return queue.Failure(derp.Wrap(err, location, "Error sending message", message))
		}

		// Otherwise, the error is "their fault" and we can try again later
		return queue.Error(derp.Wrap(err, location, "Error sending message", message))
	}

	spew.Dump("Successfully sent ActivityPub message", message)

	// Success
	return queue.Success()
}

func getActivityPubActor(factory *domain.Factory, args mapof.Any) (outbox.Actor, error) {

	const location = "consumer.SendActivityPubMessage.getActivityPubActor"

	switch args.GetString("actorType") {

	case model.FollowerTypeSearch:

		searchQueryID := args.GetString("searchQueryID")

		if actorID, err := primitive.ObjectIDFromHex(searchQueryID); err == nil {
			return factory.SearchQuery().ActivityPubActor(actorID, false)
		} else {
			return outbox.Actor{}, derp.Wrap(err, location, "Invalid searchQueryID", searchQueryID)
		}

	case model.FollowerTypeStream:

		streamID := args.GetString("streamID")

		if actorID, err := primitive.ObjectIDFromHex(streamID); err == nil {
			return factory.Stream().ActivityPubActor(actorID, false)
		} else {
			return outbox.Actor{}, derp.Wrap(err, location, "Invalid streamID", streamID)
		}

	case model.FollowerTypeUser:

		userID := args.GetString("userID")

		if actorID, err := primitive.ObjectIDFromHex(userID); err != nil {
			return factory.User().ActivityPubActor(actorID, false)
		} else {
			return outbox.Actor{}, derp.Wrap(err, location, "Invalid userID", userID)
		}
	}

	return outbox.Actor{}, derp.NewInternalError(location, "Invalid actorType", args)
}
