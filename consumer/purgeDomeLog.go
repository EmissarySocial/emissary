package consumer

import (
	"context"
	"time"

	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
)

// PurgeDomeLog removes errors from the database that are older than 1 month.
func PurgeDomeLog(factory ServerFactory) queue.Result {

	log.Trace().Msg("Task: PurgeDomeLog")

	// Purge old Error records from the error log
	collection := factory.CommonDatabase().Collection("Log")

	_, err := collection.DeleteMany(
		context.Background(),
		bson.M{
			"createDate": bson.M{"$lt": time.Now().AddDate(0, -1, 0)},
		},
	)

	// Handle error when purging errors
	if err != nil {
		log.Error().Err(err).Msg("Unable to purge old error records")
		return queue.Error(err)
	}

	// Glorious success
	return queue.Success()
}
