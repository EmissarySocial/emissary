package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DeleteStream is a scheduled job to delete a Stream after a certain period of time.
func DeleteStream(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {
	const location = "consumer.DeleteStream"

	// Locate the StreamID parameter
	token := args.GetString("streamId")
	streamID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid StreamID", token))
	}

	// Try to load the Stream from the database
	streamService := factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByID(session, streamID, &stream); err != nil {

		if derp.IsNotFound(err) {
			return queue.Success()
		}

		return queue.Error(derp.Wrap(err, location, "Error loading stream", token))
	}

	// Delete the Stream
	if err := streamService.Delete(session, &stream, "Scheduled delete"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error deleting stream", stream))
	}

	// Woot.
	return queue.Success()
}
