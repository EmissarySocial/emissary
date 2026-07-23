package consumer

import (
	"time"

	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

// PurgeActivityStreamCache deletes cached ActivityStream documents that expired more than two days ago.
func PurgeActivityStreamCache(factory ServerFactory) queue.Result {

	log.Trace().Msg("Task: PurgeActivityStreamCache")

	// Purge documents that expired >2 days ago
	collection := factory.CommonDatabase().Collection("Document")

	criteria := bson.M{
		"expires": bson.M{"$lt": time.Now().AddDate(0, 0, -2).Unix()},
	}

	// Bound the delete to 180s (matching queries.Recycle) so a slow purge of a large
	// collection cannot hold this queue worker open indefinitely.
	ctx, cancel := timeoutContext(180)
	defer cancel()

	if _, err := collection.DeleteMany(ctx, criteria); err != nil {
		return queue.Error(err)
	}

	// Glorious success
	return queue.Success()
}
