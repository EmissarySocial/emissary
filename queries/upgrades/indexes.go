package upgrades

import (
	"context"

	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MakeIndex is a pretty interface to creating a standard index in MongoDB
func MakeIndex(db *mongo.Database, collectionName string, indexName string, keyNames ...string) error {

	// Collect prerequisites
	ctx := context.Background()
	collection := db.Collection(collectionName)
	indexes := collection.Indexes()

	// Try to remove the index if it already exists
	if indexExists, err := IndexExists(indexes, indexName); err != nil {
		return derp.Wrap(err, "queries.upgrades.MakeIndex", "Error checking for index")
	} else if indexExists {
		log.Trace().Str("index", indexName).Msg("Dropping existing index")
		_, _ = indexes.DropOne(ctx, indexName)
	} else {
		log.Trace().Str("index", indexName).Msg("Creating new index")
	}

	// Define Index keys and model
	keys := make(bson.D, len(keyNames))
	for i, key := range keyNames {
		keys[i] = bson.E{Key: key, Value: 1}
	}

	indexModel := mongo.IndexModel{
		Keys: keys,
		Options: options.Index().
			SetName(indexName).
			SetPartialFilterExpression(bson.M{"deleteDate": 0}),
	}

	// Create the index
	if _, err := indexes.CreateOne(ctx, indexModel); err != nil {
		return err
	}

	log.Trace().Str("index", indexName).Msg("Index created")

	// Done
	return nil
}

// IndexExists checks if a specific index exists in a collection
func IndexExists(indexes mongo.IndexView, indexName string) (bool, error) {

	// List indexes
	indexList, err := indexes.ListSpecifications(context.Background())

	// Report Errors
	if err != nil {
		return false, derp.Wrap(err, "queries.upgrades.IndexExists", "Error checking for index")
	}

	// Search list for matching index name
	for _, index := range indexList {
		if index.Name == indexName {
			return true, nil
		}
	}

	// No match found
	return false, nil
}
