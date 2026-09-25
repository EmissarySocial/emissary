package queries

import (
	"testing"

	"github.com/EmissarySocial/emissary/realtime"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestWatchImports_SendsImportProgress pins the message a changed Import produces
func TestWatchImports_SendsImportProgress(t *testing.T) {

	database := newWatchTestDatabase(t)
	importID := primitive.NewObjectID()

	messages := startMessageWatcher(t, database, WatchImports)

	result := awaitMessage(t, messages, func() {
		upsertRecord(t, database, "Import", importID, bson.M{"stateId": "IMPORTING"})
	})

	require.Equal(t, realtime.NewMessage_ImportProgress(importID), result)
}

// TestWatchImports_SkipsUnusableRecords pins that an Import with a zero or undecodable ID sends nothing
func TestWatchImports_SkipsUnusableRecords(t *testing.T) {

	database := newWatchTestDatabase(t)
	importID := primitive.NewObjectID()

	messages := startMessageWatcher(t, database, WatchImports)

	result := awaitMessage(t, messages, func() {
		insertRecordWithStringID(t, database, "Import")
		upsertRecord(t, database, "Import", primitive.NilObjectID, bson.M{"stateId": "IMPORTING"})
		upsertRecord(t, database, "Import", importID, bson.M{"stateId": "IMPORTING"})
	})

	require.Equal(t, realtime.NewMessage_ImportProgress(importID), result)
}
