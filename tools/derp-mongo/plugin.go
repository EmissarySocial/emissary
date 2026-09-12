package derpmongo

import (
	"context"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

// Plugin is a derp.Reporter that writes error records into a MongoDB collection
type Plugin struct {
	collection   *mongo.Collection
	includeCodes sliceof.Int
	excludeCodes sliceof.Int
}

// New returns a fully initialized Plugin that writes to the provided collection
func New(collection *mongo.Collection, options mapof.Any) Plugin {

	return Plugin{
		collection:   collection,
		includeCodes: options.GetSliceOfInt("include-codes"),
		excludeCodes: options.GetSliceOfInt("exclude-codes"),
	}
}

// Report implements the derp.Plugin interface, writing the error to MongoDB unless its status code is filtered out
func (plugin Plugin) Report(err error) {

	if err == nil {
		return
	}

	// Find and keep the status code to compare against the include/exclude lists
	statusCode := derp.ErrorCode(err)

	// RULE: The configured lists decide whether this error belongs in the log at all
	if !plugin.isReportable(statusCode) {
		return
	}

	// We're gonna log the error..  I'm not scared.
	record := newRecord(err, statusCode)

	if _, err := plugin.collection.InsertOne(context.Background(), record); err != nil {
		log.Error().Err(err).Msg("Unable to insert error record into MongoDB")
	}
}

// isReportable returns TRUE if an error with this status code belongs in the log
func (plugin Plugin) isReportable(statusCode int) bool {

	// RULE: An excluded status code is never logged
	if plugin.excludeCodes.Contains(statusCode) {
		return false
	}

	// RULE: When an include list exists, nothing outside of it is logged
	if plugin.includeCodes.NotEmpty() {
		return plugin.includeCodes.Contains(statusCode)
	}

	return true
}
