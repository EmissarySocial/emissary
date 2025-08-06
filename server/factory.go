package server

import (
	"context"
	"embed"
	"html/template"
	"iter"
	"maps"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/consumer"
	"github.com/EmissarySocial/emissary/domain"
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
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/turbine/queue_mongo"
	"github.com/davidscottmills/goeditorjs"
	"github.com/labstack/echo/v4"
	"github.com/maypok86/otter"
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
	mutex   sync.RWMutex
	ready   chan struct{}

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
	queue               queue.Queue
	digitalDome         dome.Dome

	domains   map[string]*domain.Factory
	httpCache httpcache.HTTPCache
}

// NewFactory uses the provided configuration data to generate a new Factory
// if there are any errors connecting to a domain's datasource, NewFactory will derp.Report
// the error, but will continue loading without those domains.
func NewFactory(storage config.Storage, embeddedFiles embed.FS) *Factory {

	factory := Factory{
		storage:       storage,
		mutex:         sync.RWMutex{},
		domains:       make(map[string]*domain.Factory),
		embeddedFiles: embeddedFiles,
		ready:         make(chan struct{}),
		jwtService:    service.NewJWT(),
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

	go factory.start()

	return &factory
}

func (factory *Factory) start() {

	log.Info().Msg("Factory: waiting for configuration...")

	filesystemService := factory.Filesystem()

	// Read configuration files from the channel
	for config := range factory.storage.Subscribe() {

		// Set timeout threshold for slow queries
		mongodb.SetLogTimeout(config.LogSlowQueries)

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

		if attachmentOriginals, err := filesystemService.GetAfero(config.AttachmentOriginals); err == nil {
			factory.attachmentOriginals = attachmentOriginals
		} else {
			derp.Report(derp.Wrap(err, "server.Factory.start", "Error getting attachment original directory", config))
		}

		if attachmentCache, err := filesystemService.GetAfero(config.AttachmentCache); err == nil {
			factory.attachmentCache = attachmentCache
		} else {
			derp.Report(derp.Wrap(err, "server.Factory.start", "Error getting attachment cache directory", config))
		}

		if exportCache, err := filesystemService.GetAfero(config.ExportCache); err == nil {
			factory.exportCache = exportCache
		} else {
			derp.Report(derp.Wrap(err, "server.Factory.start", "Error getting export cache directory", config))
		}

		factory.config = config

		// Mark all domains for deletion (then unmark them later)
		for index := range factory.domains {
			factory.domains[index].MarkForDeletion = true
		}

		// Refresh cached values in global services
		factory.emailService.Refresh()
		factory.templateService.Refresh(config.Templates)

		if err := factory.refreshCommonDatabase(config.ActivityPubCache); err != nil {
			derp.Report(derp.Wrap(err, "server.Factory.start", "WARNING: Could not refresh common database.  Important services (like queued tasks and ActivityPub caching) may not function correctly.", config.ActivityPubCache))
		}

		factory.refreshQueue()

		// Add logging to the Silicon Dome WAF
		if factory.commonDatabase != nil {

			log.Trace().Msg("Setting up Common Database")

			// JWT Service configuration
			server := mongodb.NewServer(factory.commonDatabase)
			factory.jwtService.Refresh(server, config.MasterKey)

			// Digital Dome configuration
			collection := mongodb.NewCollection(factory.commonDatabase.Collection("DigitalDome"))
			factory.digitalDome.With(dome.LogDatabase(collection))

			// Derp configuration
			derp.Plugins.Clear()
			for _, logger := range config.Loggers {

				switch logger.GetString("type") {

				case "console":
					log.Trace().Msg("Adding console logger to DERP")
					derp.Plugins.Add(derpconsole.New())

				case "mongo":
					log.Trace().Msg("Adding mongo logger to DERP")
					derp.Plugins.Add(derpmongo.New(
						factory.commonDatabase.Collection("ErrorLog"),
						logger))

				default:
					log.Error().Str("type", logger.GetString("type")).Msg("Unknown logging type")
				}
			}
		}

		// Insert/Update a factory for each domain in the configuration
		for _, domainConfig := range config.Domains {

			factory.mutex.Lock()
			if err := factory.refreshDomain(domainConfig); err != nil {
				derp.Report(derp.Wrap(err, "server.Factory.start", "Error refreshing domain", domainConfig.ID))
			}
			factory.mutex.Unlock()
		}

		// Remove any domains that are still marked for deletion
		for domainID := range factory.domains {
			factory.mutex.Lock()
			if factory.domains[domainID].MarkForDeletion {
				delete(factory.domains, domainID)
			}
			factory.mutex.Unlock()
		}

		// Bootstrap the "Scheduler" task.  Duplicates will be dropped.
		// This task will be used to schedule all other daily/hourly tasks
		task := queue.NewTask("Scheduler", mapof.NewAny())
		if err := factory.queue.Publish(task); err != nil {
			derp.Report(derp.Wrap(err, "server.Factory.start", "Error publishing Schedule task"))
		}

		// If the "ready" channel is still open, then close it,
		// which will unblock any waiting processes
		if !channel.Closed(factory.ready) {
			close(factory.ready)
		}
	}
}

// Ready returns a channel that is held open while the Factory is still initializing
// and is closed (unblocking waiting processes) once the Factory is ready to use
func (factory *Factory) Ready() <-chan struct{} {
	return factory.ready
}

// refreshDomain attempts to refresh an existing domain, or creates a new one if it doesn't exist
// CALLS TO THIS MUST BE LOCKED
func (factory *Factory) refreshDomain(domainConfig config.Domain) error {

	const location = "server.factory.refreshDomain"

	// Try to find the domain
	if existing := factory.domains[domainConfig.Hostname]; existing != nil {

		// Even if there's an error "refreshing" the domain, we don't want to delete it
		existing.MarkForDeletion = false

		// Try to refresh the domain
		if err := existing.Refresh(domainConfig, factory.attachmentOriginals, factory.attachmentCache); err != nil {
			return derp.Wrap(err, location, "Error refreshing domain", domainConfig.Hostname)
		}

		return nil
	}

	// Fall through means that the domain does not exist, so we need to create it
	newDomain, err := domain.NewFactory(
		factory,
		mongodb.NewServer(factory.commonDatabase),
		domainConfig,
		factory.port(domainConfig),
		&factory.contentService,
		&factory.emailService,
		&factory.jwtService,
		&factory.queue,
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
	factory.domains[newDomain.Hostname()] = newDomain

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
		return derp.InternalError(location, "Must have a URI for the common database")
	}

	// RULE: Must have a database name
	if database == "" {
		return derp.InternalError(location, "Must have a database name for the common database")
	}

	// If there is already a cache connection in place, then close it before we open a new one
	if factory.commonDatabase != nil {
		log.Trace().Str("database", database).Msg("Resetting common database")
		if err := factory.commonDatabase.Client().Disconnect(context.Background()); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to disconnect from database"))
		}
	}

	// Try to connect to the cache database
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))

	if err != nil {
		return derp.Wrap(err, location, "Unable to connect to database", uri)
	}

	log.Trace().Msg("Connected to common database")
	factory.commonDatabase = client.Database(database)

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
		queue.WithConsumers(consumer.Run, remote.Consumer()),
		queue.WithRunImmediatePriority(32),
	}

	// If we have a common database configured, then use it for queue storage
	if factory.commonDatabase != nil {

		// Set up Queue storage
		mongoStorage := queue_mongo.New(factory.commonDatabase, 32, 32)

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

	// Read lock the mutex
	factory.mutex.RLock()
	defer factory.mutex.RUnlock()

	result := factory.config
	return result
}

// UpdateConfig updates the configuration for the Factory
func (factory *Factory) UpdateConfig(value config.Config) error {

	// Write lock the mutex
	factory.mutex.Lock()
	defer factory.mutex.Unlock()

	factory.config = value

	if err := factory.storage.Write(value); err != nil {
		return derp.Wrap(err, "server.factory.UpdateConfig", "Error writing configuration", value)
	}

	return nil
}

/******************************************
 * Domain Methods
 ******************************************/

func (factory *Factory) RangeDomains() iter.Seq[*domain.Factory] {

	return func(yield func(*domain.Factory) bool) {
		factory.mutex.RLock()
		defer factory.mutex.RUnlock()

		for _, domain := range factory.domains {
			if !yield(domain) {
				return
			}
		}
	}
}

// ListDomains returns a list of all domains in the Factory
func (factory *Factory) ListDomains() []config.Domain {
	return factory.config.Domains
}

// PutDomain adds a domain to the Factory
func (factory *Factory) PutDomain(session data.Session, configuration config.Domain) error {

	// Save the domain info ant write a new configuration to the storage service
	if err := factory.putDomain(configuration); err != nil {
		return derp.Wrap(err, "server.Factory.PutDomain", "Error adding domain", configuration)
	}

	// The storage service will trigger a new configuration via the Subscrbe() channel,
	// But we still want to call the owner update manually.

	domainFactory, err := factory.ByHostname(configuration.Hostname)

	if err != nil {
		return derp.Wrap(err, "server.Factory.PutDomain", "Error getting domain factory", configuration.Hostname)
	}

	userService := domainFactory.User()
	if err := userService.SetOwner(session, configuration.Owner); err != nil {
		return derp.Wrap(err, "server.Factory.PutDomain", "Error setting owner", configuration.Owner)
	}

	return nil
}

// putDomain is a helper for PutDomain that manages the locking
func (factory *Factory) putDomain(configuration config.Domain) error {

	factory.mutex.Lock()
	defer factory.mutex.Unlock()

	// Add the domain to the collection
	factory.config.Domains.Put(configuration)

	// Try to write the configuration to the storage service
	if err := factory.storage.Write(factory.config); err != nil {
		return derp.Wrap(err, "server.Factory.putDomain", "Error writing configuration")
	}

	// Try to update the domain in the in-memory cache
	if err := factory.refreshDomain(configuration); err != nil {
		return derp.Wrap(err, "server.Factory.putDomain", "Error refreshing domain", configuration)
	}

	return nil
}

// DomainByID finds a domain in the configuration by its ID
func (factory *Factory) DomainByID(domainID string) (config.Domain, error) {

	factory.mutex.RLock()
	defer factory.mutex.RUnlock()

	// If "new" then create a new domain
	if strings.ToLower(domainID) == "new" {
		return config.NewDomain(), nil
	}

	// Search for the domain in the configuration
	if domain, ok := factory.config.Domains.Get(domainID); ok {
		return domain, nil
	}

	// Not found, so return an error
	return config.NewDomain(), derp.NotFoundError("server.Factory.DomainByID", "DomainID not found", domainID)
}

// DeleteDomain removes a domain from the Factory
func (factory *Factory) DeleteDomain(domainID string) error {

	factory.mutex.Lock()
	defer factory.mutex.Unlock()

	// Delete the domain from the collection
	factory.config.Domains.Delete(domainID)

	// Write changes to the storage engine.
	if err := factory.storage.Write(factory.config); err != nil {
		return derp.Wrap(err, "server.Factory.DeleteDomain", "Error saving configuration")
	}

	return nil
}

/******************************************
 * Factory Methods
 ******************************************/

// ByDomainID retrieves a Domain factory using a DomainID
func (factory *Factory) ByDomainID(domainID string) (config.Domain, *domain.Factory, error) {

	const location = "server.Factory.ByDomainID"

	// Look up the domain name for this domainID
	domainConfig, err := factory.DomainByID(domainID)

	if err != nil {
		return config.Domain{}, nil, derp.Wrap(err, location, "Invalid domain", domainID)
	}

	// Return the domain
	result, err := factory.ByHostname(domainConfig.Hostname)

	if err != nil {
		return config.Domain{}, nil, derp.Wrap(err, location, "Invalid hostname", domainConfig.Hostname)
	}

	return domainConfig, result, nil
}

// ByContext retrieves a Domain factory using an echo.Context
func (factory *Factory) ByContext(ctx echo.Context) (*domain.Factory, error) {
	return factory.ByRequest(ctx.Request())
}

func (factory *Factory) ByRequest(req *http.Request) (*domain.Factory, error) {

	const location = "server.Factory.ByRequest"

	hostname := dt.Hostname(req)
	result, err := factory.ByHostname(hostname)

	if err != nil {
		return nil, derp.Wrap(err, location, "Invalid hostname", "hostname: "+hostname)
	}

	return result, nil
}

// ByHostname retrieves a Domain factory using a Hostname
func (factory *Factory) ByHostname(hostname string) (*domain.Factory, error) {

	const location = "server.Factory.ByHostname"

	// Clean up the hostname before using it
	hostname = factory.normalizeHostname(hostname)

	// Read Lock the mutex to prevent concurrent writes
	factory.mutex.RLock()
	defer factory.mutex.RUnlock()

	// Try to find the domain in the configuration
	if domain, exists := factory.domains[hostname]; exists {
		return domain, nil
	}

	// Failure.
	return nil, derp.MisdirectedRequestError(location, "Invalid hostname", "hostname: "+hostname, maps.Keys(factory.domains))
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
	return &factory.queue
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
	return build.FuncMap(factory.Icons())
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

func (factory *Factory) ActivityCollection() *mongo.Collection {

	if factory.commonDatabase != nil {
		return factory.commonDatabase.Collection("Document")
	}

	return nil
}

/*
// Steranko implements the steranko.Factory method, used for locating the specific
// steranko instance used by a domain.
func (factory *Factory) Steranko(ctx echo.Context) (*steranko.Steranko, error) {

		result, err := factory.ByContext(ctx)

		if err != nil {
			return nil, derp.Wrap(err, "server.Factory.Steranko", "Invalid hostname")
		}

		return result.Steranko(), nil
	}
*/

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

	// Read Lock the mutex to prevent concurrent writes
	factory.mutex.RLock()
	defer factory.mutex.RUnlock()

	// Try to find the domain in the configuration
	if domain, exists := factory.domains[hostname]; exists {
		return domain.Server(), nil
	}

	// Failure.
	return nil, derp.MisdirectedRequestError(location, "Invalid hostname", "hostname: "+hostname, maps.Keys(factory.domains))

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
