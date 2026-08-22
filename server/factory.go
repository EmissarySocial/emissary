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

// NewFactory uses the provided configuration data to generate a new Factory.  If there are any
// errors connecting to a domain's datasource, NewFactory will derp.Report the error, but will
// continue loading without those domains.
//
// RULE: A FIRST configuration that cannot be applied is an error -- a server that never had a
// working configuration has nothing to keep serving, so main refuses to start.  Configurations
// that fail to apply LATER are handled by start(), which keeps the last-known-good instead.
func NewFactory(storage config.Storage, firstConfig config.Config, subscription <-chan config.Config, embeddedFiles embed.FS) (*Factory, error) {

	const location = "server.NewFactory"

	// Build the mode-independent core in place (see factoryCore.init for why in place matters)
	factory := Factory{}
	factory.init(storage, embeddedFiles)
	factory.rewire(func(value *wiring) {
		value.clientIPStrategy = realclientip.RemoteAddrStrategy{}
	})

	// Apply the first configuration read by main
	log.Info().Msg("Factory: reading configuration file (first time)")

	if err := factory.readConfig(firstConfig); err != nil {
		return nil, derp.Wrap(err, location, "Unable to apply the server configuration")
	}

	// Listen for configuration updates for the rest of the process lifetime
	go factory.start(subscription)

	// Done configuring the factory
	return &factory, nil
}

// start listens for configuration updates for the rest of the process lifetime.
//
// RULE: A configuration that fails to apply at runtime is reported, and the node KEEPS SERVING
// on its last-known-good configuration.  The alternative -- exiting, as boot does -- would let
// one bad save take down every node in a cluster at once, and then crash-loop them all against
// the same stored document.  A running node one moment before the reload was serving perfectly
// well; the bad NEW configuration changes nothing about that.  The next good save (delivered by
// the same subscription) recovers the node with no restart.
func (factory *Factory) start(subscription <-chan config.Config) {

	const location = "server.Factory.start"

	// Read configuration files from the channel
	for config := range subscription {

		log.Info().Msg("Factory: configuration file (updated)")

		if err := factory.readConfig(config); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to apply the updated configuration. KEEPING the last working configuration. Fix and re-save the server configuration."))
			log.Error().Msg("Configuration update REJECTED. This node is still running on its previous configuration.")
		}
	}
}

// readConfig applies a new configuration to this Factory and every service that depends on it,
// or returns an error having applied NOTHING.
//
// RULE: The whole reload runs under reloadLock, which serializes it against any other reload but
// is never taken by a reader.  Requests keep running throughout -- they read the CURRENT
// generation of wiring, and see the new one the moment each step publishes it.
//
// RULE: Everything that can FAIL runs before anything is published.  The only fallible step is
// the common database; it is validated first, so a rejected configuration leaves the previous
// one fully intact -- config, log level, filesystems, queue, everything.  A caller that gets an
// error back is guaranteed the factory still runs its last-known-good configuration.
func (factory *Factory) readConfig(config config.Config) error {

	const location = "server.Factory.readConfig"

	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()

	log.Info().Msg("Factory: received new configuration...")

	// RULE: MUST be able to connect to the common database, BEFORE anything else applies.
	// Unverified (no ping): the session check just below is this mode's verification.  On
	// failure the previous connection (and everything else) is untouched.
	changed, err := factory.refreshCommonDatabase(config.ActivityPubCache, false)

	if err != nil {
		return derp.Wrap(err, location, "The common database is not properly defined in the configuration")
	}

	// RULE: Synchronize shared indexes only when the connection actually changed (which
	// includes boot: the first refresh always changes nil -> connection).  Index definitions
	// are a function of the binary, not the configuration, so an unchanged connection has
	// nothing new to sync -- and re-syncing here on every reload ran on every live node for
	// every save anywhere in the cluster.
	if changed {
		factory.syncCommonDatabaseIndexes()
	}

	server := mongodb.NewServer(factory.currentWiring().commonDatabase)
	session, err := server.Session(context.Background())

	if err != nil {
		return derp.Wrap(err, location, "Unable to connect to the common database")
	}

	// The configuration is applicable.  Everything from here down is infallible-by-design, so
	// the reload can no longer end half-applied.

	// Set logging level from the configuration file
	setLogLevel(config)

	// Update the configuration with the latest values.
	factory.setConfigLocked(config)

	// Refresh these global services with values we'll always need.
	factory.emailService.Refresh()
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

	// RULE: If we're running the setup console, then
	// do not run the remaining updates
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

	// Derive the strategy for calculating the client's real ip address
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
