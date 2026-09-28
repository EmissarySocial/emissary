package consumer

import (
	"context"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SyncStreamSourceAttachment copies one file, linked from a synchronized source, into the
// attachment that stands in for it.
//
// RULE: It runs through WithFactory, NOT WithSession.  A download can outlast MongoDB's 60-second
// transaction limit, and a retried transaction would download the file again.
func SyncStreamSourceAttachment(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.SyncStreamSourceAttachment"

	streamID, err := primitive.ObjectIDFromHex(args.GetString("streamId"))

	if err != nil {
		// No retry can repair a malformed identifier
		return queue.Failure(derp.Wrap(err, location, "Invalid 'streamId' argument", args))
	}

	attachmentID, err := primitive.ObjectIDFromHex(args.GetString("attachmentId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid 'attachmentId' argument", args))
	}

	// A plain session, deliberately NOT a transaction
	session, err := factory.Server().Session(context.Background())

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Opening database session"))
	}

	defer session.Close()

	// requeue files the author's mistakes as permanent failures, and Emissary's as retries
	if err := factory.StreamSource().ImportAttachment(session.Context(), session, streamID, attachmentID); err != nil {
		return requeue(derp.Wrap(err, location, "Importing attachment", attachmentID))
	}

	// Copied and filed
	return queue.Success()
}
