package server

import (
	"context"
	"embed"

	"github.com/EmissarySocial/emissary/config"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/realclientip/realclientip-go"
	"github.com/rs/zerolog/log"
)

// Factory manages all server-level services, and generates individual
// domain factories for each domain
type Factory struct {
	factoryCore

	setup bool // If TRUE, then the factory is in setup mode. This value cannot be changed
}

// NewFactory returns a Factory built from the first configuration, or an error if that configuration
// cannot be applied. Domains that fail to connect are reported and skipped.
func NewFactory(storage config.Storage, firstConfig config.Config, subscription <-chan config.Config, embeddedFiles embed.FS) (*Factory, error) {

	const location = "server.NewFactory"

	// Build the mode-independent core in place (see factoryCore.init for why in place matters)
	factory := Factory{}
	factory.init(storage, embeddedFiles)
	factory.rewire(func(value *wiring) {
		value.clientIPStrategy = realclientip.RemoteAddrStrategy{}
	})

	// RULE: The first configuration must apply; a server that never had a working
	// configuration has nothing to keep serving, so main refuses to start.
	log.Info().Msg("Factory: reading configuration file (first time)")

	if err := factory.readConfig(firstConfig); err != nil {
		return nil, derp.Wrap(err, location, "Unable to apply the server configuration")
	}

	// Listen for configuration updates for the rest of the process lifetime
	go factory.start(subscription)

	// Done configuring the factory
	return &factory, nil
}

// start applies every configuration update published by the storage service,
// for the rest of the process lifetime.
func (factory *Factory) start(subscription <-chan config.Config) {

	const location = "server.Factory.start"

	// Read configuration files from the channel
	for config := range subscription {

		log.Info().Msg("Factory: configuration file (updated)")

		// RULE: A rejected update is reported, and the node keeps serving its last-known-good
		// configuration.  Exiting would crash-loop every node in the cluster on one bad save.
		if err := factory.readConfig(config); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to apply the updated configuration. KEEPING the last working configuration. Fix and re-save the server configuration."))
			log.Error().Msg("Configuration update REJECTED. This node is still running on its previous configuration.")
		}
	}
}

// readConfig applies a new configuration to this Factory and every service that depends on it,
// or returns an error having applied nothing.
func (factory *Factory) readConfig(config config.Config) error {

	const location = "server.Factory.readConfig"

	// RULE: Serialize against every other reload.  Readers never take this lock, so
	// requests keep running on the current wiring throughout.
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()

	log.Info().Msg("Factory: received new configuration...")

	// RULE: The common database is the only fallible step, so it runs before anything is
	// published.  No ping here: the session check below is live mode's verification.
	changed, err := factory.refreshCommonDatabase(config.ActivityPubCache, false)

	if err != nil {
		return derp.Wrap(err, location, "The common database is not properly defined in the configuration")
	}

	// RULE: Synchronize shared indexes only when the connection changed (including boot).
	// Index definitions come from the binary, so an unchanged connection has nothing to sync.
	if changed {
		factory.syncCommonDatabaseIndexes()
	}

	server := mongodb.NewServer(factory.currentWiring().commonDatabase)
	session, err := server.Session(context.Background())

	if err != nil {
		return derp.Wrap(err, location, "Unable to connect to the common database")
	}

	// Everything below cannot fail, so the reload can no longer end half-applied.

	// Set logging level from the configuration file
	setLogLevel(config)

	// Update the configuration with the latest values.
	factory.setConfigLocked(config)

	// Refresh these global services with values we'll always need.
	factory.templateService.Refresh(config.Templates)

	// Set timeout threshold for slow queries
	mongodb.SetLogTimeout(config.LogSlowQueries)

	// Mount the attachment and export directories
	factory.refreshFilesystems(config)

	// Use new Queue configuration
	log.Trace().Str("loc", location).Msg("Setting up queue...")
	factory.refreshQueue(factory.IsLiveMode())

	// Derp configuration
	factory.refreshDerpPlugins(config)

	// Insert/Update/Delete Domains in the domain list
	factory.refreshDomains(config)

	// RULE: The setup console skips the remaining updates
	if factory.IsSetupMode() {
		log.Trace().Msg("Factory.readConfig: In setup mode, so skipping domain updates")
		return nil
	}

	// JWT Service configuration
	factory.jwtService.Refresh(server)

	// Digital Dome configuration
	factory.digitalDome.With(dome.LogDatabase(session.Collection("DigitalDome")))

	// Bootstrap the "Scheduler" task.  Duplicates will be dropped.
	// This task will be used to schedule all other daily/hourly tasks
	log.Trace().Str("loc", location).Msg("Starting Task Scheduler")
	if err := factory.currentWiring().queue.Publish(queue.NewTask("Scheduler", mapof.NewAny())); err != nil {
		derp.Report(derp.Wrap(err, location, "Starting scheduler"))
	}

	// Publish the strategy for calculating the client's real IP address
	clientIPStrategy := factory.calcClientIPStrategy(config)

	factory.rewireLocked(func(value *wiring) {
		value.clientIPStrategy = clientIPStrategy
	})

	// This configuration is fully applied. Engage.
	return nil
}

// IsLiveMode returns TRUE if the server is serving real websites, and not the setup mode.
func (factory *Factory) IsLiveMode() bool {
	return !factory.setup
}

// IsSetupMode returns TRUE if the server is in setup mode, and is not serving real websites.
func (factory *Factory) IsSetupMode() bool {
	return factory.setup
}
