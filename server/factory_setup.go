package server

import (
	"embed"

	"github.com/EmissarySocial/emissary/config"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/realclientip/realclientip-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// SetupFactory manages the server-level services used by the setup console.
// Unlike the live Factory, it runs without a common (ActivityPub Cache) database:
// the connection is attempted best-effort, and domain management stays disabled
// (with a clear error, not a crash) until one is configured. (FACTORY-MODES D1/D7)
//
// It adds no state of its own -- even the "is the database verified?" flag lives in the shared
// wiring (commonDatabaseVerified) -- so the whole difference between the modes is which
// lifecycle methods run.
type SetupFactory struct {
	factoryCore
}

// NewSetupFactory uses the provided configuration data to generate a factory
// for the setup console.  It never exits on a missing or unreachable common
// database, because fixing the configuration is the whole point of setup.
func NewSetupFactory(storage config.Storage, firstConfig config.Config, subscription <-chan config.Config, embeddedFiles embed.FS) *SetupFactory {

	// Build the mode-independent core in place (see factoryCore.init for why in place matters)
	factory := SetupFactory{}
	factory.init(storage, embeddedFiles)

	// The setup console only ever serves localhost, so the direct RemoteAddr is always correct
	factory.rewire(func(value *wiring) {
		value.clientIPStrategy = realclientip.RemoteAddrStrategy{}
	})

	// Apply the first configuration read by main
	factory.configure(firstConfig)

	// Listen for configuration updates for the rest of the process lifetime
	go factory.start(subscription)

	// Enjoy your new SetupFactory.
	return &factory
}

// start applies every configuration update published by the storage service.
//
// RULE: The setup console MUST drain this channel, even though it is the process that usually
// writes the configuration rather than reading it.  The channel holds a single slot, so an
// un-drained subscription used to wedge the storage watcher permanently AND freeze the console's
// own view of the configuration at boot -- after which its next save would write that stale
// snapshot back over whatever another node had changed in the meantime.
func (factory *SetupFactory) start(subscription <-chan config.Config) {

	for config := range subscription {
		log.Info().Msg("Setup: configuration file (updated)")
		factory.configure(config)
	}
}

// configure applies a server configuration to the SetupFactory.  It mirrors the
// live Factory's readConfig, minus everything that requires a working common
// database or would touch production state (queue storage, scheduler, JWT).
//
// RULE: The whole reload runs under reloadLock, which serializes it against any other reload
// (including a save posted from the console at the same moment) but is never taken by a reader.
func (factory *SetupFactory) configure(config config.Config) {

	const location = "server.SetupFactory.configure"

	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()

	// Set logging level from the configuration file
	setLogLevel(config)

	// RULE: The setup console never fully silences logging.  Warnings must stay
	// visible so misconfiguration (like a missing ActivityPub Cache) is not swallowed
	// by the default config's "None" debug level.
	if zerolog.GlobalLevel() > zerolog.WarnLevel {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	// Update the configuration with the latest values.
	factory.setConfigLocked(config)

	// Refresh these global services with values we'll always need.
	factory.emailService.Refresh()
	factory.templateService.Refresh(config.Templates)

	// RULE: The common database is best-effort in setup mode (FACTORY-MODES D1): connect if
	// configured, warn if not.  Domain management stays disabled until it connects.
	//
	// Verification is on, so the connection is pinged before it is published.  The unchanged-
	// guard inside refreshCommonDatabase keeps this cheap: reloads are frequent now that the
	// console drains the subscription (its own saves echo back through the change stream), and
	// re-pinging plus re-synchronizing every shared index on each echo would be for nothing.
	if config.ActivityPubCache.IsEmpty() {
		log.Warn().Msg("Setup: No ActivityPub Cache database configured yet. Domains cannot be added until it is.")
	} else if _, err := factory.refreshCommonDatabase(config.ActivityPubCache, true); err != nil {
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
	if factory.currentWiring().commonDatabase != nil {
		factory.refreshDomains(config)
	}
}

// UpdateConfig saves the configuration, then applies whatever the setup console can
// act on immediately: connecting the ActivityPub Cache database and (re)building
// domain factories.  It shadows the core's UpdateConfig for setup mode only.
func (factory *SetupFactory) UpdateConfig(value config.Config) error {

	const location = "server.SetupFactory.UpdateConfig"

	// RULE: This is a reload in everything but name -- it connects a database and rebuilds
	// domain factories -- so it takes reloadLock, and cannot interleave with one arriving from
	// the storage subscription.
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()

	// Save the configuration.  This is the "accept" half of accept-but-warn (FACTORY-MODES
	// D1): the config persists even if the connection attempt below fails.
	if err := factory.updateConfigLocked(value); err != nil {
		return derp.Wrap(err, location, "Writing configuration")
	}

	// RULE: Nothing to connect if no cache database is configured yet
	newCache := value.ActivityPubCache

	if newCache.IsEmpty() {
		return nil
	}

	// Connect + verify.  This is the "warn" half of accept-but-warn: the settings are already
	// saved, so a failure comes back as a warning on the console form, not a rejected save.
	// The unchanged-guard inside skips all of it when the settings match the current verified
	// connection -- a gratuitous reconnect would strand existing domain factories on a closed
	// mongo client.
	changed, err := factory.refreshCommonDatabase(newCache, true)

	if err != nil {
		return derp.Wrap(err, location, "Your settings were SAVED, but Emissary could not connect to the ActivityPub Cache database. Domains cannot be added until this is fixed.")
	}

	if !changed {
		return nil
	}

	// Rebuild every domain factory: existing ones are bound to the previous (now closed)
	// connection, so a simple Refresh is not enough.
	factory.domains.Clear()
	factory.refreshDomains(value)

	// The cache is connected and domains are open for business. Huzzah!
	return nil
}
