package queries

import (
	"testing"

	"github.com/EmissarySocial/emissary/realtime"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestWatchUsers_SendsUpdated pins the message a changed User produces
func TestWatchUsers_SendsUpdated(t *testing.T) {

	database := newWatchTestDatabase(t)
	userID := primitive.NewObjectID()

	messages := startMessageWatcher(t, database, WatchUsers)

	result := awaitMessage(t, messages, func() {
		upsertRecord(t, database, "User", userID, bson.M{"username": "groot"})
	})

	require.Equal(t, realtime.NewMessage_Updated(userID), result)
}

// TestWatchUsers_SkipsUnusableRecords pins that a User with a zero or undecodable ID sends nothing
func TestWatchUsers_SkipsUnusableRecords(t *testing.T) {

	database := newWatchTestDatabase(t)
	userID := primitive.NewObjectID()

	messages := startMessageWatcher(t, database, WatchUsers)

	result := awaitMessage(t, messages, func() {
		insertRecordWithStringID(t, database, "User")
		upsertRecord(t, database, "User", primitive.NilObjectID, bson.M{"username": "nobody"})
		upsertRecord(t, database, "User", userID, bson.M{"username": "groot"})
	})

	require.Equal(t, realtime.NewMessage_Updated(userID), result)
}
