package upgrades

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// These tests drive Version29 against a REAL MongoDB, because the behavior that matters is
// $addToSet's: whether it creates the array on a record that has no notificationChannels field at
// all, whether it leaves an already-present value alone, and whether it preserves the other
// channels. A fake would only re-assert the query I wrote. They skip when no database is reachable,
// so `go test ./...` still passes without one.

// insertUserWithChannels writes one raw User document. Pass nil for `channels` to write a record
// with NO notificationChannels field at all, as the oldest records (predating the field) look.
func insertUserWithChannels(t *testing.T, database *mongo.Database, channels []string) primitive.ObjectID {

	t.Helper()

	userID := primitive.NewObjectID()
	document := bson.M{"_id": userID}

	if channels != nil {
		document["notificationChannels"] = channels
	}

	_, err := database.Collection("User").InsertOne(context.Background(), document)
	require.NoError(t, err)

	return userID
}

// loadChannels reads back one User's notificationChannels.
func loadChannels(t *testing.T, database *mongo.Database, userID primitive.ObjectID) []string {

	t.Helper()

	var result struct {
		NotificationChannels []string `bson:"notificationChannels"`
	}

	err := database.Collection("User").FindOne(context.Background(), bson.M{"_id": userID}).Decode(&result)
	require.NoError(t, err)

	return result.NotificationChannels
}

// TestVersion29 covers every shape of stored channel list, including the two that could silently
// lose direct messages: a record with no field at all, and a record with an empty list.
func TestVersion29(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	// A typical user with the pre-existing defaults.
	typical := insertUserWithChannels(t, database, []string{
		model.NotificationChannelMentionFollowing,
		model.NotificationChannelMentionNotFollowing,
		model.NotificationChannelReply,
	})

	// A user who deliberately switched everything off. DECIDED: they get the channel anyway.
	silent := insertUserWithChannels(t, database, []string{})

	// The oldest records carry no notificationChannels key whatsoever.
	ancient := insertUserWithChannels(t, database, nil)

	// A user who somehow already has the channel -- the migration must not duplicate it.
	already := insertUserWithChannels(t, database, []string{
		model.NotificationChannelDirectMessage,
		model.NotificationChannelFollow,
	})

	require.NoError(t, Version29(ctx, database))

	require.Contains(t, loadChannels(t, database, typical), model.NotificationChannelDirectMessage)
	require.Contains(t, loadChannels(t, database, silent), model.NotificationChannelDirectMessage)
	require.Contains(t, loadChannels(t, database, ancient), model.NotificationChannelDirectMessage)

	// The other channels survive untouched.
	require.ElementsMatch(t, []string{
		model.NotificationChannelDirectMessage,
		model.NotificationChannelMentionFollowing,
		model.NotificationChannelMentionNotFollowing,
		model.NotificationChannelReply,
	}, loadChannels(t, database, typical))

	// No duplicate for the user who already had it.
	require.ElementsMatch(t, []string{
		model.NotificationChannelDirectMessage,
		model.NotificationChannelFollow,
	}, loadChannels(t, database, already))
}

// TestVersion29_Idempotent guards the boot-chain requirement that a migration can be re-run
// safely -- a second pass must add nothing.
func TestVersion29_Idempotent(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	userID := insertUserWithChannels(t, database, []string{model.NotificationChannelReply})

	require.NoError(t, Version29(ctx, database))
	first := loadChannels(t, database, userID)

	require.NoError(t, Version29(ctx, database))
	second := loadChannels(t, database, userID)

	require.Equal(t, first, second)
	require.Len(t, second, 2)
}
