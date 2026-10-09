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

// SetupFactory manages the server-level services used by the setup console, which
// can run without a common (ActivityPub Cache) database.
type SetupFactory struct {
	factoryCore // No state of its own; the modes differ only in which lifecycle methods run
}

// NewSetupFactory returns a SetupFactory built from the first configuration.
// A missing or unreachable common database is reported, never fatal.
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
func (factory *SetupFactory) start(subscription <-chan config.Config) {

	// RULE: Always drain this single-slot channel, or the storage watcher wedges
	// and the console's next save overwrites other nodes' changes.
	for config := range subscription {
		log.Info().Msg("Setup: configuration file (updated)")
		factory.configure(config)
	}
}

// configure applies a server configuration to the SetupFactory, skipping everything
// that requires the common database or touches production state.
func (factory *SetupFactory) configure(config config.Config) {

	const location = "server.SetupFactory.configure"

	// RULE: Serialize against every other reload, including a save posted from the console
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
	factory.templateService.Refresh(config.Templates)

	// RULE: The common database is best-effort here: connect (with a ping) if configured,
	// warn if not.  Domain management stays disabled until it connects.
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

// UpdateConfig saves the configuration, then connects the ActivityPub Cache database
// and rebuilds domain factories if its settings changed.
func (factory *SetupFactory) UpdateConfig(value config.Config) error {

	const location = "server.SetupFactory.UpdateConfig"

	// RULE: This is a reload in everything but name, so it takes reloadLock and
	// cannot interleave with one arriving from the storage subscription.
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()

	// Save the configuration.  It persists even if the connection attempt below fails.
	if err := factory.updateConfigLocked(value); err != nil {
		return derp.Wrap(err, location, "Writing configuration")
	}

	// RULE: Nothing to connect if no cache database is configured yet
	newCache := value.ActivityPubCache

	if newCache.IsEmpty() {
		return nil
	}

	// Connect and verify.  The settings are already saved, so a failure is a warning on the
	// console form, not a rejected save.  Unchanged settings keep the current connection.
	changed, err := factory.refreshCommonDatabase(newCache, true)

	if err != nil {
		return derp.Wrap(err, location, "Your settings were SAVED, but Emissary could not connect to the ActivityPub Cache database. Domains cannot be added until this is fixed.")
	}

	// Existing domain factories are still valid on an unchanged connection
	if !changed {
		return nil
	}

	// Rebuild every domain factory: existing ones are bound to the previous (now closed)
	// connection, so a simple Refresh is not enough.
	factory.removeAllDomains()
	factory.refreshDomains(value)

	// The cache is connected and domains are open for business. Huzzah!
	return nil
}
