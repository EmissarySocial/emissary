package queries

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests cover SyncSharedIndexes through the connection handle the caller provides -- the
// contract that replaced the old dial-a-fresh-client-per-call signature, whose never-disconnected
// clients leaked topology goroutines and sockets on every configuration reload.  They skip when
// no database is reachable, so `go test ./...` still passes on a machine without one.

// syncTestConnection is the local MongoDB that these tests require
const syncTestConnection = "mongodb://localhost:27017/?directConnection=true"

// newSyncTestDatabase connects to a local MongoDB and returns a throwaway database that is
// dropped when the test ends.  The test is skipped when no database is reachable.
func newSyncTestDatabase(t *testing.T) *mongo.Database {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(syncTestConnection))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + syncTestConnection)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Skip("Skipping: no MongoDB at " + syncTestConnection)
	}

	// A per-test database name keeps parallel runs (and real data) well clear of this
	database := client.Database("emissary_synctest_" + primitive.NewObjectID().Hex())

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = database.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	return database
}

// indexNames returns the names of every index on a collection
func indexNames(t *testing.T, collection *mongo.Collection) []string {

	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Indexes().List(ctx)
	require.NoError(t, err)

	result := make([]string, 0)

	var index bson.M
	for cursor.Next(ctx) {
		require.NoError(t, cursor.Decode(&index))
		result = append(result, index["name"].(string))
	}

	return result
}

// TestSyncSharedIndexes pins the base case: syncing through a caller-provided database handle
// creates the shared indexes.  No connect string, no second client, nothing to leak.
func TestSyncSharedIndexes(t *testing.T) {

	database := newSyncTestDatabase(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	SyncSharedIndexes(ctx, database)

	// Spot-check one declared index on each of two shared collections
	require.Contains(t, indexNames(t, database.Collection("Document")), "idx_Document_Context")
	require.Contains(t, indexNames(t, database.Collection("ErrorLog")), "idx_ErrorLog_CreateDate")
}

// TestSyncSharedIndexes_Idempotent pins the property that makes syncing on every connection
// change safe: running twice converges to the same index set, with no errors and no duplicates.
func TestSyncSharedIndexes_Idempotent(t *testing.T) {

	database := newSyncTestDatabase(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	SyncSharedIndexes(ctx, database)
	first := indexNames(t, database.Collection("Document"))

	SyncSharedIndexes(ctx, database)
	second := indexNames(t, database.Collection("Document"))

	require.ElementsMatch(t, first, second)
}

// TestSyncSharedIndexes_CanceledContext pins the deadline contract: the caller owns the context
// (the factory runs this under its reload lock with a 60s budget), and a canceled context must
// end the sync promptly -- reported, not hung, not panicking.
func TestSyncSharedIndexes_CanceledContext(t *testing.T) {

	database := newSyncTestDatabase(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled: every index operation should fail fast

	finished := make(chan struct{})

	go func() {
		defer close(finished)
		SyncSharedIndexes(ctx, database)
	}()

	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("SyncSharedIndexes did not return promptly with a canceled context")
	}
}

// TestSyncDomainIndexes pins the domain half of the same contract: syncing through a
// caller-provided database handle creates the domain indexes, with no second client dialed.
func TestSyncDomainIndexes(t *testing.T) {

	database := newSyncTestDatabase(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	SyncDomainIndexes(ctx, database)

	// Spot-check a declared index on a domain collection
	require.Contains(t, indexNames(t, database.Collection("Stream")), "idx_Stream_Parent")
}

// TestSyncDomainIndexes_Idempotent pins convergence: running twice yields the same index set.
func TestSyncDomainIndexes_Idempotent(t *testing.T) {

	database := newSyncTestDatabase(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	SyncDomainIndexes(ctx, database)
	first := indexNames(t, database.Collection("Stream"))

	SyncDomainIndexes(ctx, database)
	second := indexNames(t, database.Collection("Stream"))

	require.ElementsMatch(t, first, second)
}
