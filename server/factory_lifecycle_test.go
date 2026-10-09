package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/service"
	derpconsole "github.com/EmissarySocial/emissary/tools/derp-console"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/turbine/queue"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

// lifecycleConnection returns a new connection map on every call
func lifecycleConnection(port string, database string) mapof.String {
	// The unchanged-guard must compare values, never map identity, because config handlers rebuild
	// and mutate these maps in place (see the snapshot-scalars comment on factoryCore).
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

// requireDisconnected asserts that the client has been disconnected
func requireDisconnected(t *testing.T, client *mongo.Client) {
	t.Helper()

	// A disconnected client's Ping fails at once with mongo.ErrClientDisconnected.  A connected
	// one times out server selection (~200ms) with a different error, which fails this clearly.

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := client.Ping(ctx, readpref.Primary())
	require.ErrorIs(t, err, mongo.ErrClientDisconnected)
}

// requireConnected asserts that the client is still connected, by disconnecting it, so it can
// only be a test's final assertion on that client
func requireConnected(t *testing.T, client *mongo.Client) {
	t.Helper()

	// The first Disconnect of a live client returns nil, and a second returns ErrClientDisconnected
	require.NoError(t, client.Disconnect(context.Background()))
}

/******************************************
 * refreshCommonDatabase
 ******************************************/

// TestRefreshCommonDatabase_UnchangedSettingsKeepClient requires a reload whose database settings
// are unchanged to keep the live client
func TestRefreshCommonDatabase_UnchangedSettingsKeepClient(t *testing.T) {

	// Reconnecting would disconnect the old client and strand every captured handle, which was the
	// original incident.
	factory := &factoryCore{}

	require.NoError(t, reloadCommonDatabase(factory, lifecycleConnection("59999", "lifecycle-a")))
	first := factory.currentWiring().commonDatabase
	require.NotNil(t, first)

	// Reload with a FRESH map carrying equal values (value comparison, not map identity)
	require.NoError(t, reloadCommonDatabase(factory, lifecycleConnection("59999", "lifecycle-a")))

	require.Same(t, first, factory.currentWiring().commonDatabase, "unchanged settings must keep the same database handle")
	requireConnected(t, first.Client())
}

// TestRefreshCommonDatabase_ChangedDatabaseNameReconnects pins the other half of the guard: when
// the settings DO change, the connection must be replaced and the old client disconnected.
func TestRefreshCommonDatabase_ChangedDatabaseNameReconnects(t *testing.T) {

	factory := &factoryCore{}

	require.NoError(t, reloadCommonDatabase(factory, lifecycleConnection("59999", "lifecycle-a")))
	oldClient := factory.currentWiring().commonDatabase.Client()

	require.NoError(t, reloadCommonDatabase(factory, lifecycleConnection("59999", "lifecycle-b")))

	require.Equal(t, "lifecycle-b", factory.currentWiring().commonDatabase.Name())
	require.NotSame(t, oldClient, factory.currentWiring().commonDatabase.Client(), "changed settings must build a new client")
	requireDisconnected(t, oldClient)
	requireConnected(t, factory.currentWiring().commonDatabase.Client())
}

// TestRefreshCommonDatabase_ChangedConnectStringReconnects covers the connectString leg of the
// comparison (same database name, different server address).
func TestRefreshCommonDatabase_ChangedConnectStringReconnects(t *testing.T) {

	factory := &factoryCore{}

	require.NoError(t, reloadCommonDatabase(factory, lifecycleConnection("59999", "lifecycle-a")))
	oldClient := factory.currentWiring().commonDatabase.Client()

	require.NoError(t, reloadCommonDatabase(factory, lifecycleConnection("59998", "lifecycle-a")))

	require.NotSame(t, oldClient, factory.currentWiring().commonDatabase.Client(), "changed connect string must build a new client")
	requireDisconnected(t, oldClient)
	requireConnected(t, factory.currentWiring().commonDatabase.Client())
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
			factory := &factoryCore{}
			require.Error(t, reloadCommonDatabase(factory, testCase.connection))
			require.Nil(t, factory.currentWiring().commonDatabase)
		})
	}
}

// TestRefreshCommonDatabase_VerifyRollsBackOnUnreachable requires a failed ping to roll the
// factory back to "not connected", and to clear the domain registry
func TestRefreshCommonDatabase_VerifyRollsBackOnUnreachable(t *testing.T) {

	// Publishing the un-pinged client would bind later lookups to a server that never answered, and
	// keeping the previous one would disagree with the settings the operator just saved.
	factory := &factoryCore{}
	factory.domains = xsync.NewMap[string, *service.Factory]()

	// Nothing listens on this port, so the ping fails after the URI's 200ms selection timeout
	changed, err := verifyCommonDatabase(factory, lifecycleConnection("59999", "lifecycle-verify"))

	require.Error(t, err)
	require.True(t, changed, "a rollback IS a change to the live connection")

	result := factory.currentWiring()
	require.Nil(t, result.commonDatabase)
	require.False(t, result.commonDatabaseVerified)
	require.Empty(t, result.commonDatabaseURI)
}

// TestRefreshCommonDatabase_UnverifiedIsNotKeptWhenVerifying requires that an unverified client
// with the same settings does not satisfy a caller that requires verification
func TestRefreshCommonDatabase_UnverifiedIsNotKeptWhenVerifying(t *testing.T) {

	// Otherwise the setup console could report "connected" for a client that never answered a Ping
	factory := &factoryCore{}
	factory.domains = xsync.NewMap[string, *service.Factory]()

	connection := lifecycleConnection("59999", "lifecycle-verify")

	// An unverified connection to these settings exists (the live-mode path)
	require.NoError(t, reloadCommonDatabase(factory, connection))
	require.NotNil(t, factory.currentWiring().commonDatabase)

	// Verification must attempt the ping (and fail against the closed port), not skip
	changed, err := verifyCommonDatabase(factory, connection)

	require.Error(t, err, "an unverified connection must not satisfy the verified guard")
	require.True(t, changed)
	require.Nil(t, factory.currentWiring().commonDatabase)
}

// TestRefreshCommonDatabase_MalformedURIKeepsConnection verifies that settings which cannot even
// produce a client are rejected before anything is published.
func TestRefreshCommonDatabase_MalformedURIKeepsConnection(t *testing.T) {

	factory := &factoryCore{}

	require.NoError(t, reloadCommonDatabase(factory, lifecycleConnection("59999", "lifecycle-malformed")))
	before := factory.CommonDatabase()

	err := reloadCommonDatabase(factory, mapof.String{"connectString": "not-a-mongodb-uri", "database": "lifecycle-malformed"})

	require.Error(t, err)
	require.Same(t, before, factory.CommonDatabase())
	requireConnected(t, before.Client())
}

// TestRefreshCommonDatabase_VerifySucceeds connects and verifies a live common database, then
// keeps that verified connection when the same settings arrive again.
func TestRefreshCommonDatabase_VerifySucceeds(t *testing.T) {

	requireLiveMongo(t)

	factory := &factoryCore{}
	factory.domains = xsync.NewMap[string, *service.Factory]()

	connection := mapof.String{
		"connectString": liveTestConnection,
		"database":      "emissary_servertest_verify_" + primitive.NewObjectID().Hex(),
	}

	changed, err := verifyCommonDatabase(factory, connection)
	require.NoError(t, err)
	require.True(t, changed)

	live := factory.CommonDatabase()
	require.NotNil(t, live)
	require.True(t, factory.currentWiring().commonDatabaseVerified)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = live.Drop(ctx)
		_ = live.Client().Disconnect(ctx)
	})

	// The same settings, verified again, keep the connection
	changed, err = verifyCommonDatabase(factory, connection)
	require.NoError(t, err)
	require.False(t, changed)
	require.Same(t, live, factory.CommonDatabase())
}

// TestRefreshCommonDatabase_VerifyFailureStopsDomainWatchers pins that the rollback's registry
// clear also stops each dropped domain's watchers, which otherwise reopen until canceled.
func TestRefreshCommonDatabase_VerifyFailureStopsDomainWatchers(t *testing.T) {

	factory, domainConfig := newLiveDomainCore(t)
	baseline := countDomainWatchers()

	startLiveDomain(t, factory, domainConfig)
	require.Greater(t, countDomainWatchers(), baseline)

	// The operator saves settings for a server that never answers
	changed, err := verifyCommonDatabase(factory, lifecycleConnection("59999", "lifecycle-verify-watchers"))

	require.Error(t, err)
	require.True(t, changed)
	require.Zero(t, factory.domains.Size())
	requireWatchersStopped(t, baseline, "domains dropped by the rollback must stop their watchers")
}

/******************************************
 * openCommonDatabase and disconnectCommonDatabase
 ******************************************/

// TestOpenCommonDatabase covers the validations, a malformed URI, and a lazily opened client
func TestOpenCommonDatabase(t *testing.T) {

	t.Run("RequiresSettings", func(t *testing.T) {
		for _, connection := range []mapof.String{
			{"database": "lifecycle-open"},
			{"connectString": "mongodb://127.0.0.1:59999/"},
			{},
			nil,
		} {
			database, err := openCommonDatabase(connection)
			require.Error(t, err)
			require.Nil(t, database)
		}
	})

	t.Run("MalformedURI", func(t *testing.T) {
		database, err := openCommonDatabase(mapof.String{"connectString": "not-a-mongodb-uri", "database": "lifecycle-open"})
		require.Error(t, err)
		require.Nil(t, database)
	})

	t.Run("OmitsCredentials", func(t *testing.T) {
		database, err := openCommonDatabase(mapof.String{
			"connectString": "mongodb://user:db-password-secret@127.0.0.1:notaport/",
			"database":      "lifecycle-open",
		})
		require.Error(t, err)
		require.Nil(t, database)
		secretcheck.RequireAbsent(t, err, "db-password-secret")
	})

	t.Run("OpensLazily", func(t *testing.T) {
		database, err := openCommonDatabase(lifecycleConnection("59999", "lifecycle-open"))
		require.NoError(t, err)
		require.NotNil(t, database)
		require.Equal(t, "lifecycle-open", database.Name())
		requireConnected(t, database.Client())
	})
}

// TestDisconnectCommonDatabase covers a nil database, a live client, and a client already closed
func TestDisconnectCommonDatabase(t *testing.T) {

	recorder := recordReports(t)

	// Nothing to disconnect
	require.NotPanics(t, func() { disconnectCommonDatabase(nil) })

	// A live client is disconnected
	database := lazyDatabase(t, "lifecycle-disconnect")
	disconnectCommonDatabase(database)
	requireDisconnected(t, database.Client())
	require.Empty(t, recorder.reported())

	// A second disconnect is reported, not fatal
	disconnectCommonDatabase(database)
	require.Len(t, recorder.reported(), 1)
}

/******************************************
 * syncCommonDatabaseIndexes
 ******************************************/

// TestSyncCommonDatabaseIndexes verifies that the shared indexes land in the common database,
// and that there is nothing to do without one.
func TestSyncCommonDatabaseIndexes(t *testing.T) {

	t.Run("NoDatabase", func(t *testing.T) {
		factory := &factoryCore{}
		require.NotPanics(t, factory.syncCommonDatabaseIndexes)
	})

	t.Run("Live", func(t *testing.T) {

		factory, _ := newLiveDomainCore(t)
		factory.reloadLock.Lock()
		factory.syncCommonDatabaseIndexes()
		factory.reloadLock.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cursor, err := factory.CommonDatabase().Collection("ErrorLog").Indexes().List(ctx)
		require.NoError(t, err)

		var indexes []map[string]any
		require.NoError(t, cursor.All(ctx, &indexes))
		require.Greater(t, len(indexes), 1, "more than the default _id index")
	})
}

/******************************************
 * refreshFilesystems
 ******************************************/

// reloadFilesystems runs refreshFilesystems under reloadLock, as a configuration reload does
func reloadFilesystems(factory *factoryCore, value config.Config) {
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()
	factory.refreshFilesystems(value)
}

// TestRefreshFilesystems_MountsEachDirectory verifies that each filesystem is mounted from its
// own setting
func TestRefreshFilesystems_MountsEachDirectory(t *testing.T) {

	root := t.TempDir()

	for _, directory := range []string{"originals", "cache", "exports"} {
		require.NoError(t, os.Mkdir(filepath.Join(root, directory), 0o750))
	}

	value := config.DefaultConfig()
	value.AttachmentOriginals = mapof.String{"adapter": config.FolderAdapterFile, "location": filepath.Join(root, "originals")}
	value.AttachmentCache = mapof.String{"adapter": config.FolderAdapterFile, "location": filepath.Join(root, "cache")}
	value.ExportCache = mapof.String{"adapter": config.FolderAdapterFile, "location": filepath.Join(root, "exports")}

	factory := &factoryCore{}
	reloadFilesystems(factory, value)

	current := factory.currentWiring()

	// writeProbe writes a file through one filesystem, and returns where it landed on disk
	writeProbe := func(filesystem afero.Fs) string {
		require.NoError(t, afero.WriteFile(filesystem, "probe.txt", []byte("probe"), 0o600))
		matches, err := filepath.Glob(filepath.Join(root, "*", "probe.txt"))
		require.NoError(t, err)
		require.Len(t, matches, 1)
		require.NoError(t, os.Remove(matches[0]))
		return filepath.Base(filepath.Dir(matches[0]))
	}

	require.Equal(t, "originals", writeProbe(current.attachmentOriginals))
	require.Equal(t, "cache", writeProbe(current.attachmentCache))
	require.Equal(t, "exports", writeProbe(current.exportCache))
}

// TestRefreshFilesystems_KeepsPreviousOnFailure verifies that a directory that cannot be mounted
// keeps the previous generation's filesystem, and is reported without the configuration's secrets.
func TestRefreshFilesystems_KeepsPreviousOnFailure(t *testing.T) {

	recorder := recordReports(t)

	factory := &factoryCore{}
	setTestFilesystems(factory)
	previous := factory.currentWiring()

	value := configWithDomains(config.Domain{DomainID: "1", Hostname: "one.example.com", MasterKey: testMasterKey})
	value.AttachmentOriginals = mapof.String{"adapter": "UNSUPPORTED"}
	value.AttachmentCache = mapof.String{"adapter": "UNSUPPORTED"}
	value.ExportCache = mapof.String{"adapter": "UNSUPPORTED"}

	reloadFilesystems(factory, value)

	current := factory.currentWiring()
	require.Same(t, previous.attachmentOriginals, current.attachmentOriginals)
	require.Same(t, previous.attachmentCache, current.attachmentCache)
	require.Same(t, previous.exportCache, current.exportCache)

	// Each failure is reported once, and none of them carries a domain's MasterKey
	reported := recorder.reported()
	require.Len(t, reported, 3)

	for _, err := range reported {
		secretcheck.RequireAbsent(t, err, testMasterKey)
	}
}

/******************************************
 * refreshQueue
 ******************************************/

// newTestFactoryCore returns a factoryCore carrying the same inert placeholder queue that init()
// installs, so refreshQueue's first run behaves exactly as it does at boot.
func newTestFactoryCore() *factoryCore {

	result := &factoryCore{}

	result.rewire(func(value *wiring) {
		value.queue = queue.New()
	})

	return result
}

// reloadCommonDatabase runs refreshCommonDatabase the way a live-mode configuration reload
// does: holding reloadLock, which every writer of the server wiring is required to hold.
func reloadCommonDatabase(factory *factoryCore, connection mapof.String) error {
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()

	_, err := factory.refreshCommonDatabase(connection, false)
	return err
}

// verifyCommonDatabase runs refreshCommonDatabase the way the setup console does: under
// reloadLock and with verification on, so the connection must answer a Ping to be published.
func verifyCommonDatabase(factory *factoryCore, connection mapof.String) (bool, error) {
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()

	return factory.refreshCommonDatabase(connection, true)
}

// reloadQueue runs refreshQueue under reloadLock, as a configuration reload does.
func reloadQueue(factory *factoryCore, withStorage bool) {
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()
	factory.refreshQueue(withStorage)
}

// setTestCommonDatabase swaps in a database handle without connecting to anything, so the
// pointer-identity half of refreshQueue's guard can be exercised directly.
func setTestCommonDatabase(factory *factoryCore, database *mongo.Database) {
	factory.rewire(func(value *wiring) {
		value.commonDatabase = database
	})
}

// stopQueueOnCleanup stops the factory's CURRENT queue when the test ends
func stopQueueOnCleanup(t *testing.T, factory *factoryCore) {
	t.Helper()

	// A queue that refreshQueue replaced was already stopped there, and turbine's Stop panics on a
	// second call.
	t.Cleanup(func() {
		factory.currentWiring().queue.Stop()
	})
}

// TestRefreshQueue_UnchangedInputsKeepQueue requires a reload with the same storage mode, and no
// database swap, to keep the running queue
func TestRefreshQueue_UnchangedInputsKeepQueue(t *testing.T) {

	// Rebuilding would stop the old queue, and every captured pointer would then feed a stopped
	// queue that drops tasks.
	factory := newTestFactoryCore()
	stopQueueOnCleanup(t, factory)
	placeholder := factory.currentWiring().queue

	// First refresh always rebuilds: the placeholder has no consumers
	reloadQueue(factory, false)
	first := factory.currentWiring().queue
	require.NotSame(t, placeholder, first, "first refresh must replace init()'s inert placeholder")

	// Reload: same inputs, same queue
	reloadQueue(factory, false)
	require.Same(t, first, factory.currentWiring().queue, "unchanged inputs must keep the same queue")
}

// TestRefreshQueue_InMemoryIgnoresDatabaseSwap requires an in-memory queue, which never uses the
// common database, to survive a swap of it
func TestRefreshQueue_InMemoryIgnoresDatabaseSwap(t *testing.T) {

	factory := newTestFactoryCore()
	stopQueueOnCleanup(t, factory)

	reloadQueue(factory, false)
	first := factory.currentWiring().queue

	setTestCommonDatabase(factory, lazyDatabase(t, "lifecycle-swap"))
	reloadQueue(factory, false)

	require.Same(t, first, factory.currentWiring().queue, "an in-memory queue must survive a database swap")
}

// TestRefreshQueue_RebuildsWhenStorageModeChanges pins the withStorage leg of the guard.
func TestRefreshQueue_RebuildsWhenStorageModeChanges(t *testing.T) {

	factory := newTestFactoryCore()
	stopQueueOnCleanup(t, factory)
	setTestCommonDatabase(factory, lazyDatabase(t, "lifecycle-mode"))

	reloadQueue(factory, false)
	first := factory.currentWiring().queue

	reloadQueue(factory, true)
	require.NotSame(t, first, factory.currentWiring().queue, "a storage-mode change must rebuild the queue")
}

// TestRefreshQueue_RebuildsWhenCommonDatabaseSwaps requires a storage-backed queue to be rebuilt
// when the common database swaps, and not before
func TestRefreshQueue_RebuildsWhenCommonDatabaseSwaps(t *testing.T) {

	// After a swap, the queue's mongo storage wraps a dead client
	factory := newTestFactoryCore()
	stopQueueOnCleanup(t, factory)
	setTestCommonDatabase(factory, lazyDatabase(t, "lifecycle-storage-a"))

	reloadQueue(factory, true)
	first := factory.currentWiring().queue

	// Reload without a swap: keep the queue
	reloadQueue(factory, true)
	require.Same(t, first, factory.currentWiring().queue, "same database handle must keep the same queue")

	// Swap the database (what refreshCommonDatabase does on a real settings change), then reload
	setTestCommonDatabase(factory, lazyDatabase(t, "lifecycle-storage-b"))
	reloadQueue(factory, true)
	require.NotSame(t, first, factory.currentWiring().queue, "a database swap must rebuild the queue")
}

/******************************************
 * The composed reload scenario
 ******************************************/

// TestConfigReload_KeepsHandlesAlive requires a reload that changes neither the database nor the
// queue to keep both handles, and the accessors to return them
func TestConfigReload_KeepsHandlesAlive(t *testing.T) {

	// This replays the original incident in readConfig's call order: boot (database, then queue),
	// then a reload.  Domain factories read through CommonDatabase() and Queue().
	factory := newTestFactoryCore()
	stopQueueOnCleanup(t, factory)

	// Boot
	require.NoError(t, reloadCommonDatabase(factory, lifecycleConnection("59999", "lifecycle-reload")))
	reloadQueue(factory, true)

	bootDatabase := factory.currentWiring().commonDatabase
	bootQueue := factory.currentWiring().queue

	// Config reload with unchanged database settings (fresh, equal-valued map)
	require.NoError(t, reloadCommonDatabase(factory, lifecycleConnection("59999", "lifecycle-reload")))
	reloadQueue(factory, true)

	// Both handles survive, and the getters domain factories read through agree
	require.Same(t, bootDatabase, factory.currentWiring().commonDatabase)
	require.Same(t, bootQueue, factory.currentWiring().queue)
	require.Same(t, bootDatabase, factory.CommonDatabase())
	require.Same(t, bootQueue, factory.Queue())

	// The boot client was never disconnected
	requireConnected(t, bootDatabase.Client())
}

/******************************************
 * refreshDerpPlugins
 ******************************************/

// TestRefreshDerpPlugins_NeverZero requires a configuration with no usable loggers to leave one
// reporter installed, with the swap as the only mutation
func TestRefreshDerpPlugins_NeverZero(t *testing.T) {

	// The registry is global, so an empty moment would swallow every error reported concurrently

	// Restore the global registry so other tests see what they expect
	t.Cleanup(func() { derp.SetPlugins(derpconsole.New()) })

	factory := &factoryCore{}

	// No loggers configured at all: the console fallback must land
	empty := config.DefaultConfig()
	empty.Loggers = nil

	factory.refreshDerpPlugins(empty)
	require.Equal(t, 1, derp.Plugins.Len(), "the fallback console reporter must be installed")

	// A mongo logger without a connected common database is skipped -- but never down to zero
	mongoOnly := config.DefaultConfig()
	mongoOnly.Loggers = sliceof.Object[mapof.Any]{{"type": "mongo"}}

	factory.refreshDerpPlugins(mongoOnly)
	require.Equal(t, 1, derp.Plugins.Len(), "an unusable logger list must still fall back to the console")
}

// TestRefreshDerpPlugins_BuildsEachLogger verifies that console and mongo loggers are installed,
// and that an unknown type is skipped.
func TestRefreshDerpPlugins_BuildsEachLogger(t *testing.T) {

	t.Cleanup(func() { derp.SetPlugins(derpconsole.New()) })

	factory := &factoryCore{}
	setTestCommonDatabase(factory, lazyDatabase(t, "lifecycle-loggers"))

	value := config.DefaultConfig()
	value.Loggers = sliceof.Object[mapof.Any]{
		{"type": "console"},
		{"type": "mongo"},
		{"type": "carrier-pigeon"},
	}

	factory.refreshDerpPlugins(value)
	require.Equal(t, 2, derp.Plugins.Len())
}
