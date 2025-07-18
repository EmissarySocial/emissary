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

	// Scan all existing indexes to Update or Delete
	for currentIndex := range rangeFunc[mapof.Any](ctx, cursor) {

		name := currentIndex.GetString("name")

		// Skip the default _id index, as it cannot be removed or modified
		if name == "_id_" {
			continue
		}

		// See if the index already exists in the new set
		newIndex, exists := newIndexes[name]

		if !exists {
			continue
		}

		delete(newIndexes, name)

		if compareModel(currentIndex, newIndex) {
			log.Debug().Str("database", database).Str("index", name).Msg("index in sync.")
			continue
		}

		log.Debug().Str("database", database).Str("index", name).Msg("deleting changed index...")
		if bsonRaw, err := collection.Indexes().DropOne(ctx, name); err != nil {
			derp.Report(derp.Wrap(err, location, "Error updating index", "index", name, newIndex, bsonRaw))
			continue
		}

		if exists {
			log.Debug().Str("database", database).Str("index", name).Msg("recreating changed index...")
			if bsonRaw, err := collection.Indexes().CreateOne(ctx, newIndex); err != nil {
				derp.Report(derp.Wrap(err, location, "Error creating index", "index", name, newIndex, bsonRaw))
				continue
			}
		}
	}

	// Add new indexes that are not already in the collection
	for indexName, newIndex := range newIndexes {
		log.Debug().Str("database", database).Str("index", indexName).Msg("adding new index...")
		if bsonRaw, err := collection.Indexes().CreateOne(ctx, newIndex); err != nil {
			derp.Report(derp.Wrap(err, location, "Error creating index", "index", indexName, newIndex, bsonRaw))
			continue
		}
	}

	// Success.
	return nil
}
