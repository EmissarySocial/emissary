package server

import (
	"context"
	"embed"
	"html/template"
	"iter"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/consumer"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/service"
	derpconsole "github.com/EmissarySocial/emissary/tools/derp-console"
	derpmongo "github.com/EmissarySocial/emissary/tools/derp-mongo"
	"github.com/EmissarySocial/emissary/tools/httpcache"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome"
	dt "github.com/benpate/domain"
	"github.com/benpate/icon"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/turbine/queue_mongo"
	"github.com/davidscottmills/goeditorjs"
	"github.com/labstack/echo/v4"
	"github.com/maypok86/otter"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Factory manages all server-level services, and generates individual
// domain factories for each domain
type Factory struct {
	storage config.Storage
	config  config.Config

	// Server-level services
	contentService      service.Content
	emailService        service.ServerEmail
	jwtService          service.JWT
	registrationService service.Registration
	themeService        service.Theme
	templateService     service.Template
	widgetService       service.Widget

	embeddedFiles       embed.FS
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs
	exportCache         afero.Fs
	commonDatabase      *mongo.Database
	workingDirectory    mediaserver.WorkingDirectory
	queue               *queue.Queue
	digitalDome         dome.Dome

	domains   *xsync.Map[string, *service.Factory]
	httpCache httpcache.HTTPCache
	setup     bool // If TRUE, then the factory is in setup mode. This value cannot be changed
}

// NewFactory uses the provided configuration data to generate a new Factory
// if there are any errors connecting to a domain's datasource, NewFactory will derp.Report
// the error, but will continue loading without those domains.
func NewFactory(commandLineArgs *config.CommandLineArgs, embeddedFiles embed.FS) *Factory {

	storage := config.Load(commandLineArgs)

	factory := Factory{
		storage:       storage,
		domains:       xsync.NewMap[string, *service.Factory](),
		embeddedFiles: embeddedFiles,
		jwtService:    service.NewJWT(),
		queue:         queue.New(),
	}

	// Build the in-memory cache
	otterCache, _ := otter.MustBuilder[string, string](1000).
		WithVariableTTL().
		Build()

	factory.httpCache = httpcache.NewOtterCache(otterCache, httpcache.WithTTL(1*time.Minute))

	// Global Registration Service
	factory.registrationService = service.NewRegistration(factory.FuncMap())

	// Global Theme service
	factory.themeService = service.NewTheme(
		factory.Template(),
		factory.Content(),
		factory.FuncMap(),
	)

	// Global Widget Service
	factory.widgetService = service.NewWidget(
		factory.FuncMap(),
	)

	// Global Template Service
	factory.templateService = *service.NewTemplate(
		factory.Filesystem(),
		factory.Registration(),
		factory.Email(),
		factory.Theme(),
		factory.Widget(),
		factory.FuncMap(),
		sliceof.NewObject[mapof.String](),
	)

	factory.contentService = service.NewContent(factory.EditorJS())

	factory.emailService = service.NewServerEmail(
		factory.Filesystem(),
		factory.FuncMap(),
		sliceof.NewObject[mapof.String](),
	)

	factory.digitalDome = dome.New(
		dome.LogStatusCodes(
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusInternalServerError,
		),
	)

	factory.jwtService = service.NewJWT()

	factory.queue = queue.New()

	factory.workingDirectory = mediaserver.NewWorkingDirectory(os.TempDir(), 4*time.Minute, 10000)
	factory.setup = commandLineArgs.Setup

	// Subscribe to configuration changes
	subscription := factory.storage.Subscribe()

	// Wait for the first "read" of the config file before we continue
	log.Info().Msg("Factory: reading configuration file (first time)")
	factory.readConfig(<-subscription)

	if factory.IsLiveMode() {

		// If the Factory is ready for domains, then start the configuration listener
		if factory.IsReadyForDomains() {
			go factory.start(subscription)

		} else {
			// Otherwise, force setup mode
			log.Warn().Msg("Factory: Server config is not complete. Switching to `setup` mode.")
			factory.setup = true
		}
	}

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
	switch config.DebugLevel {

	case "Trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "Debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "Info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "Error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "Fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "Panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	log.Info().Msg("Factory: received new configuration...")

	// Update the configuration with the latest values.
	factory.config = config

	// Refresh these global services with values we'll always need.
	factory.emailService.Refresh()
	factory.templateService.Refresh(config.Templates)

	// RULE: MUST be able to connect to the common database
	if err := factory.refreshCommonDatabase(config.ActivityPubCache); err != nil {
		message := "Halting. Common database not properly defined in configuration file."
		derp.Report(derp.InternalError(location, message))
		os.Exit(1)
	}

	server := mongodb.NewServer(factory.commonDatabase)
	session, err := server.Session(context.Background())

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to connect to common database."))
		os.Exit(1)
	}

	// Set timeout threshold for slow queries
	mongodb.SetLogTimeout(config.LogSlowQueries)

	filesystemService := factory.Filesystem()

	if attachmentOriginals, err := filesystemService.GetAfero(config.AttachmentOriginals); err == nil {
		factory.attachmentOriginals = attachmentOriginals
	} else {
		derp.Report(derp.Wrap(err, location, "Unable to get `attachment original` directory", config))
	}

	if attachmentCache, err := filesystemService.GetAfero(config.AttachmentCache); err == nil {
		factory.attachmentCache = attachmentCache
	} else {
		derp.Report(derp.Wrap(err, location, "Unable to get `attachment cache` directory", config))
	}

	if exportCache, err := filesystemService.GetAfero(config.ExportCache); err == nil {
		factory.exportCache = exportCache
	} else {
		derp.Report(derp.Wrap(err, location, "Unable to get `export cache` directory", config))
	}

	// Use new Queue configuration
	log.Trace().Str("loc", location).Msg("Setting up queue...")
	factory.refreshQueue()

	// Derp configuration
	derp.Plugins.Clear()
	for _, logger := range config.Loggers {

		switch logger.GetString("type") {

		case "console":
			log.Trace().Msg("Adding console logger to DERP...")
			derp.Plugins.Add(derpconsole.New())

		case "mongo":
			log.Trace().Msg("Adding mongo logger to DERP...")
			derp.Plugins.Add(derpmongo.New(
				factory.commonDatabase.Collection("ErrorLog"),
				logger))

		default:
			log.Error().Str("loc", location).Str("type", logger.GetString("type")).Msg("Unknown logging type")
		}
	}

	//
	// Insert/Update/Delete Domains
	// in the domain list

	// First, mark ALL for deletion
	factory.domains.Range(func(key string, domain *service.Factory) bool {
		domain.MarkForDeletion = true
		return true
	})

	// Insert/Update a factory for each domain in the configuration
	// removing MarkForDeletion on every domain we touch
	for _, domainConfig := range config.Domains {

		log.Trace().Str("loc", location).Str("domain", domainConfig.Hostname).Msg("Refreshing domain...")
		if err := factory.refreshDomain(domainConfig); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to refresh domain", domainConfig.ID))
			continue
		}
	}

	// Actually delete any domains that are still MarkForDeletion
	factory.domains.Range(func(key string, domain *service.Factory) bool {
		if domain.MarkForDeletion {
			factory.domains.Delete(key)
		}
		return true
	})

	// RULE: If we're running the setup console, then
	// do not run the remaining updates
	if factory.IsSetupMode() {
		log.Trace().Msg("Factory.readConfig: In setup mode, so skipping domain updates")
		return
	}

	// JWT Service configuration
	factory.jwtService.Refresh(server, config.MasterKey)

	// Digital Dome configuration
	factory.digitalDome.With(dome.LogDatabase(session.Collection("DigitalDome")))

	// Bootstrap the "Scheduler" task.  Duplicates will be dropped.
	// This task will be used to schedule all other daily/hourly tasks
	log.Trace().Str("loc", location).Msg("Starting Task Scheduler")
	if err := factory.queue.Publish(queue.NewTask("Scheduler", mapof.NewAny())); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to start scheduler"))
	}
}

// refreshDomain attempts to refresh an existing domain, or creates a new one if it doesn't exist
func (factory *Factory) refreshDomain(domainConfig config.Domain) error {

	const location = "server.factory.refreshDomain"

	// Try to find the domain
	if domain, exists := factory.domains.Load(domainConfig.Hostname); exists {

		// Even if there's an error "refreshing" the domain, we don't want to delete it
		domain.MarkForDeletion = false

		// Try to refresh the domain
		if err := domain.Refresh(domainConfig, factory.attachmentOriginals, factory.attachmentCache); err != nil {
			return derp.Wrap(err, location, "Unable to refresh domain", domainConfig.Hostname)
		}

		return nil
	}

	// Fall through means that the domain does not exist, so we need to create it
	newDomain, err := service.NewFactory(
		factory,
		mongodb.NewServer(factory.commonDatabase),
		domainConfig,
		factory.port(domainConfig),
		&factory.contentService,
		&factory.emailService,
		&factory.jwtService,
		factory.queue,
		&factory.registrationService,
		&factory.templateService,
		&factory.themeService,
		&factory.widgetService,
		factory.attachmentOriginals,
		factory.attachmentCache,
		factory.exportCache,
		&factory.httpCache,
		&factory.workingDirectory,
	)

	if err != nil {
		return derp.Wrap(err, location, "Unable to refresh configuration", domainConfig)
	}

	// If there are no errors, then add the domain to the list.
	factory.domains.Store(newDomain.Hostname(), newDomain)

	return nil
}

// refreshCommonDatabase updates the connection to the common database
func (factory *Factory) refreshCommonDatabase(connection mapof.String) error {

	const location = "server.factory.refreshCommonDatabase"

	// Collect arguments from the connection config
	uri := connection.GetString("connectString")
	database := connection.GetString("database")

	// RULE: Must have URI
	if uri == "" {
		return derp.InternalError(location, "Common database must have a URI")
	}

	// RULE: Must have a database name
	if database == "" {
		return derp.InternalError(location, "Common database must have a database name")
	}

	// Make a copy of the commonDatabase (pointer) so we can close it after we set up a new one
	commonDatabaseCopy := factory.commonDatabase // nolint:scopeguard

	// Try to connect to the cache database
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))

	if err != nil {
		return derp.Wrap(err, location, "Unable to connect to common database", uri)
	}

	log.Trace().Msg("Connected to common database")
	factory.commonDatabase = client.Database(database)

	// If there is already a cache connection in place, then close it before we open a new one
	if commonDatabaseCopy != nil {
		if err := commonDatabaseCopy.Client().Disconnect(context.Background()); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to disconnect from database"))
		}
	}

	// Update indexes asynchronously
	log.Trace().Str("database", factory.commonDatabase.Name()).Msg("Synchronizing common database indexes")
	go derp.Report(queries.SyncSharedIndexes(uri, database))

	return nil
}

// refreshQueue updates the connection to the task queue
func (factory *Factory) refreshQueue() {

	// If there is already a queue in place, then close it before we open a new one
	factory.queue.Stop()

	// Removing consumers because they're F@#$ing up outbound HTTP signatures
	consumer := consumer.New(factory)

	options := []queue.QueueOption{
		queue.WithConsumers(consumer.Run),
		queue.WithRunImmediatePriority(32),
	}

	// If we're in LIVE mode, then it's okay to read/write tasks to the
	// database. But we don't want to do this in SETUP mode because
	// the setup tool should not run queued tasks from production.
	if factory.IsLiveMode() {
		mongoStorage := queue_mongo.New(factory.commonDatabase, 16, 8)

		// Apply the storage to the queue
		options = append(options,
			queue.WithStorage(mongoStorage),
			queue.WithPollStorage(true),
		)
	}

	// Create a new queue object with consumers, storage, and polling
	factory.queue = queue.New(options...)
}

/******************************************
 * Server Config Methods
 ******************************************/

// Config returns the current configuration for the Factory
func (factory *Factory) Config() config.Config {
	result := factory.config
	return result
}

// UpdateConfig updates the configuration for the Factory
func (factory *Factory) UpdateConfig(value config.Config) error {

	const location = "server.factory.UpdateConfig"

	factory.config = value

	if err := factory.storage.Write(value); err != nil {
		return derp.Wrap(err, location, "Unable to write configuration", value)
	}

	return nil
}

/******************************************
 * Domain Methods
 ******************************************/

func (factory *Factory) RangeDomains() iter.Seq[*service.Factory] {

	return func(yield func(*service.Factory) bool) {

		factory.domains.Range(func(key string, domain *service.Factory) bool {
			return yield(domain)
		})
	}
}

// ListDomains returns a list of all domains in the Factory
func (factory *Factory) ListDomains() []config.Domain {
	return factory.config.Domains
}

// PutDomain adds a domain to the Factory
func (factory *Factory) PutDomain(configuration config.Domain) error {

	const location = "server.Factory.PutDomain"

	// Save the domain info ant write a new configuration to the storage service
	if err := factory.putDomain(configuration); err != nil {
		return derp.Wrap(err, location, "Unable to add domain", configuration)
	}

	// The storage service will trigger a new configuration via the Subscrbe() channel,
	// But we still want to call the owner update manually.

	domainFactory, err := factory.ByHostname(configuration.Hostname)

	if err != nil {
		return derp.Wrap(err, location, "Unable to get domain factory", configuration.Hostname)
	}

	// If the config includes a database owner, then guarantee they're written into the database
	if !configuration.Owner.IsEmpty() {

		ctx, cancel := timeoutContext(30)
		defer cancel()

		_, err = domainFactory.Server().WithTransaction(ctx, func(session data.Session) (any, error) {
			userService := domainFactory.User()
			if err := userService.SetOwner(session, configuration.Owner); err != nil {
				return nil, derp.Wrap(err, location, "Unable to set owner", configuration.Owner)
			}
			return nil, nil
		})

		if err != nil {
			return derp.Wrap(err, location, "Unable to write database owner")
		}

		return nil
	}

	return nil
}

// putDomain is a helper for PutDomain that manages the locking
func (factory *Factory) putDomain(configuration config.Domain) error {

	const location = "server.Factory.putDomain"

	// Add the domain to the collection
	factory.config.Domains.Put(configuration)

	// Try to write the configuration to the storage service
	if err := factory.storage.Write(factory.config); err != nil {
		return derp.Wrap(err, location, "Unable to write configuration")
	}

	// Try to update the domain in the in-memory cache
	if err := factory.refreshDomain(configuration); err != nil {
		return derp.Wrap(err, location, "Unable to refresh domain", configuration)
	}

	return nil
}

// FindDomain finds a domain in the configuration by its ID
func (factory *Factory) FindDomain(domainID string) (config.Domain, error) {

	const location = "server.Factory.FindDomain"

	// If "new" then create a new domain
	if strings.ToLower(domainID) == "new" {
		return config.NewDomain(), nil
	}

	// Search for the domain in the configuration
	if domain, ok := factory.config.Domains.Get(domainID); ok {
		return domain, nil
	}

	// Not found, so return an error
	return config.NewDomain(), derp.NotFoundError(location, "Unable to find Domain", domainID)
}

// DeleteDomain removes a domain from the Factory
func (factory *Factory) DeleteDomain(domainID string) error {

	const location = "server.Factory.DeleteDomain"

	// Remove the domain from the cache
	factory.domains.Delete(domainID)

	// Delete the domain from the collection
	factory.config.Domains.Delete(domainID)

	// Write changes to the storage engine.
	if err := factory.storage.Write(factory.config); err != nil {
		return derp.Wrap(err, location, "Unable to save configuration")
	}

	return nil
}

/******************************************
 * Factory Methods
 ******************************************/

// ByDomainID retrieves a Domain factory using a DomainID
func (factory *Factory) ByDomainID(domainID string) (config.Domain, *service.Factory, error) {

	const location = "server.Factory.ByDomainID"

	// Look up the domain name for this domainID
	domainConfig, err := factory.FindDomain(domainID)

	if err != nil {
		return config.Domain{}, nil, derp.Wrap(err, location, "Domain is invalid", domainID)
	}

	// Return the domain
	result, err := factory.ByHostname(domainConfig.Hostname)

	if err != nil {
		return config.Domain{}, nil, derp.Wrap(err, location, "Hostname is invalid", domainConfig.Hostname)
	}

	return domainConfig, result, nil
}

// ByContext retrieves a Domain factory using an echo.Context
func (factory *Factory) ByContext(ctx echo.Context) (*service.Factory, error) {
	return factory.ByRequest(ctx.Request())
}

func (factory *Factory) ByRequest(req *http.Request) (*service.Factory, error) {

	const location = "server.Factory.ByRequest"

	hostname := dt.TrueHostname(req)
	result, err := factory.ByHostname(hostname)

	if err != nil {
		return nil, derp.Wrap(err, location, "Hostname is invalid", "hostname: "+hostname)
	}

	return result, nil
}

// ByHostname retrieves a Domain factory using a Hostname
func (factory *Factory) ByHostname(hostname string) (*service.Factory, error) {

	const location = "server.Factory.ByHostname"

	// Clean up the hostname before using it
	hostname = factory.normalizeHostname(hostname)

	// Try to find the domain in the configuration
	if domain, exists := factory.domains.Load(hostname); exists {
		return domain, nil
	}

	// Failure.
	return nil, derp.MisdirectedRequestError(location, "Hostname is invalid", "hostname: "+hostname)
}

// normalizeHostname removes inconsistencies in host names so that they
// can be compared against the domain registry.
func (factory *Factory) normalizeHostname(hostname string) string {

	hostname, _, _ = strings.Cut(hostname, ":")     // Remove port number
	hostname = strings.TrimPrefix(hostname, "www.") // Remove leading "www"
	hostname = strings.ToLower(hostname)            // Force lowercase

	// Now isn't that pretty?
	return hostname
}

/******************************************
 * Other Global Services
 ******************************************/

// Contet returns the global content service
func (factory *Factory) Content() *service.Content {
	return &factory.contentService
}

// Queue returns the gloabl message queue service
func (factory *Factory) Queue() *queue.Queue {
	return factory.queue
}

// Registration returns the global template service
func (factory *Factory) Registration() *service.Registration {
	return &factory.registrationService
}

// Template returns the global template service
func (factory *Factory) Template() *service.Template {
	return &factory.templateService
}

// Theme returns the global theme service
func (factory *Factory) Theme() *service.Theme {
	return &factory.themeService
}

// Widget returns the global widget service
func (factory *Factory) Widget() *service.Widget {
	return &factory.widgetService
}

// FuncMap returns the global funcMap (used by all templates)
func (factory *Factory) FuncMap() template.FuncMap {
	return service.FuncMap(factory.Icons())
}

// Icons returns the global icon collection
func (factory *Factory) Icons() icon.Provider {
	return service.Icons{}
}

// Filesystem returns the global filesystem service
func (factory *Factory) Filesystem() service.Filesystem {
	return service.NewFilesystem(factory.embeddedFiles)
}

// Email returns the global email service
func (factory *Factory) Email() *service.ServerEmail {
	return &factory.emailService
}

// EditorJS returns the EditorJS adapter for the Content service
func (factory *Factory) EditorJS() *goeditorjs.HTMLEngine {
	result := goeditorjs.NewHTMLEngine()

	result.RegisterBlockHandlers(
		&goeditorjs.HeaderHandler{},
		&goeditorjs.ParagraphHandler{},
		&goeditorjs.ListHandler{},
		&goeditorjs.ImageHandler{},
		&goeditorjs.RawHTMLHandler{},
	)

	return result
}

func (factory *Factory) DigitalDome() *dome.Dome {
	return &factory.digitalDome
}

func (factory *Factory) HTTPCache() *httpcache.HTTPCache {
	return &factory.httpCache
}

// CommonDatabase returns a link to the common database server
func (factory *Factory) CommonDatabase() *mongo.Database {
	return factory.commonDatabase
}

func (factory *Factory) Server(hostname string) (data.Server, error) {

	const location = "server.Factory.Server"

	// Clean up the hostname before using it
	hostname = factory.normalizeHostname(hostname)

	// Try to find the domain in the configuration
	if domain, exists := factory.domains.Load(hostname); exists {
		return domain.Server(), nil
	}

	// Failure.
	return nil, derp.MisdirectedRequestError(location, "Hostname is invalid", "hostname: "+hostname)

}

// Session creates a new database session
func (factory *Factory) Session(ctx context.Context, hostname string) (data.Session, error) {

	const location = "server.factory.Session"

	// Locate the server from the factory
	server, err := factory.Server(hostname)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to retrieve database connection.", hostname)
	}

	// Create a database session with the server
	session, err := server.Session(ctx)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to create database session for server", hostname)
	}

	// Return the session to the caller
	return session, nil
}

// IsLiveMode returns TRUE if the server is serving real websites, and not the setup mode.
func (factory *Factory) IsLiveMode() bool {
	return !factory.setup
}

// IsSetupMode returns TRUE if the server is in setup mode, and is not serving real websites.
func (factory *Factory) IsSetupMode() bool {
	return factory.setup
}

// IsReadyForDomains returns TRUE if the basic server config is done
// and is ready for domains to be added to the server.
func (factory *Factory) IsReadyForDomains() bool {
	return factory.config.IsReadyForDomains()
}

/******************************************
 * Helper Methods
 ******************************************/

func (factory *Factory) port(domainConfig config.Domain) string {

	// If not localhost, then use standard ports and assume the
	// hosting environment will handle the port forwarding
	if !dt.IsLocalhost(domainConfig.Hostname) {
		return ""
	}

	// If using localhosts, then return the port number if it's not 80
	switch factory.config.HTTPPort {
	case 0, 80:
		return ""

	default:
		return ":" + strconv.Itoa(factory.config.HTTPPort)
	}
}
