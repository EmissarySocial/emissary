package indexer

import (
	"context"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Sync(ctx context.Context, collection *mongo.Collection, newIndexes map[string]mongo.IndexModel) error {

	const location = "tools.indexer.Sync"

	// Prepare the index set
	for key, newIndex := range newIndexes {

		if newIndex.Options == nil {
			newIndex.Options = options.Index()
		}

		newIndex.Options.SetName(key)
		newIndexes[key] = newIndex
	}

	// Get all indexes in the collection
	cursor, err := collection.Indexes().List(ctx)

	if err != nil {
		return err
	}

	database := collection.Database().Name()

	// Scan all existing indexes to Delete any that are no longer used
	for currentIndex := range rangeFunc[mapof.Any](ctx, cursor) {

		name := currentIndex.GetString("name")

		// Skip the default _id_ index, as it cannot be removed or modified
		if name == "_id_" {
			continue
		}

		// See if the indexes match.  If so, then there's nothing to delete
		if newIndex, exists := newIndexes[name]; exists {

			if compareModel(currentIndex, newIndex) {
				log.Trace().Str("database", database).Str("index", name).Msg("index in sync.")
				delete(newIndexes, name)
				continue
			}
		}

		// Fall through means that the index has been changed or deleted.  Drop the old index
		if bsonRaw, err := collection.Indexes().DropOne(ctx, name); err != nil {
			derp.Report(derp.Wrap(err, location, "Error dropping index", "index", name, bsonRaw))
		}
	}

	// Add new indexes that are not already in the collection
	for indexName, newIndex := range newIndexes {
		log.Trace().Str("database", database).Str("index", indexName).Msg("Creating added/changed index...")
		if bsonRaw, err := collection.Indexes().CreateOne(ctx, newIndex); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to create index", "index", indexName, newIndex, bsonRaw))
			continue
		}
	}

	// Success.
	return nil
}
