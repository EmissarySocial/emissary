package server

import (
	"context"
	"embed"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/consumer"
	"github.com/EmissarySocial/emissary/service"
	derpconsole "github.com/EmissarySocial/emissary/tools/derp-console"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestServerFactory verifies that both run modes satisfy the factory interfaces their consumers depend on
func TestServerFactory(t *testing.T) {

	// Both run modes must satisfy the interfaces that domain factories
	// and queue consumers depend on.
	var factory service.ServerFactory = &Factory{}
	require.NotNil(t, factory)

	var setupFactory service.ServerFactory = &SetupFactory{}
	require.NotNil(t, setupFactory)

	var consumerFactory consumer.ServerFactory = &SetupFactory{}
	require.NotNil(t, consumerFactory)
}

// TestTestDatabaseConnection verifies the fail-fast behavior used by the setup console: an
// unreachable or malformed database connection must return an error QUICKLY, instead of
// hanging on the driver's default (~30s) server-selection timeout and leaving a broken
// domain persisted in the configuration file.
func TestTestDatabaseConnection(t *testing.T) {

	t.Run("Unreachable", func(t *testing.T) {
		// Port 1 has nothing listening; directConnection skips replica-set discovery so the
		// failure is bounded by our timeout rather than topology scanning.
		configuration := config.Domain{
			ConnectString: "mongodb://127.0.0.1:1/?directConnection=true",
			DatabaseName:  "emissary_test",
		}

		start := time.Now()
		err := testDatabaseConnection(configuration, 2*time.Second)
		elapsed := time.Since(start)

		require.Error(t, err)
		require.Less(t, elapsed, 10*time.Second, "connection check must fail fast, not hang")
	})

	t.Run("MalformedConnectString", func(t *testing.T) {
		configuration := config.Domain{
			ConnectString: "not-a-mongodb-uri",
			DatabaseName:  "emissary_test",
		}

		err := testDatabaseConnection(configuration, 2*time.Second)
		require.Error(t, err)
	})
}

/******************************************
 * Reload Fault Tolerance
 ******************************************/

// TestFactory_BadReloadKeepsLastKnownGood is the regression test for the cluster kill-switch.
// readConfig used to os.Exit(1) when a configuration's common database could not be opened --
// so one bad save, delivered reliably by the change stream, took down EVERY node at once and
// then crash-looped them all against the same stored document.  A rejected reload must now
// leave the node running its previous configuration, completely untouched.
func TestFactory_BadReloadKeepsLastKnownGood(t *testing.T) {

	factory := &Factory{}

	// Establish a known-good state the way a reload would
	good := config.DefaultConfig()
	good.AdminEmail = "known-good@example.com"
	setTestConfig(&factory.factoryCore, good)
	setTestCommonDatabase(&factory.factoryCore, lazyDatabase(t, "last-known-good"))

	before := factory.CommonDatabase()
	require.NotNil(t, before)

	// A configuration with NO common database cannot be applied
	bad := config.DefaultConfig()
	bad.AdminEmail = "bad@example.com"

	err := factory.readConfig(bad)
	require.Error(t, err)

	// RULE: NOTHING from the rejected configuration may be visible
	require.Equal(t, "known-good@example.com", factory.Config().AdminEmail)
	require.Same(t, before, factory.CommonDatabase(), "a rejected reload must not touch the database connection")
}

// TestFactory_StartSurvivesABadReload pins the supervision half: the subscription loop reports
// a failed reload and KEEPS DRAINING, so the next good configuration still recovers the node.
func TestFactory_StartSurvivesABadReload(t *testing.T) {

	factory := &Factory{}

	good := config.DefaultConfig()
	good.AdminEmail = "known-good@example.com"
	setTestConfig(&factory.factoryCore, good)
	setTestCommonDatabase(&factory.factoryCore, lazyDatabase(t, "start-survives"))

	// Two bad configurations arrive; the loop must consume both without dying
	subscription := make(chan config.Config, 2)
	subscription <- config.DefaultConfig()
	subscription <- config.DefaultConfig()
	close(subscription)

	finished := make(chan struct{})

	go func() {
		defer close(finished)
		factory.start(subscription)
	}()

	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("start() did not drain the subscription after failed reloads")
	}

	require.Equal(t, "known-good@example.com", factory.Config().AdminEmail)
}

/******************************************
 * StartQueue
 ******************************************/

// TestFactory_ReloadKeepsQueueRunning requires a live Factory whose common database changes to
// keep a running queue.  It needs the local replica set.
func TestFactory_ReloadKeepsQueueRunning(t *testing.T) {

	// BUG-313, through readConfig: one saved connection change stopped every node's queue
	client := requireLiveMongo(t)
	suffix := primitive.NewObjectID().Hex()
	databaseA := "emissary_servertest_queue_a_" + suffix
	databaseB := "emissary_servertest_queue_b_" + suffix
	logLevel := zerolog.GlobalLevel()
	before := countQueueWorkers()

	factory := &Factory{}
	factory.init(&stubStorage{}, embed.FS{})

	t.Cleanup(func() {
		factory.currentWiring().queue.Stop()
		_ = factory.currentWiring().commonDatabase.Client().Disconnect(context.Background())
		_ = client.Database(databaseA).Drop(context.Background())
		_ = client.Database(databaseB).Drop(context.Background())

		// readConfig replaces both of these process-wide settings
		derp.SetPlugins(derpconsole.New())
		zerolog.SetGlobalLevel(logLevel)
	})

	// Boot on database A, then start the queue as main does
	configuration := config.DefaultConfig()
	configuration.Templates = sliceof.Object[mapof.String]{{"adapter": config.FolderAdapterFile, "location": "../_embed/templates"}}
	configuration.ActivityPubCache = mapof.String{"connectString": liveTestConnection, "database": databaseA}
	require.NoError(t, factory.readConfig(configuration))

	factory.StartQueue()
	first := factory.Queue()
	requireQueueWorkers(t, before+16, "the boot queue must be running")

	// Reload onto database B, as a change-stream event does on every node
	configuration.ActivityPubCache = mapof.String{"connectString": liveTestConnection, "database": databaseB}
	require.NoError(t, factory.readConfig(configuration))

	require.NotSame(t, first, factory.Queue(), "a new common database must rebuild the queue")
	requireQueueWorkers(t, before+16, "the rebuilt queue must be running")

	// A second StartQueue, as a late boot call would be, adds nothing
	factory.StartQueue()
	requireQueueWorkers(t, before+16, "StartQueue must not start a running queue again")
}
