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

	// Purge old Error records from the error log
	collection := factory.CommonDatabase().Collection("Document")

	_, err := collection.DeleteMany(
		context.Background(),
		bson.M{
			"expires": bson.M{"$lt": time.Now().AddDate(0, -2, 0)},
		},
	)

	// Handle error when purging errors
	if err != nil {
		log.Error().Err(err).Msg("Unable to purge old ActivityStream cache documents")
		return queue.Error(err)
	}

	// Glorious success
	return queue.Success()
}
