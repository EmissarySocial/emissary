package queries

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/******************************************
 * Supervised Change Stream Tests
 *
 * These pin the supervisor that every watcher runs inside.
 * What matters is not "does one event arrive" but "does the
 * watcher still deliver after the server closes its stream".
 *
 * Not covered: run's return on a standalone server (these
 * tests need a replica set), and runOnce's nil-stream guard
 * (the driver never returns a nil stream without an error).
 ******************************************/

/******************************************
 * Unit Tests (no database required)
 ******************************************/

// TestWatchRetryDelay pins the backoff between attempts to reopen a change stream
func TestWatchRetryDelay(t *testing.T) {

	test := func(failures int, expected time.Duration) {
		t.Helper()
		require.Equal(t, expected, watchRetryDelay(failures), "failures=%d", failures)
	}

	test(-1, time.Second)
	test(0, time.Second)
	test(1, time.Second)
	test(2, 2*time.Second)
	test(3, 4*time.Second)
	test(6, 32*time.Second)
	test(7, 60*time.Second)
	test(8, 60*time.Second)
	test(1_000_000, 60*time.Second)
}

// TestFullDocument pins which change events carry a document that a watcher can use
func TestFullDocument(t *testing.T) {

	t.Run("Document", func(t *testing.T) {
		event := mustMarshal(t, bson.M{"operationType": "insert", "fullDocument": bson.M{"name": "Groot"}})

		document, ok := fullDocument(event)

		require.True(t, ok)
		require.Equal(t, "Groot", document.Lookup("name").StringValue())
	})

	t.Run("Missing", func(t *testing.T) {
		event := mustMarshal(t, bson.M{"operationType": "update"})

		document, ok := fullDocument(event)

		require.False(t, ok)
		require.Nil(t, document)
	})

	t.Run("Null", func(t *testing.T) {
		// UpdateLookup reports null when the document was deleted before the lookup ran
		event := mustMarshal(t, bson.M{"operationType": "update", "fullDocument": nil})

		_, ok := fullDocument(event)
		require.False(t, ok)
	})

	t.Run("NotADocument", func(t *testing.T) {
		event := mustMarshal(t, bson.M{"operationType": "insert", "fullDocument": "I am Groot"})

		_, ok := fullDocument(event)
		require.False(t, ok)
	})

	t.Run("EmptyEvent", func(t *testing.T) {
		_, ok := fullDocument(bson.Raw{})
		require.False(t, ok)
	})
}

// TestIsNotReplicaSet pins the one error that stops a watcher for good
func TestIsNotReplicaSet(t *testing.T) {

	standalone := mongo.CommandError{Code: mongoErrorNotReplicaSet, Message: "The $changeStream stage is only supported on replica sets"}

	require.True(t, isNotReplicaSet(standalone))
	require.True(t, isNotReplicaSet(fmt.Errorf("wrapped: %w", standalone)))
	require.True(t, isNotReplicaSet(derp.Wrap(standalone, "TestIsNotReplicaSet", "Opening change stream")))
	require.False(t, isNotReplicaSet(mongo.CommandError{Code: 286, Message: "ChangeStreamHistoryLost"}))
	require.False(t, isNotReplicaSet(errors.New("connection refused")))
	require.False(t, isNotReplicaSet(nil))
}

// TestChangeWatcher_NotMongoDB pins that a watcher on a non-MongoDB database returns at once
func TestChangeWatcher_NotMongoDB(t *testing.T) {

	watcher := changeWatcher{
		collection: "Thing",
		onDocument: func(context.Context, bson.Raw) {},
	}

	requireReturns(t, func() { watcher.run(context.Background(), mockdb.New()) })
}

// TestChangeWatcher_SessionError pins that a watcher whose session cannot open returns at once
func TestChangeWatcher_SessionError(t *testing.T) {

	watcher := changeWatcher{
		collection: "Thing",
		onDocument: func(context.Context, bson.Raw) {},
	}

	requireReturns(t, func() { watcher.run(context.Background(), failingServer{}) })
}

/******************************************
 * Integration Tests (require a MongoDB replica set)
 ******************************************/

// TestChangeWatcher_DeliversDocuments pins the basic contract: an insert reaches onDocument
func TestChangeWatcher_DeliversDocuments(t *testing.T) {

	database := newWatchTestDatabase(t)
	probe := startTestWatcher(t, database, "")

	probe.awaitOpen(t, 1)
	insertThing(t, database, "first")

	require.Equal(t, "first", probe.awaitDocument(t))
}

// TestChangeWatcher_RecoversFromInvalidate is the regression test for watchers that stopped for
// good.  Dropping the collection ends the stream with Next() == false and Err() == nil, and the
// old loops returned there silently.
func TestChangeWatcher_RecoversFromInvalidate(t *testing.T) {

	database := newWatchTestDatabase(t)
	probe := startTestWatcher(t, database, "")

	// Prove the stream is live before breaking it
	probe.awaitOpen(t, 1)
	insertThing(t, database, "before-the-drop")
	require.Equal(t, "before-the-drop", probe.awaitDocument(t))

	// Kill the stream the way the server kills it
	require.Nil(t, database.Collection("Thing").Drop(context.Background()))

	// Everything after this point was lost forever under the old loops
	probe.awaitOpen(t, 2)
	insertThing(t, database, "after-the-drop")
	require.Equal(t, "after-the-drop", probe.awaitDocument(t))
}

// TestChangeWatcher_DeliversEventsWrittenWhileReopening pins the resume token: an event written
// after the stream closed, but before it reopened, still arrives.
func TestChangeWatcher_DeliversEventsWrittenWhileReopening(t *testing.T) {

	database := newWatchTestDatabase(t)
	probe := startTestWatcher(t, database, "")

	probe.awaitOpen(t, 1)
	insertThing(t, database, "before-the-drop")
	require.Equal(t, "before-the-drop", probe.awaitDocument(t))

	// The backoff is at least a second, so this insert lands while no stream is open
	require.Nil(t, database.Collection("Thing").Drop(context.Background()))
	insertThing(t, database, "during-the-gap")

	require.Equal(t, "during-the-gap", probe.awaitDocument(t))
}

// TestChangeWatcher_UpdateLookup pins that UpdateLookup turns a $set into a delivered document
func TestChangeWatcher_UpdateLookup(t *testing.T) {

	database := newWatchTestDatabase(t)
	thingID := insertThing(t, database, "original")

	probe := startTestWatcher(t, database, options.UpdateLookup)
	probe.awaitOpen(t, 1)

	setThingName(t, database, thingID, "updated")

	require.Equal(t, "updated", probe.awaitDocument(t))
}

// TestChangeWatcher_PlainUpdateCarriesNoDocument pins the default: without UpdateLookup, a $set
// delivers nothing, so only inserts and replacements reach onDocument.
func TestChangeWatcher_PlainUpdateCarriesNoDocument(t *testing.T) {

	database := newWatchTestDatabase(t)
	thingID := insertThing(t, database, "original")

	probe := startTestWatcher(t, database, "")
	probe.awaitOpen(t, 1)

	// The $set happens first, so if it delivered anything, it would arrive first
	setThingName(t, database, thingID, "updated")
	insertThing(t, database, "inserted")

	require.Equal(t, "inserted", probe.awaitDocument(t))
}

// TestChangeWatcher_RetriesFailedResync pins that a failed onOpen closes the stream and tries
// again, instead of carrying on with a view that may have missed events.
func TestChangeWatcher_RetriesFailedResync(t *testing.T) {

	database := newWatchTestDatabase(t)
	probe := newWatchProbe()
	probe.failFirstOpen = true

	probe.start(t, database, "")

	// The first resync fails, so the first SUCCESSFUL open is the second attempt
	probe.awaitOpen(t, 1)
	require.Equal(t, int32(2), probe.openCalls.Load())

	insertThing(t, database, "after-the-retry")
	require.Equal(t, "after-the-retry", probe.awaitDocument(t))
}

// TestChangeWatcher_StopsOnCancel pins that the supervised loop still ends when it is canceled
func TestChangeWatcher_StopsOnCancel(t *testing.T) {

	database := newWatchTestDatabase(t)
	probe := newWatchProbe()
	ctx, cancel := context.WithCancel(context.Background())

	finished := make(chan struct{})

	go func() {
		defer close(finished)
		probe.watcher().run(ctx, mongodb.NewServer(database))
	}()

	// Cancel a LIVE stream, not one that has not opened yet
	probe.awaitOpen(t, 1)
	cancel()

	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("run() did not return after its context was canceled")
	}
}

// TestChangeWatcher_FailedOpenDropsResumeToken pins that a stream which cannot open forgets its
// resume token, so a token the server refuses is never retried forever.
func TestChangeWatcher_FailedOpenDropsResumeToken(t *testing.T) {

	database := newWatchTestDatabase(t)
	probe := newWatchProbe()

	// A canceled context is the simplest reliable way to make Watch() fail
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resumeToken := mustMarshal(t, bson.M{"_data": "not-a-real-token"})
	progressed, err := probe.watcher().runOnce(ctx, database.Collection("Thing"), &resumeToken)

	require.Error(t, err)
	require.False(t, progressed)
	require.Nil(t, resumeToken)
	require.Zero(t, probe.openCalls.Load())
}

/******************************************
 * Test Helpers
 ******************************************/

// watchTestConnection is the local MongoDB that the watcher tests require
const watchTestConnection = "mongodb://localhost:27017/?directConnection=true"

// watchTestTimeout bounds every wait.  A reopen waits out the backoff, so it must exceed one second.
const watchTestTimeout = 20 * time.Second

// newWatchTestDatabase returns a throwaway database on the local MongoDB, and skips the test when
// no database is reachable or it cannot open change streams (a standalone server).
func newWatchTestDatabase(t *testing.T) *mongo.Database {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(watchTestConnection))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + watchTestConnection)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx) // nothing to clean up on a server that never answered
		t.Skip("Skipping: no MongoDB at " + watchTestConnection)
	}

	database := client.Database("emissary_watchtest_" + primitive.NewObjectID().Hex())

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = database.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	// Change streams need a replica set
	changeStream, err := database.Collection("Thing").Watch(ctx, mongo.Pipeline{})

	if err != nil {
		t.Skip("Skipping: MongoDB change streams unavailable (not a replica set): " + err.Error())
	}

	_ = changeStream.Close(ctx)

	return database
}

// watchProbe records what a test watcher sees, so tests can wait for it
type watchProbe struct {
	opens         atomic.Int32  // how many times the stream has opened successfully
	openCalls     atomic.Int32  // how many times onOpen has been called
	failFirstOpen bool          // makes the first onOpen fail
	documents     chan string   // the `name` of every document delivered
	opened        chan struct{} // signaled on every successful open
}

// newWatchProbe returns an empty watchProbe
func newWatchProbe() *watchProbe {
	return &watchProbe{
		documents: make(chan string, 100),
		opened:    make(chan struct{}, 100),
	}
}

// startTestWatcher runs a watcher on the "Thing" collection until the test ends
func startTestWatcher(t *testing.T, database *mongo.Database, lookup options.FullDocument) *watchProbe {

	t.Helper()

	probe := newWatchProbe()
	probe.start(t, database, lookup)

	return probe
}

// watcher returns a changeWatcher on the "Thing" collection that reports to this probe
func (probe *watchProbe) watcher() changeWatcher {

	return changeWatcher{
		collection: "Thing",
		onOpen: func(context.Context, *mongo.Collection) error {

			if probe.openCalls.Add(1) == 1 && probe.failFirstOpen {
				return errors.New("the first resync fails on purpose")
			}

			probe.opens.Add(1)
			probe.opened <- struct{}{}
			return nil
		},
		onDocument: func(_ context.Context, document bson.Raw) {

			// StringValueOK, because StringValue panics on a missing name and would take down the test binary
			name, _ := document.Lookup("name").StringValueOK()
			probe.documents <- name
		},
	}
}

// start runs this probe's watcher until the test ends, and fails the test if it does not stop
func (probe *watchProbe) start(t *testing.T, database *mongo.Database, lookup options.FullDocument) {

	t.Helper()

	watcher := probe.watcher()
	watcher.fullDocument = lookup

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		watcher.run(ctx, mongodb.NewServer(database))
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(10 * time.Second):
			t.Error("watcher did not stop after its context was canceled")
		}
	})
}

// awaitOpen waits until the stream has opened successfully `count` times
func (probe *watchProbe) awaitOpen(t *testing.T, count int32) {

	t.Helper()

	for deadline := time.After(watchTestTimeout); probe.opens.Load() < count; {
		select {
		case <-probe.opened:
		case <-deadline:
			t.Fatalf("change stream opened %d times, want %d", probe.opens.Load(), count)
		}
	}
}

// awaitDocument returns the `name` of the next document delivered
func (probe *watchProbe) awaitDocument(t *testing.T) string {

	t.Helper()

	select {
	case result := <-probe.documents:
		return result
	case <-time.After(watchTestTimeout):
		t.Fatal("no document was delivered")
		return ""
	}
}

// insertThing inserts a document into the "Thing" collection and returns its ID
func insertThing(t *testing.T, database *mongo.Database, name string) primitive.ObjectID {

	t.Helper()

	thingID := primitive.NewObjectID()
	_, err := database.Collection("Thing").InsertOne(context.Background(), bson.M{"_id": thingID, "name": name})
	require.Nil(t, err)

	return thingID
}

// setThingName changes a document's name with a $set, which produces an "update" event
func setThingName(t *testing.T, database *mongo.Database, thingID primitive.ObjectID, name string) {

	t.Helper()

	_, err := database.Collection("Thing").UpdateOne(context.Background(), bson.M{"_id": thingID}, bson.M{"$set": bson.M{"name": name}})
	require.Nil(t, err)
}

// mustMarshal encodes a value as BSON, failing the test if it cannot
func mustMarshal(t *testing.T, value any) bson.Raw {

	t.Helper()

	result, err := bson.Marshal(value)
	require.Nil(t, err)

	return result
}

// requireReturns fails the test unless fn returns within a few seconds
func requireReturns(t *testing.T, fn func()) {

	t.Helper()

	finished := make(chan struct{})

	go func() {
		defer close(finished)
		fn()
	}()

	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("function did not return")
	}
}

// failingServer is a data.Server whose sessions never open
type failingServer struct{}

// Session implements the data.Server interface, and always fails
func (failingServer) Session(context.Context) (data.Session, error) {
	return nil, errors.New("the database is on fire")
}

// WithTransaction implements the data.Server interface, and always fails
func (failingServer) WithTransaction(context.Context, data.TransactionCallbackFunc) (any, error) {
	return nil, errors.New("the database is still on fire")
}
