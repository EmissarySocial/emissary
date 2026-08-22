package config

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests cover the half of MongoStorage that keeps a CLUSTER in step: the supervised change
// stream that tells every other node a configuration has changed.  The behavior that matters is
// not "does one write land in one document" -- it is "does a node still see changes after the
// change stream dies," which is the failure that used to require a reboot.
//
// The unit tests below need no database.  The integration tests do, and they skip cleanly when
// none is reachable, so `go test ./...` still passes on a machine without one.

/******************************************
 * Unit Tests (no database required)
 ******************************************/

// newUnitTestStorage returns a MongoStorage with no database behind it.  It is enough for the
// pure logic -- the update channel, the backoff, and decoding an event that carries its own
// document -- none of which touch the collection.
func newUnitTestStorage() MongoStorage {

	done, cancelFunc := context.WithCancel(context.Background())

	return MongoStorage{
		source:        ConfigSourceCommandLine,
		location:      "mongodb://localhost:27017/emissary",
		updateChannel: newUpdateChannel(),
		done:          done,
		cancelFunc:    cancelFunc,
	}
}

// TestConfigFromEvent_UsesFullDocument pins the fast path: the event carries the exact version of
// the document that the event describes, so the reload uses it directly.  Re-querying instead
// would race a concurrent write and could apply a DIFFERENT version than the one we were told
// about -- and since no second event is coming, that wrong version would stick.
func TestConfigFromEvent_UsesFullDocument(t *testing.T) {

	storage := newUnitTestStorage()
	defer storage.Close()

	original := NewConfig()
	original.AdminEmail = "admin@example.com"
	original.HTTPPort = 8080
	original.MongoID = primitive.NewObjectID()

	document, err := bson.Marshal(original)
	require.Nil(t, err)

	event := makeChangeEvent(t, "replace", document)

	result, ok := storage.configFromEvent(event)

	require.True(t, ok)
	require.Equal(t, "admin@example.com", result.AdminEmail)
	require.Equal(t, 8080, result.HTTPPort)
	require.Equal(t, original.MongoID, result.MongoID)
}

// TestConfigFromEvent_DecoratesNodeLocalFields pins that Source and Location survive the fast
// path.  They describe where THIS process found its configuration, are deliberately never stored,
// and so must be re-stamped onto every value that leaves storage -- by both read paths, not just
// by load().
func TestConfigFromEvent_DecoratesNodeLocalFields(t *testing.T) {

	storage := newUnitTestStorage()
	defer storage.Close()

	document, err := bson.Marshal(NewConfig())
	require.Nil(t, err)

	result, ok := storage.configFromEvent(makeChangeEvent(t, "replace", document))

	require.True(t, ok)
	require.Equal(t, storage.source, result.Source)
	require.Equal(t, storage.location, result.Location)
}

// TestConfigFromEvent_InitializesCollections pins that the fast path starts from NewConfig(), so
// a stored document that omits a collection still arrives with a usable (empty, non-nil) one.
// Decoding into a bare Config would leave nil maps for every caller downstream to trip over.
func TestConfigFromEvent_InitializesCollections(t *testing.T) {

	storage := newUnitTestStorage()
	defer storage.Close()

	// A minimal document, as if someone hand-wrote it into MongoDB
	document, err := bson.Marshal(bson.M{"_id": primitive.NewObjectID()})
	require.Nil(t, err)

	result, ok := storage.configFromEvent(makeChangeEvent(t, "replace", document))

	require.True(t, ok)
	require.NotNil(t, result.Domains)
	require.NotNil(t, result.Templates)
	require.NotNil(t, result.AttachmentOriginals)
	require.NotNil(t, result.AttachmentCache)
	require.NotNil(t, result.ExportCache)
	require.NotNil(t, result.Certificates)
	require.NotNil(t, result.ActivityPubCache)
	require.NotNil(t, result.Loggers)
}

// mustWrite persists a configuration and returns it AS STORED (revision incremented), so the
// caller's next write carries the right compare-and-swap base.
func mustWrite(t *testing.T, storage MongoStorage, config Config) Config {

	t.Helper()

	stored, err := storage.Write(config)
	require.NoError(t, err)

	return stored
}

// makeChangeEvent builds the change stream event envelope that configFromEvent reads.
func makeChangeEvent(t *testing.T, operationType string, fullDocument []byte) bson.Raw {

	t.Helper()

	event := bson.M{"operationType": operationType}

	if fullDocument != nil {
		event["fullDocument"] = bson.Raw(fullDocument)
	}

	result, err := bson.Marshal(event)
	require.Nil(t, err)

	return result
}

/******************************************
 * Integration Tests (require MongoDB)
 ******************************************/

// mongoTestConnection is the local MongoDB that these tests require.  Change streams need a
// replica set, so a standalone mongod skips the watcher tests below.
const mongoTestConnection = "mongodb://localhost:27017/?directConnection=true"

// TestMongoStorage_WriteAndLoadRoundTrip pins that what Write stores is what load returns.  Every
// other test here assumes this, and it is the one place the BSON encoding of Config is exercised.
func TestMongoStorage_WriteAndLoadRoundTrip(t *testing.T) {

	storage := newIntegrationTestStorage(t)

	original := DefaultConfig()
	original.AdminEmail = "owner@example.com"
	original.HTTPPort = 9999
	original.DebugLevel = "Trace"
	original.ActivityPubCache = map[string]string{"connectString": mongoTestConnection, "database": "cache"}

	original = mustWrite(t, storage, original)

	result, err := storage.load(context.Background())

	require.Nil(t, err)
	require.Equal(t, "owner@example.com", result.AdminEmail)
	require.Equal(t, 9999, result.HTTPPort)
	require.Equal(t, "Trace", result.DebugLevel)
	require.Equal(t, "cache", result.ActivityPubCache.GetString("database"))

	// The node-local fields are stamped on by storage, never read from the document
	require.Equal(t, storage.source, result.Source)
	require.Equal(t, storage.location, result.Location)
}

// TestMongoStorage_LoadNotFound pins the sentinel that NewMongoStorage branches on to decide
// whether to bootstrap a default configuration or refuse to start.
func TestMongoStorage_LoadNotFound(t *testing.T) {

	storage := newIntegrationTestStorage(t)

	_, err := storage.load(context.Background())

	require.NotNil(t, err)
	require.True(t, derp.IsNotFound(err), "an empty collection must read as NotFound")
}

// TestMongoStorage_LoadIsDeterministicWithMultipleDocuments pins the sort.  An unsorted FindOne
// returns whichever document the server reaches first, which can differ between nodes in a
// cluster -- so two servers would run different configurations forever, with nothing to reconcile
// them.  Sorting by _id at least guarantees they agree on which document is THE configuration.
func TestMongoStorage_LoadIsDeterministicWithMultipleDocuments(t *testing.T) {

	storage := newIntegrationTestStorage(t)

	// Deliberately out of _id order, so an unsorted read would likely return the second one
	second := DefaultConfig()
	second.MongoID = primitive.NewObjectIDFromTimestamp(time.Now().Add(1 * time.Hour))
	second.AdminEmail = "second@example.com"
	second = mustWrite(t, storage, second)

	first := DefaultConfig()
	first.MongoID = primitive.NewObjectIDFromTimestamp(time.Now().Add(-1 * time.Hour))
	first.AdminEmail = "first@example.com"
	first = mustWrite(t, storage, first)

	for range 5 {
		result, err := storage.load(context.Background())
		require.Nil(t, err)
		require.Equal(t, "first@example.com", result.AdminEmail)
	}
}

// TestMongoStorage_LoadWithOperationTime pins that the initial read reports its cluster time.
// That timestamp is what closes the gap between reading the configuration and opening the change
// stream; without it, a write landing in that window is never seen. (See the startAt test below.)
func TestMongoStorage_LoadWithOperationTime(t *testing.T) {

	storage := newIntegrationTestStorage(t)

	mustWrite(t, storage, DefaultConfig())

	result, operationTime, err := storage.loadWithOperationTime(context.Background())

	require.Nil(t, err)
	require.Equal(t, DefaultConfig().HTTPPort, result.HTTPPort)
	require.NotNil(t, operationTime, "a replica set must report an operation time for the read")
}

// TestMongoStorage_WatchPropagatesChanges is the base case for a cluster: another node writes the
// configuration, and this node is told about it without being restarted.
func TestMongoStorage_WatchPropagatesChanges(t *testing.T) {

	storage := newIntegrationTestStorage(t)
	requireChangeStreams(t, storage)

	config := DefaultConfig()
	config = mustWrite(t, storage, config)

	go storage.watch(nil)

	result := writeAndAwait(t, storage, config, "changed", 20*time.Second)
	require.Contains(t, result.AdminEmail, "changed")
}

// TestMongoStorage_WatchStartsAtOperationTime pins the boot-gap fix.  A configuration written
// between "load the config" and "open the change stream" must still be delivered.  The test makes
// that window enormous -- it writes while no watcher exists at all -- and then starts watching
// from the timestamp of the original read.
func TestMongoStorage_WatchStartsAtOperationTime(t *testing.T) {

	storage := newIntegrationTestStorage(t)
	requireChangeStreams(t, storage)

	config := DefaultConfig()
	config = mustWrite(t, storage, config)

	// Read the configuration, capturing the cluster time -- exactly what NewMongoStorage does
	_, operationTime, err := storage.loadWithOperationTime(context.Background())
	require.Nil(t, err)
	require.NotNil(t, operationTime)

	// Now write a change while NOTHING is watching. This is the window that used to lose writes.
	config.AdminEmail = "written-in-the-gap@example.com"
	config = mustWrite(t, storage, config)

	go storage.watch(operationTime)

	// The wait tolerates earlier versions arriving first.  A read's operation time is the cluster
	// time of the last write it saw, so starting there replays that write as well.  Erring toward
	// re-delivering a configuration the node already has is exactly the right trade: reloads are
	// idempotent, and the alternative -- starting a moment later -- is how the write got lost.
	result := awaitConfig(t, storage, 20*time.Second, func(value Config) bool {
		return value.AdminEmail == "written-in-the-gap@example.com"
	})

	require.Equal(t, "written-in-the-gap@example.com", result.AdminEmail)
}

// TestMongoStorage_WatchRecoversFromInvalidate is the regression test for the bug that made
// configuration changes "need a reboot".
//
// Dropping the watched collection ends the change stream with an `invalidate` event, and the
// driver surfaces that as Next() == false with Err() == nil -- no error, nothing logged.  The old
// single-shot loop simply returned there, and that node never saw another configuration change
// for the life of the process.  The supervised loop must reopen and keep delivering.
func TestMongoStorage_WatchRecoversFromInvalidate(t *testing.T) {

	storage := newIntegrationTestStorage(t)
	requireChangeStreams(t, storage)

	config := DefaultConfig()
	config = mustWrite(t, storage, config)

	go storage.watch(nil)

	// Prove the stream is live before breaking it
	writeAndAwait(t, storage, config, "before-the-drop", 20*time.Second)

	// Kill the change stream the way the server kills it: drop the collection out from under it
	require.Nil(t, storage.collection.Drop(context.Background()))

	// Everything after this point would be lost forever under the old implementation.  The marker
	// prefix is what makes the assertion honest: a configuration left over from before the drop
	// can still be sitting in the channel, and it must NOT be mistaken for recovery.
	result := writeAndAwait(t, storage, config, "after-the-drop", 60*time.Second)
	require.Contains(t, result.AdminEmail, "after-the-drop")
}

// TestMongoStorage_CloseStopsTheWatcher pins that the supervised loop is still stoppable.  A loop
// that cannot end except by cancellation is only correct if cancellation actually ends it.
func TestMongoStorage_CloseStopsTheWatcher(t *testing.T) {

	storage := newIntegrationTestStorage(t)
	requireChangeStreams(t, storage)

	mustWrite(t, storage, DefaultConfig())

	finished := make(chan struct{})

	go func() {
		defer close(finished)
		storage.watch(nil)
	}()

	// Give the watcher time to open its stream, so Close interrupts a LIVE stream
	time.Sleep(500 * time.Millisecond)
	storage.Close()

	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("watch() did not return after Close()")
	}
}

/******************************************
 * Integration Test Helpers
 ******************************************/

// newIntegrationTestStorage builds a MongoStorage against a throwaway database on the local
// MongoDB, and skips the test when no database is reachable.  It deliberately does NOT call
// NewMongoStorage, which exits the process on failure and is therefore untestable.
func newIntegrationTestStorage(t *testing.T) MongoStorage {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := CommandLineArgs{
		Source:     ConfigSourceCommandLine,
		Location:   mongoTestConnection,
		Database:   "emissary_configtest_" + primitive.NewObjectID().Hex(),
		Collection: DefaultConfigCollection,
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(args.Location))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + mongoTestConnection)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Skip("Skipping: no MongoDB at " + mongoTestConnection)
	}

	done, cancelFunc := context.WithCancel(context.Background())

	storage := MongoStorage{
		source:        args.Source,
		location:      args.Location,
		collection:    client.Database(args.Database).Collection(args.Collection),
		updateChannel: newUpdateChannel(),
		done:          done,
		cancelFunc:    cancelFunc,
	}

	t.Cleanup(func() {
		storage.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = client.Database(args.Database).Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	return storage
}

// requireChangeStreams skips the test when the server cannot open a change stream, which is the
// case on a standalone mongod.  Worth naming out loud: a standalone server means configuration
// NEVER propagates between nodes, so a cluster must not be run on one.
func requireChangeStreams(t *testing.T, storage MongoStorage) {

	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	changeStream, err := storage.collection.Watch(ctx, mongo.Pipeline{})

	if err != nil {
		t.Skip("Skipping: MongoDB change streams unavailable (not a replica set): " + err.Error())
	}

	_ = changeStream.Close(ctx)
}

// awaitConfig drains published configurations until one satisfies `match`, and fails if none
// does before the timeout.
//
// RULE: Tests here must wait for the configuration they EXPECT, never assert on whichever one
// arrives first.  Two properties of the design make anything else flaky, and both are deliberate:
// the update channel keeps only the newest entry (so an older one may or may not still be there),
// and a restarted stream re-reads the configuration (so the same version can arrive twice).
func awaitConfig(t *testing.T, storage MongoStorage, timeout time.Duration, match func(Config) bool) Config {

	t.Helper()

	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); {

		select {

		case result := <-storage.Subscribe():
			if match(result) {
				return result
			}

		case <-time.After(100 * time.Millisecond):
		}
	}

	t.Fatal("timed out waiting for the expected configuration update")
	return Config{}
}

// writeAndAwait writes copies of `config` until the watcher delivers one back, and returns the
// delivered value.
//
// Each attempt stamps a DISTINCT admin email built from `marker`, for two reasons.  A change
// stream only reports changes made while it is open, and the watcher opens asynchronously, so the
// first write can legitimately land before anything is listening -- but MongoDB emits no event
// for a replacement that changes nothing, so a plain retry of the same document would be silent.
// The shared prefix then lets the wait recognize this phase's configuration and discard any left
// over from an earlier one.
func writeAndAwait(t *testing.T, storage MongoStorage, config Config, marker string, timeout time.Duration) Config {

	t.Helper()

	for deadline, attempt := time.Now().Add(timeout), 0; time.Now().Before(deadline); {

		attempt++
		config.AdminEmail = fmt.Sprintf("%s-%d@example.com", marker, attempt)
		config = mustWrite(t, storage, config)

		// Watch for a delivery from this phase before writing again
		for waitUntil := time.Now().Add(2 * time.Second); time.Now().Before(waitUntil); {

			select {

			case result := <-storage.Subscribe():
				if strings.HasPrefix(result.AdminEmail, marker) {
					return result
				}

			case <-time.After(100 * time.Millisecond):
			}
		}
	}

	t.Fatal("the change stream never delivered a configuration for marker: " + marker)
	return Config{}
}

// TestNewMongoStorage_MalformedURIErrs pins the one constructor failure reachable without a
// server: mongo.Connect is lazy, so only a URI it cannot parse fails at construction.  It must
// come back as an error for main -- never an exit from inside this package.
func TestNewMongoStorage_MalformedURIErrs(t *testing.T) {

	args := CommandLineArgs{
		Source:     ConfigSourceCommandLine,
		Location:   "mongodb://user:pass@[not-a-host/%%%",
		Database:   DefaultConfigDatabase,
		Collection: DefaultConfigCollection,
	}

	_, err := NewMongoStorage(&args)
	require.Error(t, err)
}

// TestMongoStorage_WriteRejectsStaleRevision is the regression test for the lost-update race:
// two nodes saving from the same base.  The second write must be REJECTED with a 409 -- before
// the revision guard it silently replaced the first write, destroying changes that could
// include a domain's MasterKey, stored nowhere else.
func TestMongoStorage_WriteRejectsStaleRevision(t *testing.T) {

	storage := newIntegrationTestStorage(t)

	base := DefaultConfig()
	base.AdminEmail = "base@example.com"

	// Node A saves from the base
	fromA := base
	fromA.AdminEmail = "node-a@example.com"
	mustWrite(t, storage, fromA)

	// Node B saves from the SAME base (its revision is now stale)
	fromB := base
	fromB.AdminEmail = "node-b@example.com"

	_, err := storage.Write(fromB)

	require.Error(t, err)
	require.True(t, derp.IsConflict(err), "a stale write must surface as a 409")

	// Node A's write survived untouched
	result, err := storage.load(context.Background())
	require.NoError(t, err)
	require.Equal(t, "node-a@example.com", result.AdminEmail)
}

// TestMongoStorage_WriteThreadsRevisions pins the happy path across generations: each write
// bases on the PREVIOUS stored revision, and each succeeds.
func TestMongoStorage_WriteThreadsRevisions(t *testing.T) {

	storage := newIntegrationTestStorage(t)

	config := DefaultConfig()

	for index := range 3 {
		config.AdminEmail = fmt.Sprintf("save-%d@example.com", index)
		config = mustWrite(t, storage, config)
		require.Equal(t, int64(index+1), config.Revision)
	}

	result, err := storage.load(context.Background())
	require.NoError(t, err)
	require.Equal(t, "save-2@example.com", result.AdminEmail)
	require.Equal(t, int64(3), result.Revision)
}

// TestMongoStorage_WriteAcceptsLegacyDocument pins the migration leg: a document written before
// revisions existed has no `revision` field at all.  A caller holding Revision 0 must be able to
// replace it -- the first conditional write stamps the field, and the legacy leg never matches
// again.
func TestMongoStorage_WriteAcceptsLegacyDocument(t *testing.T) {

	storage := newIntegrationTestStorage(t)

	// A legacy document: hand-written, no revision field
	legacyID := primitive.NewObjectID()
	_, err := storage.collection.InsertOne(context.Background(), bson.M{
		"_id":        legacyID,
		"adminEmail": "legacy@example.com",
	})
	require.NoError(t, err)

	// Load it the way a booting node would: Revision arrives as 0
	loaded, err := storage.load(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(0), loaded.Revision)

	// The first guarded write succeeds and stamps the revision...
	loaded.AdminEmail = "upgraded@example.com"
	stored := mustWrite(t, storage, loaded)
	require.Equal(t, int64(1), stored.Revision)

	// ...and a second write from the legacy base now conflicts: the $exists:false leg is gone
	stale := loaded
	stale.AdminEmail = "stale@example.com"

	_, err = storage.Write(stale)
	require.Error(t, err)
	require.True(t, derp.IsConflict(err))
}
