package server

import (
	"context"
	"embed"
	"time"

	"github.com/EmissarySocial/emissary/config"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/realclientip/realclientip-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// SetupFactory manages the server-level services used by the setup console.
// Unlike the live Factory, it runs without a common (ActivityPub Cache) database:
// the connection is attempted best-effort, and domain management stays disabled
// (with a clear error, not a crash) until one is configured. (FACTORY-MODES D1/D7)
type SetupFactory struct {
	factoryCore

	// Snapshot of the settings behind the current VERIFIED common-database connection.
	// Stored as plain strings (not the config's mapof.String) because the setup handlers
	// mutate the config's maps in place, so map comparisons cannot detect changes.
	connectedConnectString string
	connectedDatabase      string
}

// NewSetupFactory uses the provided configuration data to generate a factory
// for the setup console.  It never exits on a missing or unreachable common
// database, because fixing the configuration is the whole point of setup.
func NewSetupFactory(storage config.Storage, firstConfig config.Config, embeddedFiles embed.FS) *SetupFactory {

	// Build the mode-independent core in place (see factoryCore.init for why in place matters)
	factory := SetupFactory{}
	factory.init(storage, embeddedFiles)

	// The setup console only ever serves localhost, so the direct RemoteAddr is always correct
	factory.clientIPStrategy = realclientip.RemoteAddrStrategy{}

	// Apply the first configuration read by main
	factory.configure(firstConfig)

	// Enjoy your new SetupFactory.
	return &factory
}

// configure applies a server configuration to the SetupFactory.  It mirrors the
// live Factory's readConfig, minus everything that requires a working common
// database or would touch production state (queue storage, scheduler, JWT).
func (factory *SetupFactory) configure(config config.Config) {

	const location = "server.SetupFactory.configure"

	// Set logging level from the configuration file
	setLogLevel(config)

	// RULE: The setup console never fully silences logging.  Warnings must stay
	// visible so misconfiguration (like a missing ActivityPub Cache) is not swallowed
	// by the default config's "None" debug level.
	if zerolog.GlobalLevel() > zerolog.WarnLevel {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	// Update the configuration with the latest values.
	factory.config = config

	// Refresh these global services with values we'll always need.
	factory.emailService.Refresh()
	factory.templateService.Refresh(config.Templates)

	// RULE: The common database is best-effort in setup mode (FACTORY-MODES D1): connect if
	// configured, warn if not.  Domain management stays disabled until it connects.
	if config.ActivityPubCache.IsEmpty() {
		log.Warn().Msg("Setup: No ActivityPub Cache database configured yet. Domains cannot be added until it is.")
	} else if err := factory.connectCommonDatabase(config.ActivityPubCache); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to connect to the ActivityPub Cache database"))
		log.Warn().Msg("Setup: Could not connect to the ActivityPub Cache database. Check the connection settings.")
	}

	// Set timeout threshold for slow queries
	mongodb.SetLogTimeout(config.LogSlowQueries)

	// Mount the attachment and export directories
	factory.refreshFilesystems(config)

	// In-memory queue only: the setup console must never consume production tasks
	factory.refreshQueue(false)

	// Derp configuration (the mongo logger is skipped automatically while disconnected)
	factory.refreshDerpPlugins(config)

	// Domain factories require the common database; without one the domain list stays empty
	if factory.commonDatabase != nil {
		factory.refreshDomains(config)
	}
}

// UpdateConfig saves the configuration, then applies whatever the setup console can
// act on immediately: connecting the ActivityPub Cache database and (re)building
// domain factories.  It shadows the core's UpdateConfig for setup mode only.
func (factory *SetupFactory) UpdateConfig(value config.Config) error {

	const location = "server.SetupFactory.UpdateConfig"

	// Save the configuration.  This is the "accept" half of accept-but-warn (FACTORY-MODES
	// D1): the config persists even if the connection attempt below fails.
	if err := factory.factoryCore.UpdateConfig(value); err != nil {
		return derp.Wrap(err, location, "Writing configuration")
	}

	// RULE: Nothing to connect if no cache database is configured yet
	newCache := value.ActivityPubCache

	if newCache.IsEmpty() {
		return nil
	}

	// RULE: Skip the reconnect when the settings match the current verified connection.
	// Gratuitous reconnects would strand existing domain factories on a closed mongo client.
	if factory.commonDatabase != nil {
		if newCache.GetString("connectString") == factory.connectedConnectString {
			if newCache.GetString("database") == factory.connectedDatabase {
				return nil
			}
		}
	}

	// Connect + verify.  This is the "warn" half of accept-but-warn: the settings are already
	// saved, so a failure comes back as a warning on the console form, not a rejected save.
	if err := factory.connectCommonDatabase(newCache); err != nil {
		return derp.Wrap(err, location, "Your settings were SAVED, but Emissary could not connect to the ActivityPub Cache database. Domains cannot be added until this is fixed.")
	}

	// Rebuild every domain factory: existing ones are bound to the previous (now closed)
	// connection, so a simple Refresh is not enough.
	factory.domains.Clear()
	factory.refreshDomains(value)

	// The cache is connected and domains are open for business. Huzzah!
	return nil
}

// connectCommonDatabase connects to the ActivityPub Cache database and verifies that it
// is actually reachable.  On failure the factory rolls back to "not connected", so domain
// management stays gated (FACTORY-MODES D6) and a later attempt can retry cleanly.
func (factory *SetupFactory) connectCommonDatabase(connection mapof.String) error {

	const location = "server.SetupFactory.connectCommonDatabase"

	// Open the connection.  mongo.Connect is lazy, so this only fails on malformed settings.
	if err := factory.refreshCommonDatabase(connection); err != nil {
		return derp.Wrap(err, location, "Invalid database connection settings")
	}

	// Ping forces real server selection, bounded so an unreachable host fails in seconds
	// with a clear message NOW, instead of failing darkly when the first domain loads.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := factory.commonDatabase.Client().Ping(ctx, readpref.Primary()); err != nil {

		// Roll back to "not connected" so a later save retries, and so domain management
		// stays gated with a clear message instead of binding domains to a dead client.
		// The core's connection snapshots roll back too, so refreshCommonDatabase's
		// unchanged-guard cannot skip the reconnect when the same settings are retried.
		client := factory.commonDatabase.Client()
		factory.commonDatabase = nil
		factory.commonDatabaseURI = ""
		factory.commonDatabaseName = ""
		factory.connectedConnectString = ""
		factory.connectedDatabase = ""
		_ = client.Disconnect(context.Background())

		// Any existing domain factories are bound to the previous (now closed) connection.
		// Drop them so lookups fail cleanly instead of surfacing dark mongo errors.
		factory.domains.Clear()

		return derp.Wrap(err, location, `Unable to reach the database. Check the connect string — a single-member replica set needs "?directConnection=true".`)
	}

	// Record the settings behind this verified connection (see the field comment for why strings)
	factory.connectedConnectString = connection.GetString("connectString")
	factory.connectedDatabase = connection.GetString("database")

	// Synchronize shared indexes, now that the ping proved the server reachable
	factory.syncCommonDatabaseIndexes(connection)

	// Connected and verified. Splendid.
	return nil
}
