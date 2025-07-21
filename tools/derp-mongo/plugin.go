package derpmongo

import (
	"context"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Plugin struct {
	collection   *mongo.Collection
	includeCodes sliceof.Int
	excludeCodes sliceof.Int
}

func New(collection *mongo.Collection, options mapof.Any) Plugin {

	return Plugin{
		collection:   collection,
		includeCodes: options.GetSliceOfInt("include-codes"),
		excludeCodes: options.GetSliceOfInt("exclude-codes"),
	}
}

func (plugin Plugin) Report(err error) {

	if err == nil {
		return
	}

	// Find and keep the status code to compare against the include/exclude lists
	statusCode := derp.ErrorCode(err)

	// If the status code is excluded, then do not log it.
	if plugin.excludeCodes.Contains(statusCode) {
		return
	}

	// If the "include list" is not empty, then only log errors that match the list.
	if plugin.includeCodes.NotEmpty() {
		if !plugin.includeCodes.Contains(statusCode) {
			return
		}
	}

	// We're gonna log the error..  I'm not scared.
	record := Record{
		RecordID:   primitive.NewObjectID(),
		StatusCode: statusCode,
		Location:   derp.RootLocation(err),
		Message:    derp.RootMessage(err),
		Error:      err,
		CreateDate: primitive.NewDateTimeFromTime(time.Now()),
	}

	if _, err := plugin.collection.InsertOne(context.Background(), record); err != nil {
		log.Error().Err(err).Msg("Unable to insert error record into MongoDB")
	}
}
