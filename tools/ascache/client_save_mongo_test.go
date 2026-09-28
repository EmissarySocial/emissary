package ascache

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	querysync "github.com/EmissarySocial/emissary/queries/sync"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests run the cache write against a real replica set, because the conflict they pin comes
// from MongoDB's unique index and transactions, which the fake database does not model.  They skip
// when no database is reachable, so `go test ./...` still passes on a machine without one.

// TestClient_Save_ConcurrentServers confirms that writers on several servers saving one uncached
// URL at the same moment all succeed, and leave one cached document behind.
func TestClient_Save_ConcurrentServers(t *testing.T) {

	database := newSaveTestDatabase(t)

	const servers = 8
	const url = "https://mastodon.example/ap/users/117117165858773039"

	errs := make([]error, servers)
	start := make(chan struct{})

	var done sync.WaitGroup

	for index := range servers {
		done.Go(func() {

			// Each server builds its own client, configured as production does, and its own copy of the document
			client := New(&countingClient{}, nil, mongodb.NewServer(database), "Application", primitive.NilObjectID, "server"+strconv.Itoa(index)+".example", WithIgnoreHeaders())
			value := asValue(streams.NewDocument(map[string]any{vocab.PropertyID: url, vocab.PropertyType: vocab.ActorTypePerson}))

			<-start

			ctx, cancel := timeoutContext(defaultDatabaseTimeout)
			defer cancel()

			errs[index] = client.save(ctx, url, &value)
		})
	}

	// Release every server at once, as a burst of deliveries does
	close(start)
	done.Wait()

	for index, err := range errs {
		require.NoError(t, err, "server %d", index)
	}

	count, err := database.Collection("Document").CountDocuments(context.Background(), bson.M{"urls": url})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

/******************************************
 * Helpers
 ******************************************/

// saveTestConnection is the local MongoDB that these tests require
const saveTestConnection = "mongodb://localhost:27017/?directConnection=true"

// newSaveTestDatabase connects to a local replica set and returns a throwaway database carrying the
// production Document indexes, dropped when the test ends.  The test is skipped when none is reachable.
func newSaveTestDatabase(t *testing.T) *mongo.Database {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(saveTestConnection))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + saveTestConnection)
	}

	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})

	if err := client.Ping(ctx, nil); err != nil {
		t.Skip("Skipping: no MongoDB at " + saveTestConnection)
	}

	// RULE: The write runs in a transaction, which a standalone server refuses
	var hello bson.M
	if err := client.Database("admin").RunCommand(ctx, bson.M{"hello": 1}).Decode(&hello); err != nil || hello["setName"] == nil {
		t.Skip("Skipping: MongoDB at " + saveTestConnection + " is not a replica set")
	}

	// A per-test database name keeps parallel runs (and the real cache) well clear of this
	database := client.Database("ascache_test_" + primitive.NewObjectID().Hex())

	t.Cleanup(func() {
		_ = database.Drop(context.Background())
	})

	// The unique index on urls is what makes concurrent writers conflict, so it must match production
	require.NoError(t, querysync.Document(ctx, database))

	return database
}
