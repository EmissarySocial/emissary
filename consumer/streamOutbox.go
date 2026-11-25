package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamOutbox brings a Stream's followers up to date after a save, by sending the
// Create, Update, or Delete that they are missing
func StreamOutbox(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.StreamOutbox"

	// RULE: A task that names no Stream can never succeed, so it is not retried
	streamID, err := primitive.ObjectIDFromHex(args.GetString("streamId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Parsing streamId", args))
	}

	if streamID.IsZero() {
		return queue.Failure(derp.BadRequest(location, "StreamID is required", args))
	}

	// Send whatever followers are missing.  An error is retried, and the transaction takes back any partial send.
	if err := factory.Stream().SyncOutbox(session, streamID); err != nil {
		return queue.Error(derp.Wrap(err, location, "Sending to followers", streamID))
	}

	// Make it so
	return queue.Success()
}
