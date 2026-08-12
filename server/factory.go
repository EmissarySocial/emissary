package server

import (
	"context"
	"embed"
	"os"

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

// NewFactory uses the provided configuration data to generate a new Factory
// if there are any errors connecting to a domain's datasource, NewFactory will derp.Report
// the error, but will continue loading without those domains.
func NewFactory(storage config.Storage, firstConfig config.Config, subscription <-chan config.Config, embeddedFiles embed.FS) *Factory {

	// Build the mode-independent core in place (see factoryCore.init for why in place matters)
	factory := Factory{}
	factory.init(storage, embeddedFiles)
	factory.clientIPStrategy = realclientip.RemoteAddrStrategy{}

	// Apply the first configuration read by main
	log.Info().Msg("Factory: reading configuration file (first time)")
	factory.readConfig(firstConfig)

	// Listen for configuration updates for the rest of the process lifetime
	go factory.start(subscription)

	// Done configuring the factory
	return &factory
}

func (factory *Factory) start(subscription <-chan config.Config) {

	// Read configuration files from the channel
	for config := range subscription {
		log.Info().Msg("Factory: configuration file (updated)")
		factory.readConfig(config)
	}
}

func (factory *Factory) readConfig(config config.Config) {

	const location = "server.Factory.readConfig"

	// Set logging level from the configuration file
	setLogLevel(config)

	log.Info().Msg("Factory: received new configuration...")

	// Update the configuration with the latest values.
	factory.config = config

	// Refresh these global services with values we'll always need.
	factory.emailService.Refresh()
	factory.templateService.Refresh(config.Templates)

	// RULE: MUST be able to connect to the common database
	if err := factory.refreshCommonDatabase(config.ActivityPubCache); err != nil {
		message := "Halting. Common database not properly defined in configuration file."
		derp.Report(derp.Internal(location, message))
		os.Exit(1)
	}

	// Synchronize shared indexes on the common database
	factory.syncCommonDatabaseIndexes(config.ActivityPubCache)

	server := mongodb.NewServer(factory.commonDatabase)
	session, err := server.Session(context.Background())

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Connecting to common database."))
		os.Exit(1)
	}

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
		return
	}

	// JWT Service configuration
	factory.jwtService.Refresh(server)

	// Digital Dome configuration
	factory.digitalDome.With(dome.LogDatabase(session.Collection("DigitalDome")))

	// Bootstrap the "Scheduler" task.  Duplicates will be dropped.
	// This task will be used to schedule all other daily/hourly tasks
	log.Trace().Str("loc", location).Msg("Starting Task Scheduler")
	if err := factory.queue.Publish(queue.NewTask("Scheduler", mapof.NewAny())); err != nil {
		derp.Report(derp.Wrap(err, location, "Starting scheduler"))
	}

	// Derive the strategry for calculating the client's real ip address
	factory.clientIPStrategy = factory.calcClientIPStrategy(config)
}

// IsLiveMode returns TRUE if the server is serving real websites, and not the setup mode.
func (factory *Factory) IsLiveMode() bool {
	return !factory.setup
}

// IsSetupMode returns TRUE if the server is in setup mode, and is not serving real websites.
func (factory *Factory) IsSetupMode() bool {
	return factory.setup
}
