package queries

import (
	"context"
	"fmt"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestUpgradeMongoDB_CopiesTheLegacyOutbox verifies that the legacy outbox is copied into
// OutboxItem whether a database recorded version 37 or version 38
func TestUpgradeMongoDB_CopiesTheLegacyOutbox(t *testing.T) {

	// upgradeFrom upgrades a database that recorded startVersion, and checks the copy and the version
	upgradeFrom := func(startVersion uint) {
		t.Run(fmt.Sprintf("from version %d", startVersion), func(t *testing.T) {

			// Dev branches recorded version 38 for migrations that never released, so a database
			// at 38 must still run the copy in slot 39 (OUTBOX2-OMNIBUS §1.21, 2026-11-22).
			database := newSyncTestDatabase(t)
			ctx := context.Background()

			// A Domain at the starting version, and one legacy outbox row
			_, err := database.Collection("Domain").InsertOne(ctx, bson.M{"_id": primitive.NilObjectID, "databaseVersion": startVersion})
			require.NoError(t, err)

			rowID := primitive.NewObjectID()
			row := bson.M{
				"_id":          rowID,
				"actorId":      primitive.NewObjectID(),
				"actorType":    "User",
				"actorUrl":     "https://example.com/@ben",
				"activityType": "Create",
				"objectId":     "https://example.com/@ben/notes/1",
				"permissions":  bson.A{model.MagicGroupIDAnonymous},
				"createDate":   int64(1_500_000_000_000),
			}

			_, err = database.Collection("Outbox").InsertOne(ctx, row)
			require.NoError(t, err)

			// Upgrade, then find the row in OutboxItem by its own _id
			require.NoError(t, UpgradeMongoDB(ctx, database, startVersion))

			var copied struct {
				ObjectID   string `bson:"objectId"`
				CreateDate int64  `bson:"createDate"`
			}

			require.NoError(t, database.Collection("OutboxItem").FindOne(ctx, bson.M{"_id": rowID}).Decode(&copied), "the legacy row is copied")
			require.Equal(t, "https://example.com/@ben/notes/1", copied.ObjectID)
			require.Equal(t, int64(1_500_000_000_000), copied.CreateDate)

			// The Domain records slot 39 or a later one
			var domain struct {
				DatabaseVersion uint `bson:"databaseVersion"`
			}

			require.NoError(t, database.Collection("Domain").FindOne(ctx, bson.M{"_id": primitive.NilObjectID}).Decode(&domain))
			require.GreaterOrEqual(t, domain.DatabaseVersion, uint(39))
		})
	}

	upgradeFrom(37)
	upgradeFrom(38)
}

// TestUpgradeMongoDB_FlattensSyndication verifies that a database at version 39 runs slot 40,
// which rewrites a Stream's legacy syndication document as a plain array, and records the slot
func TestUpgradeMongoDB_FlattensSyndication(t *testing.T) {

	database := newSyncTestDatabase(t)
	ctx := context.Background()

	_, err := database.Collection("Domain").InsertOne(ctx, bson.M{"_id": primitive.NilObjectID, "databaseVersion": uint(39)})
	require.NoError(t, err)

	streamID := primitive.NewObjectID()
	legacy := bson.M{"_id": streamID, "syndication": bson.M{"values": bson.A{"mastodon", "bluesky"}}}

	_, err = database.Collection("Stream").InsertOne(ctx, legacy)
	require.NoError(t, err)

	require.NoError(t, UpgradeMongoDB(ctx, database, 39))

	stored := bson.M{}
	require.NoError(t, database.Collection("Stream").FindOne(ctx, bson.M{"_id": streamID}).Decode(&stored))
	require.Equal(t, bson.A{"mastodon", "bluesky"}, stored["syndication"])

	var domain struct {
		DatabaseVersion uint `bson:"databaseVersion"`
	}

	require.NoError(t, database.Collection("Domain").FindOne(ctx, bson.M{"_id": primitive.NilObjectID}).Decode(&domain))
	require.GreaterOrEqual(t, domain.DatabaseVersion, uint(40))

	// A database that already recorded slot 40 is left alone
	require.NoError(t, UpgradeMongoDB(ctx, database, domain.DatabaseVersion))
}
