package service

import (
	"context"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	querysync "github.com/EmissarySocial/emissary/queries/sync"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/EmissarySocial/emissary/tools/ashash"
	"github.com/EmissarySocial/emissary/tools/asrules"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/hannibal/clients"
	"github.com/benpate/hannibal/metadata"
	"github.com/benpate/hannibal/streams"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests run every cycle shape through the whole production stack, including ascache over a
// real replica set, because the stub and the cache only meet there.  They skip when no database is
// reachable, so `go test ./...` still passes on a machine without one.

// cycleTestConnection is the local MongoDB that these tests require
const cycleTestConnection = "mongodb://localhost:27017/?directConnection=true"

// TestClientCycle_FullStack_EveryShapeLoads confirms that every cycle shape loads through the whole
// stack, and that nothing without a type is left in the cache.
func TestClientCycle_FullStack_EveryShapeLoads(t *testing.T) {

	// BUG-212: the chain's depth-limit stub is the one no later save overwrites, so without
	// ascache's type check it stays cached as the document for its URL.
	database := newCycleTestDatabase(t)
	server := newCycleShapeServer(t)

	for _, url := range server.shapeURLs() {
		loadWithin(t, fullStack(clients.NewCarpool(), database), url)
	}

	require.Zero(t, untypedCacheEntries(t, database))
}

// TestClientCycle_FullStack_SecondLoadIsCached confirms that loading every shape again is answered
// by the cache, with no request reaching the server.
func TestClientCycle_FullStack_SecondLoadIsCached(t *testing.T) {

	database := newCycleTestDatabase(t)
	server := newCycleShapeServer(t)

	for _, url := range server.shapeURLs() {
		loadWithin(t, fullStack(clients.NewCarpool(), database), url)
	}

	before := server.requestsWithPrefix("/")

	for _, url := range server.shapeURLs() {
		loadWithin(t, fullStack(clients.NewCarpool(), database), url)
	}

	require.Equal(t, before, server.requestsWithPrefix("/"))
}

// TestClientCycle_FullStack_ConcurrentMutualAuthors confirms that two full stacks sharing one
// Carpool, each loading one of two Notes that name each other as author, both return.
func TestClientCycle_FullStack_ConcurrentMutualAuthors(t *testing.T) {

	// BUG-212: each leader's author lookup would wait as a rider on the other's load, forever.
	database := newCycleTestDatabase(t)
	server := newCycleServer(t)
	server.setDelay(cycleDelay)
	firstURL := server.url("/notes/a")
	secondURL := server.url("/notes/b")

	server.serve("/notes/a", map[string]any{"id": firstURL, "type": "Note", "attributedTo": secondURL})
	server.serve("/notes/b", map[string]any{"id": secondURL, "type": "Note", "attributedTo": firstURL})

	carpool := clients.NewCarpool()
	newStack := func() streams.Client { return fullStack(carpool, database) }

	documents := loadConcurrently(t, newStack, firstURL, secondURL)

	require.Equal(t, firstURL, documents[0].ID())
	require.Equal(t, secondURL, documents[1].ID())
	require.Zero(t, untypedCacheEntries(t, database))
}

/******************************************
 * Helpers
 ******************************************/

// fullStack builds every layer of service.ActivityStream.Client, in the same order, with ascache over
// the given database and a rule checker that reads what the production checker reads.
func fullStack(carpool *clients.Carpool, database *mongo.Database) streams.Client {

	var server data.Server = mongodb.NewServer(database)

	// RULE: This MUST mirror service.ActivityStream.Client, including WithIgnoreHeaders, so that
	// the stub's no-store header cannot be what keeps it out of the cache.
	var client streams.Client = ascache.New(cycleLowerLayers(), nil, server, model.ActorTypeApplication, primitive.NilObjectID, "cycle.test", ascache.WithIgnoreHeaders())
	client = carpool.Client(client, cycleSigner)
	client = asrules.New(client, cycleRuleChecker)

	return ashash.New(client)
}

// cycleRuleChecker reads a document's rule keys as the production checker does, because reading them
// may load linked documents, and never hides anything
func cycleRuleChecker(uri string, document streams.Document) (metadata.LabelSet, error) {

	keys := model.ActorMatchKeys(uri)

	if document.NotNil() {
		keys = append(keys, model.DocumentMatchKeys(document)...)
	}

	// The keys are read only for the loads that reading them causes
	_ = keys

	return nil, nil
}

// newCycleShapeServer starts a cycleServer that serves every cycle shape from the tests above
func newCycleShapeServer(t *testing.T) *cycleServer {

	t.Helper()

	server := newCycleServer(t)

	server.serve("/notes/self", map[string]any{"id": server.url("/notes/self"), "type": "Note", "attributedTo": server.url("/notes/self")})
	server.serve("/notes/a", map[string]any{"id": server.url("/notes/a"), "type": "Note", "attributedTo": server.url("/notes/b")})
	server.serve("/notes/b", map[string]any{"id": server.url("/notes/b"), "type": "Note", "attributedTo": server.url("/notes/a")})
	server.serve("/activities/self", map[string]any{"id": server.url("/activities/self"), "type": "Create", "actor": server.url("/actors/alice"), "object": server.url("/activities/self")})
	server.serve("/actors/alice", cycleActor(server.url("/actors/alice")))

	keySelf := cycleActor(server.url("/actors/keyself"))
	keySelf["publicKey"] = server.url("/actors/keyself") + "#main-key"
	server.serve("/actors/keyself", keySelf)

	server.serveChain(cycleChainLength)

	return server
}

// shapeURLs returns the URL that starts each cycle shape served by newCycleShapeServer
func (server *cycleServer) shapeURLs() []string {
	return []string{
		server.url("/notes/self"),
		server.url("/notes/a"),
		server.url("/activities/self"),
		server.url("/actors/keyself"),
		server.url("/chain/0"),
	}
}

// untypedCacheEntries counts the cached documents whose type is missing, null, empty, or an empty list
func untypedCacheEntries(t *testing.T, database *mongo.Database) int64 {

	t.Helper()

	criteria := bson.M{"$or": []bson.M{
		{"object.type": nil},
		{"object.type": ""},
		{"object.type": bson.M{"$size": 0}},
	}}

	count, err := database.Collection("Document").CountDocuments(context.Background(), criteria)
	require.NoError(t, err)

	return count
}

// newCycleTestDatabase connects to a local replica set and returns a throwaway database carrying the
// production Document indexes, dropped when the test ends.  The test is skipped when none is reachable.
func newCycleTestDatabase(t *testing.T) *mongo.Database {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cycleTestConnection))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + cycleTestConnection)
	}

	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})

	if err := client.Ping(ctx, nil); err != nil {
		t.Skip("Skipping: no MongoDB at " + cycleTestConnection)
	}

	// RULE: The cache write runs in a transaction, which a standalone server refuses
	var hello bson.M
	if err := client.Database("admin").RunCommand(ctx, bson.M{"hello": 1}).Decode(&hello); err != nil || hello["setName"] == nil {
		t.Skip("Skipping: MongoDB at " + cycleTestConnection + " is not a replica set")
	}

	// A per-test database name keeps parallel runs, and the real cache, well clear of this
	database := client.Database("cycle_test_" + primitive.NewObjectID().Hex())

	t.Cleanup(func() {
		_ = database.Drop(context.Background())
	})

	require.NoError(t, querysync.Document(ctx, database))

	return database
}
