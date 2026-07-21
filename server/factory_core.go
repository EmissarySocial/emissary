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
	"github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome"
	"github.com/benpate/icon"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/turbine/queue_mongo"
	"github.com/benpate/uri"
	"github.com/davidscottmills/goeditorjs"
	"github.com/labstack/echo/v4"
	"github.com/maypok86/otter"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/realclientip/realclientip-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// factoryCore holds the state and behavior shared by every server mode
// (live, setup, and any future mode). Mode structs embed it and add their
// own lifecycle; see emissary-specs/FACTORY-MODES.md.
type factoryCore struct {
	storage config.Storage
	config  config.Config

	// Server-level services
	contentService      service.Content
	emailService        service.ServerEmail
	iconService         service.Icons
	jwtService          service.JWT
	registrationService service.Registration
	themeService        service.Theme
	templateService     service.Template
	widgetService       service.Widget

	embeddedFiles       embed.FS
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs
	exportCache         afero.Fs
	commonDatabase      *mongo.Database // nillable: connected by each mode's lifecycle, so it may be nil before that happens (FACTORY-MODES D7)
	workingDirectory    *mediaserver.WorkingDirectory
	queue               *queue.Queue
	digitalDome         *dome.Dome
	clientIPStrategy    realclientip.Strategy

	funcMap   template.FuncMap
	domains   *xsync.Map[string, *service.Factory]
	httpCache httpcache.HTTPCache
}

/******************************************
 * Server Config Methods
 ******************************************/

// Config returns the current configuration for the Factory
func (factory *factoryCore) Config() config.Config {
	result := factory.config
	return result
}

// AllowPrivateIPs reports whether outbound ActivityPub delivery may connect to
// non-public (private/loopback) addresses. FALSE in production; enabled only for
// local/dev federation between machines on a private network.
func (factory *factoryCore) AllowPrivateIPs() bool {
	return factory.config.AllowPrivateIPs
}

// UpdateConfig updates the configuration for the Factory
func (factory *factoryCore) UpdateConfig(value config.Config) error {

	const location = "server.factory.UpdateConfig"

	factory.config = value

	if err := factory.storage.Write(value); err != nil {
		return derp.Wrap(err, location, "Writing configuration", value)
	}

	return nil
}

/******************************************
 * Domain Methods
 ******************************************/

func (factory *factoryCore) RangeDomains() iter.Seq[*service.Factory] {

	return func(yield func(*service.Factory) bool) {

		factory.domains.Range(func(key string, domain *service.Factory) bool {
			return yield(domain)
		})
	}
}

// ListDomains returns a list of all domains in the Factory
func (factory *factoryCore) ListDomains() []config.Domain {
	return factory.config.Domains
}

// TestConnection verifies that a domain's database is actually reachable.  It is used to
// validate a domain BEFORE it is persisted, so a bad connect string fails fast with a clear
// message instead of hanging on the driver's default (~30s) server-selection timeout and
// leaving a broken domain in the configuration file.  mongo.Connect is lazy -- it never
// contacts the server -- so we must Ping to actually exercise the connection.
func (factory *factoryCore) TestConnection(configuration config.Domain) error {
	return testDatabaseConnection(configuration, 5*time.Second)
}

// testDatabaseConnection pings the domain's database, bounding server selection and the
// ping itself to `timeout` so an unreachable host fails in seconds.  The timeout is a
// parameter so tests can drive the fail-fast path quickly.
func testDatabaseConnection(configuration config.Domain, timeout time.Duration) error {

	const location = "server.Factory.TestConnection"

	// Bound server selection so an unreachable host fails in seconds, not the ~30s default.
	opts := options.Client().SetServerSelectionTimeout(timeout)

	server, err := mongodb.New(configuration.ConnectString, configuration.DatabaseName, opts)

	if err != nil {
		return derp.Wrap(err, location, "Connecting to the database. Please check the connect string.")
	}

	client := server.Client()

	// Always release the throwaway client created for this check.
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Ping forces real server selection, surfacing an unreachable host or replica set now.
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return derp.Wrap(err, location, `Unable to reach the database. Check the connect string — a single-member replica set needs "?directConnection=true".`)
	}

	return nil
}

// PutDomain adds a domain to the Factory
func (factory *factoryCore) PutDomain(configuration config.Domain) error {

	const location = "server.Factory.PutDomain"

	// RULE: Domains cannot be added until the common (ActivityPub Cache) database is
	// connected, because every domain factory stores shared caches there.  Checked BEFORE
	// the config write so a rejected domain never persists half-added (FACTORY-MODES D6),
	// and returned UN-wrapped so the message reaches the setup console user verbatim.
	if factory.commonDatabase == nil {
		return derp.BadRequest(location, "Configure the ActivityPub Cache database before adding domains")
	}

	// Save the domain info ant write a new configuration to the storage service
	if err := factory.putDomain(configuration); err != nil {
		return derp.Wrap(err, location, "Adding domain", configuration)
	}

	// The storage service will trigger a new configuration via the Subscrbe() channel,
	// But we still want to call the owner update manually.

	domainFactory, err := factory.ByHostname(configuration.Hostname)

	if err != nil {
		return derp.Wrap(err, location, "Getting domain factory", configuration.Hostname)
	}

	// If the config includes a database owner, then guarantee they're written into the database
	if !configuration.Owner.IsEmpty() {

		ctx, cancel := timeoutContext(30)
		defer cancel()

		_, err = domainFactory.WithTransaction(ctx, func(session data.Session) (any, error) {
			userService := domainFactory.User()
			if err := userService.SetOwner(session, configuration.Owner); err != nil {
				return nil, derp.Wrap(err, location, "Setting owner", configuration.Owner)
			}
			return nil, nil
		})

		if err != nil {
			return derp.Wrap(err, location, "Writing database owner")
		}

		return nil
	}

	return nil
}

// putDomain is a helper for PutDomain that manages the locking
func (factory *factoryCore) putDomain(configuration config.Domain) error {

	const location = "server.Factory.putDomain"

	// Add the domain to the collection.  (The D6 common-database guard runs in PutDomain,
	// before any of this.)
	factory.config.Domains.Put(configuration)

	// Try to write the configuration to the storage service
	if err := factory.storage.Write(factory.config); err != nil {
		return derp.Wrap(err, location, "Writing configuration")
	}

	// Try to update the domain in the in-memory cache
	if err := factory.refreshDomain(configuration); err != nil {
		return derp.Wrap(err, location, "Refreshing domain", configuration)
	}

	return nil
}

// FindDomain finds a domain in the configuration by its ID
func (factory *factoryCore) FindDomain(domainID string) (config.Domain, error) {

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
	return config.NewDomain(), derp.NotFound(location, "Unable to find Domain", domainID)
}

// DeleteDomain removes a domain from the Factory
func (factory *factoryCore) DeleteDomain(domainID string) error {

	const location = "server.Factory.DeleteDomain"

	// Remove the domain from the cache
	factory.domains.Delete(domainID)

	// Delete the domain from the collection
	factory.config.Domains.Delete(domainID)

	// Write changes to the storage engine.
	if err := factory.storage.Write(factory.config); err != nil {
		return derp.Wrap(err, location, "Saving configuration")
	}

	return nil
}

// refreshDomain attempts to refresh an existing domain, or creates a new one if it doesn't exist
func (factory *factoryCore) refreshDomain(domainConfig config.Domain) error {

	const location = "server.factory.refreshDomain"

	// Try to find the domain
	if domain, exists := factory.domains.Load(domainConfig.Hostname); exists {

		// Even if there's an error "refreshing" the domain, we don't want to delete it
		domain.MarkForDeletion = false

		// Try to refresh the domain
		if err := domain.Refresh(domainConfig, factory.attachmentOriginals, factory.attachmentCache); err != nil {
			return derp.Wrap(err, location, "Refreshing domain", domainConfig.Hostname)
		}

		return nil
	}

	// RULE: Creating a domain factory requires the common database.  Callers gate on this
	// too (putDomain, the mode lifecycles); this is defense in depth against a nil-pointer
	// panic inside mongodb.NewServer.
	if factory.commonDatabase == nil {
		return derp.Internal(location, "Common database must be connected before creating domains")
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
		factory.workingDirectory,
	)

	if err != nil {
		return derp.Wrap(err, location, "Refreshing configuration", domainConfig)
	}

	// If there are no errors, then add the domain to the list.
	factory.domains.Store(newDomain.Hostname(), newDomain)

	return nil
}

/******************************************
 * Factory Methods
 ******************************************/

// ByDomainID retrieves a Domain factory using a DomainID
func (factory *factoryCore) ByDomainID(domainID string) (config.Domain, *service.Factory, error) {

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
func (factory *factoryCore) ByContext(ctx echo.Context) (*service.Factory, error) {
	return factory.ByRequest(ctx.Request())
}

// ByRequest retrieves a Domain factory using an http.Request
func (factory *factoryCore) ByRequest(req *http.Request) (*service.Factory, error) {

	const location = "server.Factory.ByRequest"

	hostname := factory.Hostname(req)
	result, err := factory.ByHostname(hostname)

	if err != nil {
		return nil, derp.Wrap(err, location, "Hostname is invalid", "hostname: "+hostname)
	}

	return result, nil
}

// ByHostname retrieves a Domain factory using a Hostname
func (factory *factoryCore) ByHostname(hostname string) (*service.Factory, error) {

	const location = "server.Factory.ByHostname"

	// Clean up the hostname before using it
	hostname = factory.normalizeHostname(hostname)

	// Try to find the domain in the configuration
	if domain, exists := factory.domains.Load(hostname); exists {
		return domain, nil
	}

	// Failure.
	return nil, derp.MisdirectedRequest(location, "Hostname is invalid", "hostname: "+hostname)
}

// normalizeHostname removes inconsistencies in host names so that they
// can be compared against the domain registry.
func (factory *factoryCore) normalizeHostname(hostname string) string {

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
func (factory *factoryCore) Content() *service.Content {
	return &factory.contentService
}

// Queue returns the gloabl message queue service
func (factory *factoryCore) Queue() *queue.Queue {
	return factory.queue
}

// Registration returns the global template service
func (factory *factoryCore) Registration() *service.Registration {
	return &factory.registrationService
}

// Template returns the global template service
func (factory *factoryCore) Template() *service.Template {
	return &factory.templateService
}

// Theme returns the global theme service
func (factory *factoryCore) Theme() *service.Theme {
	return &factory.themeService
}

// Widget returns the global widget service
func (factory *factoryCore) Widget() *service.Widget {
	return &factory.widgetService
}

// FuncMap returns the global funcMap (used by all templates)
func (factory *factoryCore) FuncMap() template.FuncMap {
	return factory.funcMap
}

// Icons returns the global icon collection
func (factory *factoryCore) Icons() icon.Provider {
	return factory.iconService
}

// Filesystem returns the global filesystem service
func (factory *factoryCore) Filesystem() service.Filesystem {
	return service.NewFilesystem(factory.embeddedFiles)
}

// Email returns the global email service
func (factory *factoryCore) Email() *service.ServerEmail {
	return &factory.emailService
}

// EditorJS returns the EditorJS adapter for the Content service
func (factory *factoryCore) EditorJS() *goeditorjs.HTMLEngine {
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

func (factory *factoryCore) DigitalDome() *dome.Dome {
	return factory.digitalDome
}

func (factory *factoryCore) HTTPCache() *httpcache.HTTPCache {
	return &factory.httpCache
}

// CommonDatabase returns a link to the common database server
func (factory *factoryCore) CommonDatabase() *mongo.Database {
	return factory.commonDatabase
}

func (factory *factoryCore) Server(hostname string) (data.Server, error) {

	const location = "server.Factory.Server"

	// Clean up the hostname before using it
	hostname = factory.normalizeHostname(hostname)

	// Try to find the domain in the configuration
	if domain, exists := factory.domains.Load(hostname); exists {
		return domain.Server(), nil
	}

	// Failure.
	return nil, derp.MisdirectedRequest(location, "Hostname is invalid", "hostname: "+hostname)

}

// Session creates a new database session
func (factory *factoryCore) Session(ctx context.Context, hostname string) (data.Session, error) {

	const location = "server.factory.Session"

	// Locate the server from the factory
	server, err := factory.Server(hostname)

	if err != nil {
		return nil, derp.Wrap(err, location, "Retrieving database connection.", hostname)
	}

	// Create a database session with the server
	session, err := server.Session(ctx)

	if err != nil {
		return nil, derp.Wrap(err, location, "Creating database session for server", hostname)
	}

	// Return the session to the caller
	return session, nil
}

// IsReadyForDomains returns TRUE if the basic server config is done
// and is ready for domains to be added to the server.
func (factory *factoryCore) IsReadyForDomains() bool {
	return factory.config.IsReadyForDomains()
}

func (factory *factoryCore) calcClientIPStrategy(config config.Config) realclientip.Strategy {

	const location = "server.Factory.ClientIPStrategy"

	var strategy realclientip.Strategy
	var err error

	switch config.ClientIPStrategy {

	case "REMOTE-ADDR":
		return realclientip.RemoteAddrStrategy{}

	case "RIGHTMOST-TRUSTED-COUNT":
		strategy, err = realclientip.NewRightmostTrustedCountStrategy("X-Forwarded-For", config.ClientIPTrustedCount)

	case "SINGLE-IP-HEADER":
		strategy, err = realclientip.NewSingleIPHeaderStrategy(config.ClientIPHeader)

	default:
		err = derp.Internal(location, "Unknown Client IP strategy", config.ClientIPStrategy)
	}

	// If there is no error, then
	if err != nil {
		derp.Report(derp.Wrap(err, location, "Creating Client IP strategy", config.ClientIPStrategy))
		return realclientip.RemoteAddrStrategy{}
	}

	return strategy
}

// ClientIP returns the client's real IP address using the strategy defined in the configuration file
func (factory *factoryCore) ClientIP(request *http.Request) string {

	if factory.clientIPStrategy == nil {
		derp.Report(derp.Internal("server.Factory.ClientIPStrategy", "Client IP strategy cannot be nil"))
		return ""
	}

	return factory.clientIPStrategy.ClientIP(request.Header, request.RemoteAddr)
}

// Hostname returns the hostname for the request.
func (factory *factoryCore) Hostname(request *http.Request) string {

	// If the server config includes TrustForwardedHost, then the X-Forwarded-Host header is used.
	if factory.config.TrustForwardedHost {
		if forwardedHost := request.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
			return forwardedHost
		}
	}

	// Default is to use the standard Host header
	return request.Host
}

/******************************************
 * Helper Methods
 ******************************************/

func (factory *factoryCore) port(domainConfig config.Domain) string {

	// If not localhost, then use standard ports and assume the
	// hosting environment will handle the port forwarding
	if !uri.IsLocalHostname(domainConfig.Hostname) {
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

/******************************************
 * Shared Lifecycle Steps
 ******************************************/

// setLogLevel applies the configured global logging level
func setLogLevel(config config.Config) {

	switch config.DebugLevel {

	case "Trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "Debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "Info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "Warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "Error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "Fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "Panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)

	// RULE: Only an explicit "None" disables logging entirely
	case "None":
		zerolog.SetGlobalLevel(zerolog.Disabled)

	// RULE: Unrecognized values (typos, old configs) fall back to Info -- never to a
	// silent server, which reads as a hang.
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// refreshCommonDatabase updates the connection to the common database
func (factory *factoryCore) refreshCommonDatabase(connection mapof.String) error {

	const location = "server.factory.refreshCommonDatabase"

	// Collect arguments from the connection config
	uri := connection.GetString("connectString")
	database := connection.GetString("database")

	// RULE: Must have URI
	if uri == "" {
		return derp.Internal(location, "Common database must have a URI")
	}

	// RULE: Must have a database name
	if database == "" {
		return derp.Internal(location, "Common database must have a database name")
	}

	// Make a copy of the commonDatabase (pointer) so we can close it after we set up a new one
	commonDatabaseCopy := factory.commonDatabase // nolint:scopeguard

	// Try to connect to the cache database
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))

	if err != nil {
		return derp.Wrap(err, location, "Connecting to common database", uri)
	}

	log.Trace().Msg("Connected to common database")
	factory.commonDatabase = client.Database(database)

	// If there is already a cache connection in place, then close it before we open a new one
	if commonDatabaseCopy != nil {
		if err := commonDatabaseCopy.Client().Disconnect(context.Background()); err != nil {
			derp.Report(derp.Wrap(err, location, "Disconnecting from database"))
		}
	}

	// Index synchronization is the CALLER's job (each mode decides when it is safe to run).
	// NOTE: the old `go derp.Report(queries.SyncSharedIndexes(...))` here was a gotcha --
	// `go f(g())` evaluates g() synchronously, so the "async" sync always blocked, including
	// against unreachable servers at 30s per collection.

	return nil
}

// syncCommonDatabaseIndexes synchronizes the shared indexes on the common database.
// Callers run this only when the connection is (believed) reachable: against a dead
// server it blocks ~30 seconds per collection.
func (factory *factoryCore) syncCommonDatabaseIndexes(connection mapof.String) {
	log.Trace().Str("database", factory.commonDatabase.Name()).Msg("Synchronizing common database indexes")
	derp.Report(queries.SyncSharedIndexes(connection.GetString("connectString"), connection.GetString("database")))
}

// refreshFilesystems mounts the attachment and export directories named in the configuration
func (factory *factoryCore) refreshFilesystems(config config.Config) {

	const location = "server.factoryCore.refreshFilesystems"

	filesystemService := factory.Filesystem()

	if attachmentOriginals, err := filesystemService.GetAfero(config.AttachmentOriginals); err == nil {
		factory.attachmentOriginals = attachmentOriginals
	} else {
		derp.Report(derp.Wrap(err, location, "Getting `attachment original` directory", config))
	}

	if attachmentCache, err := filesystemService.GetAfero(config.AttachmentCache); err == nil {
		factory.attachmentCache = attachmentCache
	} else {
		derp.Report(derp.Wrap(err, location, "Getting `attachment cache` directory", config))
	}

	if exportCache, err := filesystemService.GetAfero(config.ExportCache); err == nil {
		factory.exportCache = exportCache
	} else {
		derp.Report(derp.Wrap(err, location, "Getting `export cache` directory", config))
	}
}

// refreshQueue rebuilds the task queue, optionally backed by mongo storage
func (factory *factoryCore) refreshQueue(withStorage bool) {

	// If there is already a queue in place, then close it before we open a new one
	factory.queue.Stop()

	// Configure queue options, including task consumers
	options := []queue.Option{
		queue.WithConsumers(consumer.New(factory).Run),
		queue.WithRunImmediatePriority(32),
	}

	// RULE: Only the live server may read/write queued tasks in the database.  The setup
	// console runs an in-memory queue so it cannot consume production tasks.
	if withStorage {
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

// refreshDerpPlugins rebuilds the derp error-reporting plugins named in the configuration
func (factory *factoryCore) refreshDerpPlugins(config config.Config) {

	const location = "server.factoryCore.refreshDerpPlugins"

	derp.Plugins.Clear()

	for _, logger := range config.Loggers {

		switch logger.GetString("type") {

		case "console":
			log.Trace().Msg("Adding console logger to DERP...")
			derp.Plugins.Add(derpconsole.New())

		case "mongo":

			// RULE: The mongo logger writes to the common database, which may not be
			// connected yet in setup mode.  Skip (loudly) instead of panicking.
			if factory.commonDatabase == nil {
				log.Warn().Msg("Cannot add mongo logger until the common database is connected")
				continue
			}

			log.Trace().Msg("Adding mongo logger to DERP...")
			derp.Plugins.Add(derpmongo.New(
				factory.commonDatabase.Collection("ErrorLog"),
				logger))

		default:
			log.Error().Str("loc", location).Str("type", logger.GetString("type")).Msg("Unknown logging type")
		}
	}

	// RULE: The application must NEVER run without an error sink.  A config that declares no
	// (valid) loggers — e.g. a hand-written file that omits "loggers" — would send every
	// derp.Report() into a black hole, silently swallowing failures like a domain that cannot
	// bootstrap.  Default to a console reporter so reported errors always reach stdout.
	if len(derp.Plugins) == 0 {
		log.Warn().Str("loc", location).Msg("No loggers configured; defaulting to a console error reporter so failures are visible")
		derp.Plugins.Add(derpconsole.New())
	}
}

// refreshDomains synchronizes the in-memory domain registry with the configuration:
// new domains are created, existing ones are refreshed, and removed ones are deleted.
func (factory *factoryCore) refreshDomains(config config.Config) {

	const location = "server.factoryCore.refreshDomains"

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

			// RULE: A domain that fails to refresh is left OUT of the registry, so every request
			// to it later returns "421 Hostname is invalid" with no other clue.  Log the failure
			// on the always-on zerolog channel (not only derp.Report, which needs a configured
			// sink) and name the unreachable hostname so the root cause is tied to the symptom.
			log.Error().Err(err).Str("loc", location).Str("hostname", domainConfig.Hostname).Msg("Domain failed to load and will be UNREACHABLE (requests to it will return 421)")
			derp.Report(derp.Wrap(err, location, "Refreshing domain", domainConfig.ID))
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
}

// init populates the mode-independent parts of a server factory: the domain
// registry, global services, caches, and function map.
func (factory *factoryCore) init(storage config.Storage, embeddedFiles embed.FS) {

	// It populates in place (rather than returning a value) because the services below capture
	// method values and field pointers bound to this exact address; a copy would dangle them.
	factory.storage = storage
	factory.domains = xsync.NewMap[string, *service.Factory]()
	factory.embeddedFiles = embeddedFiles
	factory.jwtService = service.NewJWT()
	factory.queue = queue.New()

	// Build the in-memory cache
	otterCache, _ := otter.MustBuilder[string, string](1000).
		WithVariableTTL().
		Build()

	factory.funcMap = templates.FuncMap(factory.Icons())

	factory.httpCache = httpcache.NewOtterCache(otterCache, httpcache.WithTTL(1*time.Minute))

	factory.iconService = service.NewIcons()

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
		factory.ClientIP, // resolve client IPs using the configured trusted-proxy strategy
		dome.LogStatusCodes(
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusInternalServerError,
		),
	)

	factory.workingDirectory = mediaserver.NewWorkingDirectory(os.TempDir(), 4*time.Minute, 10000)

	// The core is assembled; each mode's lifecycle takes it from here.
}
