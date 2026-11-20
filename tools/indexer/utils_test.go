package indexer

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestCompareModel_TTLIndex confirms a TTL index matches the shape MongoDB reports for it, so Sync keeps it
func TestCompareModel_TTLIndex(t *testing.T) {

	declared := mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetName("idx_Example_Expires").SetExpireAfterSeconds(0),
	}

	// What listIndexes returns for that index
	live := mapof.Any{
		"v":                  int32(2),
		"key":                mapof.Any{"expiresAt": int32(1)},
		"name":               "idx_Example_Expires",
		"expireAfterSeconds": int32(0),
	}

	require.True(t, compareModel(live, declared))

	// A different expiry is a different index, which Sync rebuilds
	live["expireAfterSeconds"] = int32(60)
	require.False(t, compareModel(live, declared))
}
