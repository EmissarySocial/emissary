package upgrades

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// insertFollowerWithState inserts a Follower carrying only the field Version35 reads.
func insertFollowerWithState(t *testing.T, database *mongo.Database, stateID string) primitive.ObjectID {

	t.Helper()

	id := primitive.NewObjectID()

	_, err := database.Collection("Follower").InsertOne(context.Background(), bson.M{"_id": id, "stateId": stateID})
	require.NoError(t, err)

	return id
}

// loadFollowerState returns the state of one Follower.
func loadFollowerState(t *testing.T, database *mongo.Database, id primitive.ObjectID) string {

	t.Helper()

	var result struct {
		StateID string `bson:"stateId"`
	}

	err := database.Collection("Follower").FindOne(context.Background(), bson.M{"_id": id}).Decode(&result)
	require.NoError(t, err)

	return result.StateID
}

func TestVersion35(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	pausedID := insertFollowerWithState(t, database, "PAUSED")
	activeID := insertFollowerWithState(t, database, "ACTIVE")
	pendingID := insertFollowerWithState(t, database, "PENDING")

	require.NoError(t, Version35(ctx, database))

	require.Equal(t, "BLOCKED", loadFollowerState(t, database, pausedID), "a blocked Follower must be renamed")
	require.Equal(t, "ACTIVE", loadFollowerState(t, database, activeID), "other states are left alone")
	require.Equal(t, "PENDING", loadFollowerState(t, database, pendingID), "other states are left alone")
}

func TestVersion35_Idempotent(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	pausedID := insertFollowerWithState(t, database, "PAUSED")

	require.NoError(t, Version35(ctx, database))
	require.NoError(t, Version35(ctx, database))

	require.Equal(t, "BLOCKED", loadFollowerState(t, database, pausedID), "a second run should make no additional changes")
}
