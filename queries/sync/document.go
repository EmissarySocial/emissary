package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Document synchronizes the Document collection in the SHARED DATABASE.
func Document(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Document").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Document"), indexer.IndexSet{

		"idx_Document_Metadata": mongo.IndexModel{
			Keys: bson.D{
				{Key: "metadata.relationType", Value: 1},
				{Key: "metadata.relationHref", Value: 1},
				{Key: "published", Value: 1},
			},
		},

		"idx_Document_FullText": mongo.IndexModel{
			Keys: bson.D{
				{Key: "_fts", Value: "text"},
				{Key: "_ftsx", Value: 1},
			},
			Options: options.Index().SetWeights(bson.M{
				"object.content":           1,
				"object.name":              1,
				"object.preferredUsername": 1,
				"object.summary":           1,
				"urls":                     1,
			}),
		},

		"idx_Document_Context": mongo.IndexModel{
			Keys: bson.D{
				{Key: "object.context", Value: 1},
				{Key: "published", Value: 1},
			},
		},

		"idx_Document_URLs": mongo.IndexModel{
			Keys: bson.D{
				{Key: "urls", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	})
}
