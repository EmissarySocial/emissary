package server

import (
	"context"
	"testing"
	"time"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// These tests pin the config-reload lifecycle of the two server-level resources that domain
// factories depend on: the common (ActivityPub Cache) database client and the task queue.
//
// The incident they guard against: every config save re-ran refreshCommonDatabase and
// refreshQueue unconditionally.  The old mongo client was disconnected and the old queue was
// stopped, but existing domain factories held captured handles to both -- so every ActivityStream
// cache call failed with "client is disconnected" (which broke inbound HTTP-signature
// verification with a bare 401), and every queued task was handed to a stopped queue and silently
// dropped ("Turbine Queue: stopped").
//
// None of these tests need a reachable Mongo server: mongo.Connect is lazy (it never contacts the
// server), connected-vs-disconnected is client-side state, and Ping on a DISCONNECTED client
// short-circuits with mongo.ErrClientDisconnected before server selection.  The URIs point at
// closed ports with a short serverSelectionTimeoutMS so that any accidental real I/O fails fast
// with a DIFFERENT error than the ones asserted here.

// lifecycleConnection returns a FRESH connection map for each call.  Freshness matters: the
// unchanged-guard must compare VALUES, never map identity, because config handlers rebuild and
// mutate these maps in place (see the snapshot-scalars comment on factoryCore).
func lifecycleConnection(port string, database string) mapof.String {
	return mapof.String{
		"connectString": "mongodb://127.0.0.1:" + port + "/?directConnection=true&serverSelectionTimeoutMS=200",
		"database":      database,
	}
}

// lazyDatabase returns a *mongo.Database backed by a client that has never contacted a server.
// The cleanup disconnects it (tolerating a test that already disconnected it).
func lazyDatabase(t *testing.T, database string) *mongo.Database {
	t.Helper()

	client, err := mongo.Connect(context.Background(),
		options.Client().ApplyURI("mongodb://127.0.0.1:59997/?directConnection=true&serverSelectionTimeoutMS=200"))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})

	return client.Database(database)
}

// requireDisconnected asserts that the client has been disconnected: Ping on a disconnected
// client returns mongo.ErrClientDisconnected IMMEDIATELY (closed topology short-circuits before
// server selection).  A still-connected client would instead time out server selection (~200ms
// per the test URI) with a different error, failing this assertion with a clear diff.
func requireDisconnected(t *testing.T, client *mongo.Client) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := client.Ping(ctx, readpref.Primary())
	require.ErrorIs(t, err, mongo.ErrClientDisconnected)
}

// requireConnected asserts that the client is still connected, by disconnecting it: the FIRST
// Disconnect of a live client returns nil, while a client someone already disconnected returns
// mongo.ErrClientDisconnected.  Use this only as a test's final assertion/cleanup on a client.
func requireConnected(t *testing.T, client *mongo.Client) {
	t.Helper()
	require.NoError(t, client.Disconnect(context.Background()))
}

/******************************************
 * refreshCommonDatabase
 ******************************************/

// TestRefreshCommonDatabase_UnchangedSettingsKeepClient pins the reload guard: reloading a config
// whose database settings are unchanged must KEEP the live client.  Reconnecting would
// disconnect the old client and strand every captured handle (the original incident).
func TestRefreshCommonDatabase_UnchangedSettingsKeepClient(t *testing.T) {

	factory := factoryCore{}

	require.NoError(t, factory.refreshCommonDatabase(lifecycleConnection("59999", "lifecycle-a")))
	first := factory.commonDatabase
	require.NotNil(t, first)

	// Reload with a FRESH map carrying equal values (value comparison, not map identity)
	require.NoError(t, factory.refreshCommonDatabase(lifecycleConnection("59999", "lifecycle-a")))

	require.Same(t, first, factory.commonDatabase, "unchanged settings must keep the same database handle")
	requireConnected(t, first.Client())
}

// TestRefreshCommonDatabase_ChangedDatabaseNameReconnects pins the other half of the guard: when
// the settings DO change, the connection must be replaced and the old client disconnected.
func TestRefreshCommonDatabase_ChangedDatabaseNameReconnects(t *testing.T) {

	factory := factoryCore{}

	require.NoError(t, factory.refreshCommonDatabase(lifecycleConnection("59999", "lifecycle-a")))
	oldClient := factory.commonDatabase.Client()

	require.NoError(t, factory.refreshCommonDatabase(lifecycleConnection("59999", "lifecycle-b")))

	require.Equal(t, "lifecycle-b", factory.commonDatabase.Name())
	require.NotSame(t, oldClient, factory.commonDatabase.Client(), "changed settings must build a new client")
	requireDisconnected(t, oldClient)
	requireConnected(t, factory.commonDatabase.Client())
}

// TestRefreshCommonDatabase_ChangedConnectStringReconnects covers the connectString leg of the
// comparison (same database name, different server address).
func TestRefreshCommonDatabase_ChangedConnectStringReconnects(t *testing.T) {

	factory := factoryCore{}

	require.NoError(t, factory.refreshCommonDatabase(lifecycleConnection("59999", "lifecycle-a")))
	oldClient := factory.commonDatabase.Client()

	require.NoError(t, factory.refreshCommonDatabase(lifecycleConnection("59998", "lifecycle-a")))

	require.NotSame(t, oldClient, factory.commonDatabase.Client(), "changed connect string must build a new client")
	requireDisconnected(t, oldClient)
	requireConnected(t, factory.commonDatabase.Client())
}

// TestRefreshCommonDatabase_RequiresSettings pins the validations: missing settings error out and
// must not touch the (nil) connection.
func TestRefreshCommonDatabase_RequiresSettings(t *testing.T) {

	table := []struct {
		name       string
		connection mapof.String
	}{
		{"missing connectString", mapof.String{"database": "lifecycle-a"}},
		{"missing database", mapof.String{"connectString": "mongodb://127.0.0.1:59999/"}},
		{"empty", mapof.String{}},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {
			factory := factoryCore{}
			require.Error(t, factory.refreshCommonDatabase(testCase.connection))
			require.Nil(t, factory.commonDatabase)
		})
	}
}

/******************************************
 * refreshQueue
 ******************************************/

// newTestFactoryCore returns a factoryCore carrying the same inert placeholder queue that init()
// installs, so refreshQueue's first run behaves exactly as it does at boot.
func newTestFactoryCore() *factoryCore {
	return &factoryCore{
		queue: queue.New(),
	}
}

// stopQueueOnCleanup stops the factory's CURRENT queue when the test ends.  Queues that
// refreshQueue already replaced were stopped by refreshQueue itself and must not be stopped
// again (turbine's Stop panics on a second call).
func stopQueueOnCleanup(t *testing.T, factory *factoryCore) {
	t.Helper()
	t.Cleanup(func() {
		factory.queue.Stop()
	})
}

// TestRefreshQueue_UnchangedInputsKeepQueue pins the reload guard: reloading with the same
// storage mode (and no database swap) must KEEP the running queue.  Rebuilding would stop the
// old queue, and every captured pointer would then feed a stopped queue that drops tasks.
func TestRefreshQueue_UnchangedInputsKeepQueue(t *testing.T) {

	factory := newTestFactoryCore()
	stopQueueOnCleanup(t, factory)
	placeholder := factory.queue

	// First refresh always rebuilds: the placeholder has no consumers
	factory.refreshQueue(false)
	first := factory.queue
	require.NotSame(t, placeholder, first, "first refresh must replace init()'s inert placeholder")

	// Reload: same inputs, same queue
	factory.refreshQueue(false)
	require.Same(t, first, factory.queue, "unchanged inputs must keep the same queue")
}

// TestRefreshQueue_InMemoryIgnoresDatabaseSwap pins the !withStorage leg of the guard: an
// in-memory queue (setup mode) does not touch the common database, so swapping the database must
// not rebuild it.
func TestRefreshQueue_InMemoryIgnoresDatabaseSwap(t *testing.T) {

	factory := newTestFactoryCore()
	stopQueueOnCleanup(t, factory)

	factory.refreshQueue(false)
	first := factory.queue

	factory.commonDatabase = lazyDatabase(t, "lifecycle-swap")
	factory.refreshQueue(false)

	require.Same(t, first, factory.queue, "an in-memory queue must survive a database swap")
}

// TestRefreshQueue_RebuildsWhenStorageModeChanges pins the withStorage leg of the guard.
func TestRefreshQueue_RebuildsWhenStorageModeChanges(t *testing.T) {

	factory := newTestFactoryCore()
	stopQueueOnCleanup(t, factory)
	factory.commonDatabase = lazyDatabase(t, "lifecycle-mode")

	factory.refreshQueue(false)
	first := factory.queue

	factory.refreshQueue(true)
	require.NotSame(t, first, factory.queue, "a storage-mode change must rebuild the queue")
}

// TestRefreshQueue_RebuildsWhenCommonDatabaseSwaps pins the storage-bearing leg: when
// refreshCommonDatabase swaps the connection, the queue's mongo storage wraps a dead client, so
// the queue MUST be rebuilt -- and until the swap happens, it must NOT be.
func TestRefreshQueue_RebuildsWhenCommonDatabaseSwaps(t *testing.T) {

	factory := newTestFactoryCore()
	stopQueueOnCleanup(t, factory)
	factory.commonDatabase = lazyDatabase(t, "lifecycle-storage-a")

	factory.refreshQueue(true)
	first := factory.queue

	// Reload without a swap: keep the queue
	factory.refreshQueue(true)
	require.Same(t, first, factory.queue, "same database handle must keep the same queue")

	// Swap the database (what refreshCommonDatabase does on a real settings change), then reload
	factory.commonDatabase = lazyDatabase(t, "lifecycle-storage-b")
	factory.refreshQueue(true)
	require.NotSame(t, first, factory.queue, "a database swap must rebuild the queue")
}

/******************************************
 * The composed reload scenario
 ******************************************/

// TestConfigReload_KeepsHandlesAlive replays the original incident at the factoryCore level,
// in readConfig's call order: boot (database, then queue), then a config reload that does not
// change either.  Both handles must survive, and the accessors that domain factories read
// through -- CommonDatabase() and Queue() -- must return the same live values.
func TestConfigReload_KeepsHandlesAlive(t *testing.T) {

	factory := newTestFactoryCore()
	stopQueueOnCleanup(t, factory)

	// Boot
	require.NoError(t, factory.refreshCommonDatabase(lifecycleConnection("59999", "lifecycle-reload")))
	factory.refreshQueue(true)

	bootDatabase := factory.commonDatabase
	bootQueue := factory.queue

	// Config reload with unchanged database settings (fresh, equal-valued map)
	require.NoError(t, factory.refreshCommonDatabase(lifecycleConnection("59999", "lifecycle-reload")))
	factory.refreshQueue(true)

	// Both handles survive, and the getters domain factories read through agree
	require.Same(t, bootDatabase, factory.commonDatabase)
	require.Same(t, bootQueue, factory.queue)
	require.Same(t, bootDatabase, factory.CommonDatabase())
	require.Same(t, bootQueue, factory.Queue())

	// The boot client was never disconnected
	requireConnected(t, bootDatabase.Client())
}
