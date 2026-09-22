package upgrades

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// insertFollowingWithStatus inserts a Following carrying only the two fields Version34 reads.
func insertFollowingWithStatus(t *testing.T, database *mongo.Database, status string, statusMessage string) primitive.ObjectID {

	t.Helper()

	id := primitive.NewObjectID()
	document := bson.M{"_id": id, "status": status, "statusMessage": statusMessage}

	_, err := database.Collection("Following").InsertOne(context.Background(), document)
	require.NoError(t, err)

	return id
}

// loadFollowingStatus returns the status and message of one Following.
func loadFollowingStatus(t *testing.T, database *mongo.Database, id primitive.ObjectID) (string, string) {

	t.Helper()

	var result struct {
		Status        string `bson:"status"`
		StatusMessage string `bson:"statusMessage"`
	}

	err := database.Collection("Following").FindOne(context.Background(), bson.M{"_id": id}).Decode(&result)
	require.NoError(t, err)

	return result.Status, result.StatusMessage
}

func TestVersion34(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	// A legacy block, exactly as the old Pause() wrote it
	legacyID := insertFollowingWithStatus(t, database, "PAUSED", "Paused by a block rule")

	// RULE: A new-meaning PAUSED row (backed off after failures) carries a different message and
	// must be left alone, because this upgrade can run AFTER the new binary has already polled.
	backedOffID := insertFollowingWithStatus(t, database, "PAUSED", "Could not reach this server: no such host")

	// Every other status is untouched
	successID := insertFollowingWithStatus(t, database, "SUCCESS", "")
	failureID := insertFollowingWithStatus(t, database, "FAILURE", "This account refused our request (403).")

	require.NoError(t, Version34(ctx, database))

	status, message := loadFollowingStatus(t, database, legacyID)
	require.Equal(t, "BLOCKED", status, "a legacy block row must be renamed")
	require.Equal(t, "You blocked this account", message, "and its message brought up to date")

	status, message = loadFollowingStatus(t, database, backedOffID)
	require.Equal(t, "PAUSED", status, "a backed-off row must NOT be mistaken for a block")
	require.Equal(t, "Could not reach this server: no such host", message)

	status, _ = loadFollowingStatus(t, database, successID)
	require.Equal(t, "SUCCESS", status)

	status, _ = loadFollowingStatus(t, database, failureID)
	require.Equal(t, "FAILURE", status)
}

func TestVersion34_Idempotent(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	legacyID := insertFollowingWithStatus(t, database, "PAUSED", "Paused by a block rule")

	require.NoError(t, Version34(ctx, database))
	require.NoError(t, Version34(ctx, database))

	status, _ := loadFollowingStatus(t, database, legacyID)
	require.Equal(t, "BLOCKED", status, "a second run should make no additional changes")
}
