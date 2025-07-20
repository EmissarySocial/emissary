package derpmongo

import (
	"context"
	"time"

	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Plugin struct {
	collection *mongo.Collection
}

func New(collection *mongo.Collection) Plugin {
	return Plugin{
		collection: collection,
	}
}

func (plugin Plugin) Report(err error) {

	if err == nil {
		return
	}

	record := Record{
		RecordID:   primitive.NewObjectID(),
		StatusCode: derp.ErrorCode(err),
		Location:   derp.Location(err),
		Message:    derp.Message(err),
		Error:      err,
		CreateDate: primitive.NewDateTimeFromTime(time.Now()),
	}

	if _, err := plugin.collection.InsertOne(context.Background(), record); err != nil {
		log.Error().Err(err).Msg("Unable to insert error record into MongoDB")
	}
}
