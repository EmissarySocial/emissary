package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamFillFingerprints replaces the placeholder fingerprints on one page of a Domain's Streams,
// and queues itself again until no page remains.  It sends nothing to followers.
func StreamFillFingerprints(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.StreamFillFingerprints"

	// The first run has no "after", and starts from the beginning
	afterID := primitive.NilObjectID

	if after := args.GetString("after"); after != "" {

		// RULE: A page marker that cannot be read can never succeed, so it is not retried
		parsed, err := primitive.ObjectIDFromHex(after)

		if err != nil {
			return queue.Failure(derp.Wrap(err, location, "Parsing 'after'", args))
		}

		afterID = parsed
	}

	lastID, done, err := factory.Stream().FillFingerprints(session, afterID)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Filling fingerprints", factory.Hostname()))
	}

	if done {
		return queue.Success()
	}

	// Read the next page in a task of its own, so that no transaction outlasts its time limit
	postcommit.Publish(session, factory.Queue(), service.StreamFillFingerprintsTaskName, mapof.Any{
		"hostname": factory.Hostname(),
		"after":    lastID.Hex(),
	})

	// To be continued
	return queue.Success()
}
