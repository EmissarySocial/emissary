package consumer

import (
	"context"
	"time"

	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

// PurgeActivityStreamCache removes errors from the database that are older than 1 week.
func PurgeActivityStreamCache(factory ServerFactory) queue.Result {

	log.Trace().Msg("Task: PurgeActivityStreamCache")

	// Purge documents that expired >2 days ago
	collection := factory.CommonDatabase().Collection("Document")

	criteria := bson.M{
		"expires": bson.M{"$lt": time.Now().AddDate(0, 0, -2).Unix()},
	}

	if _, err := collection.DeleteMany(context.Background(), criteria); err != nil {
		return queue.Error(err)
	}

	// Glorious success
	return queue.Success()
}
