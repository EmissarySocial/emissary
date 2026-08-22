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
	"sync"
	"sync/atomic"
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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// factoryCore holds the state and behavior shared by every server mode
// (live, setup, and any future mode). Mode structs embed it and add their
// own lifecycle; see emissary-specs/FACTORY-MODES.md.
type factoryCore struct {
	storage config.Storage

	// Server-level services
	contentService      service.Content
	emailService        service.ServerEmail
	iconService         service.Icons
	jwtService          service.JWT
	registrationService service.Registration
	themeService        service.Theme
	templateService     service.Template
	widgetService       service.Widget

	embeddedFiles    embed.FS
	workingDirectory *mediaserver.WorkingDirectory
	digitalDome      *dome.Dome

	// reloadLock serializes writers of `wiring` against EACH OTHER.  No reader ever takes it,
	// so it can be held across the slow parts of a reload -- opening a mongo client, a
	// five-second ping, a storage write, synchronizing every shared index -- without stalling a
	// single request.
	//
	// RULE: Every writer of `wiring` holds this for the WHOLE decide-then-publish sequence.
	// It is what makes the "have the settings changed?" guards sound: nothing else can rewire
	// the server between the compare and the swap.
	reloadLock sync.Mutex

	// wiring is everything a configuration reload replaces: the configuration itself, the
	// mounted filesystems, the common database, the task queue, and the client-IP strategy.
	// Readers load it with no lock at all and get one internally consistent generation --
	// never the new queue alongside the old database.
	//
	// RULE: NEVER touch this field directly -- go through currentWiring/rewire.
	wiring atomic.Pointer[wiring]

	funcMap   template.FuncMap
	domains   *xsync.Map[string, *service.Factory]
	httpCache httpcache.HTTPCache
}

/******************************************
 * Server Config Methods
 ******************************************/

// Config returns an independent copy of the current configuration for the Factory.
//
// RULE: The copy is DEEP (config.Config.Copy).  Callers -- the setup console's form handlers
// above all -- treat the returned value as scratch space and edit it in place; a shallow copy
// would share its maps with the published wiring, whose configuration is immutable.
func (factory *factoryCore) Config() config.Config {
	return factory.currentWiring().config.Copy()
}

// setConfigLocked replaces the server configuration.  Callers must hold reloadLock.
//
// The value is deep-copied on the way in: the caller usually still holds a reference to it (the
// setup console hands over the very struct its form handler was editing), and published wiring
// must never share map storage with anybody's scratch space.
func (factory *factoryCore) setConfigLocked(value config.Config) {

	copied := value.Copy()

	factory.rewireLocked(func(w *wiring) {
		w.config = copied
	})
}

// AllowPrivateIPs reports whether outbound ActivityPub delivery may connect to
// non-public (private/loopback) addresses. FALSE in production; enabled only for
// local/dev federation between machines on a private network.
func (factory *factoryCore) AllowPrivateIPs() bool {
	return factory.currentWiring().config.AllowPrivateIPs
}

// UpdateConfig replaces and persists the configuration for the Factory
func (factory *factoryCore) UpdateConfig(value config.Config) error {

	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()

	return factory.updateConfigLocked(value)
}

// updateConfigLocked persists the configuration, and publishes it locally only once the write
// succeeded.  Callers must hold reloadLock -- which is safe to hold across the storage write,
// because no reader ever takes it, and correct to hold, because it serializes this save against
// any reload arriving from another node.
//
// RULE: Write first, publish after.  Storage.Write is a compare-and-swap that can return a 409
// when another node changed the configuration underneath this save; a rejected save must leave
// this node running exactly what it ran before.  The STORED version (with its incremented
// revision) is what gets published, so the next save from this node carries the right base.
//
// A conflict is deliberately NOT retried here: this path carries a human's form edit, made
// against the configuration they were looking at.  Re-applying it over someone else's change
// would be the silent overwrite the revision exists to prevent -- the human reloads, sees the
// other change, and decides.
func (factory *factoryCore) updateConfigLocked(value config.Config) error {

	const location = "server.factory.UpdateConfig"

	stored, err := factory.storage.Write(value)

	if err != nil {

		// Surfaced UN-wrapped: the message is written for the setup-console user, and the
		// handler displays exactly what it receives.
		if derp.IsConflict(err) {
			return err
		}

		return derp.Wrap(err, location, "Writing configuration", value)
	}

	factory.setConfigLocked(stored)

	return nil
}

// mutateConfigLocked persists a read-modify-write of the configuration, retrying on revision
// conflicts.  Callers must hold reloadLock.
//
// It exists for the MECHANICAL mutations -- adding and removing domains -- where `fn` states an
// intent ("this domain is in the list") that is safe to re-apply on top of whatever another node
// changed.  Contrast updateConfigLocked, which carries a human's whole-form edit and must NOT be
// replayed over someone else's changes.
//
// RULE: The rebase reads from STORAGE, not from this factory's wiring.  A conflict means the
// wiring is stale by definition -- the winning write's echo may not have arrived yet -- so
// rebasing on the wiring could just conflict forever.
func (factory *factoryCore) mutateConfigLocked(fn func(*config.Config)) error {

	const location = "server.factory.mutateConfigLocked"

	base := factory.currentWiring().config.Copy()

	for attempt := 0; ; attempt++ {

		// Apply the mutation to a scratch copy of the base
		updated := base.Copy()
		fn(&updated)

		stored, err := factory.storage.Write(updated)

		if err == nil {
			factory.setConfigLocked(stored)
			return nil
		}

		// RULE: Only a revision conflict is retryable, and only a few times -- repeated
		// conflicts mean something is writing faster than we can rebase, and a human should
		// look at that rather than a loop hiding it.
		if !derp.IsConflict(err) || attempt >= 2 {
			return derp.Wrap(err, location, "Saving configuration")
		}

		// Rebase on what is actually stored, and re-apply the mutation
		fresh, readErr := factory.storage.Read()

		if readErr != nil {
			return derp.Wrap(readErr, location, "Re-reading configuration after a save conflict")
		}

		base = fresh
	}
}

/******************************************
 * Domain Methods
 ******************************************/

// RangeDomains returns an iterator over every Domain factory in this server
func (factory *factoryCore) RangeDomains() iter.Seq[*service.Factory] {

	return func(yield func(*service.Factory) bool) {

		factory.domains.Range(func(_ string, domain *service.Factory) bool {
			return yield(domain)
		})
	}
}

// ListDomains returns a list of all domains in the Factory
func (factory *factoryCore) ListDomains() []config.Domain {

	// Cloned, not returned directly: published wiring is immutable, and handing out its slice
	// would invite callers to edit it in place.  Built with make() so the result is always a
	// usable (possibly empty) slice, never nil.
	domains := factory.currentWiring().config.Domains

	result := make([]config.Domain, len(domains))
	copy(result, domains)

	return result
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
	if factory.currentWiring().commonDatabase == nil {
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

	// RULE: The whole add runs under reloadLock, so a configuration reload cannot interleave
	// between reading the config and persisting the merged result -- which would silently
	// revert whatever the reload carried.
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()

	// Add the domain to the configuration and persist it, rebasing on conflicts.  (The D6
	// common-database guard runs in PutDomain, before any of this.)  Put is keyed by DomainID,
	// so re-applying it over another node's changes cannot lose anything -- least of all this
	// domain's MasterKey, whose ONLY copy is in `configuration`.
	if err := factory.mutateConfigLocked(func(value *config.Config) {
		value.Domains.Put(configuration)
	}); err != nil {
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

	// Search for the domain in the current configuration
	if domain, ok := factory.currentWiring().config.Domains.Get(domainID); ok {
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

	// RULE: The read-modify-write runs under reloadLock; see putDomain
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()

	// Delete the domain from the configuration and persist it, rebasing on conflicts.  Delete
	// is keyed by DomainID, so re-applying it over another node's changes is always safe.
	if err := factory.mutateConfigLocked(func(value *config.Config) {
		value.Domains.Delete(domainID)
	}); err != nil {
		return derp.Wrap(err, location, "Saving configuration")
	}

	return nil
}

// refreshDomain attempts to refresh an existing domain, or creates a new one if it doesn't exist
func (factory *factoryCore) refreshDomain(domainConfig config.Domain) error {

	const location = "server.factory.refreshDomain"

	// Read ONE generation of wiring, so the filesystems and the database that this domain is
	// built from cannot come from two different configurations.
	current := factory.currentWiring()

	// Try to find the domain
	if domain, exists := factory.domains.Load(domainConfig.Hostname); exists {

		// Even if there's an error "refreshing" the domain, we don't want to delete it
		domain.MarkForDeletion = false

		// Try to refresh the domain
		if err := domain.Refresh(domainConfig, current.attachmentOriginals, current.attachmentCache); err != nil {
			return derp.Wrap(err, location, "Refreshing domain", domainConfig.Hostname)
		}

		return nil
	}

	// RULE: Creating a domain factory requires the common database.  Callers gate on this
	// too (putDomain, the mode lifecycles); this is defense in depth against a nil-pointer
	// panic inside mongodb.NewServer.
	if current.commonDatabase == nil {
		return derp.Internal(location, "Common database must be connected before creating domains")
	}

	// Fall through means that the domain does not exist, so we need to create it.  The common
	// database and queue are not passed: the domain factory reads them through `factory` (its
	// ServerFactory) on every use, so a later config reload can never strand it on a dead handle.
	newDomain, err := service.NewFactory(
		factory,
		domainConfig,
		factory.port(domainConfig),
		&factory.contentService,
		&factory.jwtService,
		&factory.registrationService,
		&factory.templateService,
		&factory.themeService,
		&factory.widgetService,
		current.attachmentOriginals,
		current.attachmentCache,
		current.exportCache,
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

// Content returns the global content service
func (factory *factoryCore) Content() *service.Content {
	return &factory.contentService
}

// Queue returns the global message queue service
func (factory *factoryCore) Queue() *queue.Queue {
	return factory.currentWiring().queue
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

// DigitalDome returns the shared Digital Dome instance, which guards the server against abusive traffic
func (factory *factoryCore) DigitalDome() *dome.Dome {
	return factory.digitalDome
}

// HTTPCache returns the shared HTTP cache used by outbound requests
func (factory *factoryCore) HTTPCache() *httpcache.HTTPCache {
	return &factory.httpCache
}

// CommonDatabase returns a link to the common database server
func (factory *factoryCore) CommonDatabase() *mongo.Database {
	return factory.currentWiring().commonDatabase
}

// Server returns the database server for the named domain
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
	return factory.currentWiring().config.IsReadyForDomains()
}

// calcClientIPStrategy returns the strategy used to identify a request's true client IP, per the configuration
func (factory *factoryCore) calcClientIPStrategy(config config.Config) realclientip.Strategy {

	const location = "server.Factory.ClientIPStrategy"

	var strategy realclientip.Strategy
	var err error

	switch config.ClientIPStrategy {

	// RULE: An empty strategy falls back to REMOTE-ADDR, the documented default.
	// A config file that omits "clientIpStrategy" unmarshals into an empty string,
	// so this MUST be treated as the default rather than an unknown value.
	case "", "REMOTE-ADDR":
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

	strategy := factory.currentWiring().clientIPStrategy

	if strategy == nil {
		derp.Report(derp.Internal("server.Factory.ClientIPStrategy", "Client IP strategy cannot be nil"))
		return ""
	}

	return strategy.ClientIP(request.Header, request.RemoteAddr)
}

// Hostname returns the hostname for the request.
func (factory *factoryCore) Hostname(request *http.Request) string {

	// If the server config includes TrustForwardedHost, then the X-Forwarded-Host header is used.
	if factory.currentWiring().config.TrustForwardedHost {
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

// port returns the ":port" suffix to use in URLs for the provided Domain, which is empty for public hostnames
func (factory *factoryCore) port(domainConfig config.Domain) string {

	// If not localhost, then use standard ports and assume the
	// hosting environment will handle the port forwarding
	if !uri.IsLocalHostname(domainConfig.Hostname) {
		return ""
	}

	// If using localhosts, then return the port number if it's not 80
	switch httpPort := factory.currentWiring().config.HTTPPort; httpPort {
	case 0, 80:
		return ""

	default:
		return ":" + strconv.Itoa(httpPort)
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

// openCommonDatabase validates connection settings and opens a client for them.  It publishes
// NOTHING: the caller decides when (and whether) the new connection becomes the live one, which
// is what lets the setup console verify a server with a Ping before committing to it.
//
// mongo.Connect is lazy -- it never contacts the server -- so this only fails on settings that
// cannot produce a client at all.
func openCommonDatabase(connection mapof.String) (*mongo.Database, error) {

	const location = "server.openCommonDatabase"

	uri := connection.GetString("connectString")
	database := connection.GetString("database")

	// RULE: Must have URI
	if uri == "" {
		return nil, derp.Internal(location, "Common database must have a URI")
	}

	// RULE: Must have a database name
	if database == "" {
		return nil, derp.Internal(location, "Common database must have a database name")
	}

	// Open the connection
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))

	if err != nil {
		return nil, derp.Wrap(err, location, "Connecting to common database", uri)
	}

	return client.Database(database), nil
}

// disconnectCommonDatabase closes a database client that is no longer the live one
func disconnectCommonDatabase(database *mongo.Database) {

	const location = "server.disconnectCommonDatabase"

	if database == nil {
		return
	}

	if err := database.Client().Disconnect(context.Background()); err != nil {
		derp.Report(derp.Wrap(err, location, "Disconnecting from database"))
	}
}

// refreshCommonDatabase updates the connection to the common database, and reports whether the
// live connection actually changed.  It is the ONE connect path for every mode; `verify` is
// where the modes differ:
//
//   - verify TRUE (setup console): the new connection must answer a Ping before it is
//     published, and the shared indexes are synchronized once it does.  A server that does not
//     answer never becomes the connection that requests read; the factory rolls back to "not
//     connected" so domain management stays gated with a clear error (FACTORY-MODES D6) and a
//     later save can retry cleanly.
//
//   - verify FALSE (live server): the connection is published as opened.  The live server
//     verifies by using it -- readConfig exits the process if a session cannot be built -- and
//     index synchronization stays the caller's job, because syncing against an unreachable
//     server blocks ~30s per collection.
//
// RULE: The caller MUST hold reloadLock.  The unchanged-guard below reads the live connection
// and then publishes a new generation based on what it read, so a second reload interleaving
// there would decide on settings that had already been replaced.
func (factory *factoryCore) refreshCommonDatabase(connection mapof.String, verify bool) (bool, error) {

	const location = "server.factory.refreshCommonDatabase"

	uri := connection.GetString("connectString")
	database := connection.GetString("database")

	// RULE: Validate BEFORE the unchanged-guard.  An empty spec must always be an error -- the
	// guard compares against the CURRENT settings, and comparing empty-to-empty would otherwise
	// wave an unconfigured database through as "unchanged".
	if uri == "" {
		return false, derp.Internal(location, "Common database must have a URI")
	}

	if database == "" {
		return false, derp.Internal(location, "Common database must have a database name")
	}

	// RULE: Keep the live connection when the settings are unchanged.  Every config reload runs
	// this method, and most reloads do not touch the database settings.  Reconnecting anyway
	// would disconnect the old client -- stranding every existing domain factory, the queue's
	// storage, and the ActivityStream cache on a dead client ("client is disconnected").
	//
	// RULE: When verification is required, only a VERIFIED connection counts as unchanged.  A
	// client that was opened but never answered a Ping is exactly the case this guard must not
	// skip past.
	current := factory.currentWiring()

	if current.commonDatabase != nil &&
		uri == current.commonDatabaseURI &&
		database == current.commonDatabaseName &&
		(current.commonDatabaseVerified || !verify) {

		log.Trace().Msg("Common database settings unchanged. Keeping current connection.")
		return false, nil
	}

	// Open the new connection BEFORE publishing anything, so a failure leaves the live
	// connection exactly as it was.
	opened, err := openCommonDatabase(connection)

	if err != nil {
		return false, derp.Wrap(err, location, "Opening common database")
	}

	// RULE: Verify BEFORE publishing.  Ping forces real server selection, bounded so an
	// unreachable host fails in seconds with a clear message NOW, instead of failing darkly
	// when the first domain loads.
	if verify {

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := opened.Client().Ping(ctx, readpref.Primary()); err != nil {

			// Roll back to "not connected" so a later save retries.  The configuration now
			// names a database we cannot reach, so continuing to serve from the PREVIOUS one
			// would quietly disagree with what the operator just saved.
			factory.setCommonDatabase(nil, "", "", false)

			disconnectCommonDatabase(opened)
			disconnectCommonDatabase(current.commonDatabase)

			// Any existing domain factories are bound to the previous (now closed) connection.
			// Drop them so lookups fail cleanly instead of surfacing dark mongo errors.
			factory.domains.Clear()

			return true, derp.Wrap(err, location, `Unable to reach the database. Check the connect string — a single-member replica set needs "?directConnection=true".`)
		}
	}

	log.Trace().Msg("Connected to common database")

	// Publish the new generation, then close the client it replaces.  The disconnect happens
	// AFTER the swap and outside every lock: no reader can still reach the old client, and
	// Disconnect is network I/O.
	factory.setCommonDatabase(opened, uri, database, verify)
	disconnectCommonDatabase(current.commonDatabase)

	// Synchronize shared indexes, now that the ping proved the server reachable.  On the
	// unverified path this is the CALLER's job -- see the doc comment.
	// NOTE: the old `go derp.Report(queries.SyncSharedIndexes(...))` here was a gotcha --
	// `go f(g())` evaluates g() synchronously, so the "async" sync always blocked, including
	// against unreachable servers at 30s per collection.
	if verify {
		factory.syncCommonDatabaseIndexes()
	}

	return true, nil
}

// setCommonDatabase publishes a generation carrying this common-database connection and the
// settings behind it.  Callers must hold reloadLock.
func (factory *factoryCore) setCommonDatabase(database *mongo.Database, uri string, name string, verified bool) {

	factory.rewireLocked(func(value *wiring) {
		value.commonDatabase = database
		value.commonDatabaseURI = uri
		value.commonDatabaseName = name
		value.commonDatabaseVerified = verified
	})
}

// syncCommonDatabaseIndexes synchronizes the shared indexes on the common database, through the
// connection the factory already holds.  Callers run this only after a connection is published
// (boot, or an actual settings change) -- index definitions are a function of the BINARY, not
// the configuration, so re-syncing on every reload was pure amplification: every save anywhere
// in the cluster re-ran it on every live node, and the old helper leaked a fresh mongo client
// per call on top.
func (factory *factoryCore) syncCommonDatabaseIndexes() {

	const location = "server.factoryCore.syncCommonDatabaseIndexes"

	commonDatabase := factory.currentWiring().commonDatabase

	// RULE: Nothing to sync without a connection.  Unreachable from the current call sites
	// (both run just after a connection is published), so this is defense in depth.
	if commonDatabase == nil {
		return
	}

	// RULE: Bounded, because this runs under reloadLock.  Against a degraded server every index
	// operation waits out server selection; without a deadline one sick sync would pin every
	// configuration write in the process for minutes.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Trace().Str("loc", location).Str("database", commonDatabase.Name()).Msg("Synchronizing common database indexes")
	queries.SyncSharedIndexes(ctx, commonDatabase)
}

// refreshFilesystems mounts the attachment and export directories named in the configuration.
// Callers must hold reloadLock.
func (factory *factoryCore) refreshFilesystems(config config.Config) {

	const location = "server.factoryCore.refreshFilesystems"

	filesystemService := factory.Filesystem()

	// Mount each directory first, so that a single bad path leaves the others alone.  A mount
	// that fails keeps whatever the previous generation had.
	current := factory.currentWiring()

	attachmentOriginals, err := filesystemService.GetAfero(config.AttachmentOriginals)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Getting `attachment original` directory", config))
		attachmentOriginals = current.attachmentOriginals
	}

	attachmentCache, err := filesystemService.GetAfero(config.AttachmentCache)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Getting `attachment cache` directory", config))
		attachmentCache = current.attachmentCache
	}

	exportCache, err := filesystemService.GetAfero(config.ExportCache)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Getting `export cache` directory", config))
		exportCache = current.exportCache
	}

	// Publish all three together, so no request can see one configuration's originals beside
	// another's cache
	factory.rewireLocked(func(value *wiring) {
		value.attachmentOriginals = attachmentOriginals
		value.attachmentCache = attachmentCache
		value.exportCache = exportCache
	})
}

// refreshQueue rebuilds the task queue, optionally backed by mongo storage.
// Callers must hold reloadLock.
func (factory *factoryCore) refreshQueue(withStorage bool) {

	// RULE: Keep the running queue when nothing it depends on has changed.  Rebuilding stops the
	// old queue, and every domain service that captured it -- plus any task already handed to it
	// -- dies silently ("Turbine Queue: stopped").  The queue depends on its storage mode and,
	// when storage is on, on the common database connection: queueDatabase is compared by pointer
	// identity, which changes exactly when refreshCommonDatabase swaps the connection.  (Ordering:
	// readConfig refreshes the common database BEFORE the queue, so this comparison always sees
	// the current connection.)
	current := factory.currentWiring()

	if current.queueReady && withStorage == current.queueWithStorage {
		if !withStorage || current.commonDatabase == current.queueDatabase {
			log.Trace().Msg("Queue inputs unchanged. Keeping current queue.")
			return
		}
	}

	// If there is already a queue in place, then close it before we open a new one.  Each queue is
	// stopped AT MOST ONCE (a second Stop would panic on its closed `done` channel): the guard
	// above returns early unless we are about to replace it, and the replacement drops the only
	// long-lived reference.
	current.queue.Stop()

	// Configure queue options, including task consumers
	options := []queue.Option{
		queue.WithConsumers(consumer.New(factory).Run),
		queue.WithRunImmediatePriority(32),
	}

	// RULE: Only the live server may read/write queued tasks in the database.  The setup
	// console runs an in-memory queue so it cannot consume production tasks.
	if withStorage {
		mongoStorage := queue_mongo.New(current.commonDatabase, 16, 8)

		// Apply the storage to the queue
		options = append(options,
			queue.WithStorage(mongoStorage),
			queue.WithPollStorage(true),
		)
	}

	// Create a new queue object with consumers, storage, and polling, and publish it alongside
	// the inputs that produced it, so the guard above can detect "unchanged"
	newQueue := queue.New(options...)

	factory.rewireLocked(func(value *wiring) {
		value.queue = newQueue
		value.queueReady = true
		value.queueWithStorage = withStorage
		value.queueDatabase = current.commonDatabase
	})
}

// refreshDerpPlugins rebuilds the derp error-reporting plugins named in the configuration
func (factory *factoryCore) refreshDerpPlugins(config config.Config) {

	const location = "server.factoryCore.refreshDerpPlugins"

	commonDatabase := factory.currentWiring().commonDatabase

	// Build the new reporter list LOCALLY, and publish it with one atomic swap at the end.
	// The old Clear-then-Add sequence mutated the global mid-reload: a data race against every
	// concurrent derp.Report in the process, and a brief window with no reporters at all -- in
	// which the errors most likely to fire are reload errors, the ones we most need to keep.
	reporters := make([]derp.Reporter, 0, len(config.Loggers))

	for _, logger := range config.Loggers {

		switch logger.GetString("type") {

		case "console":
			log.Trace().Msg("Adding console logger to DERP...")
			reporters = append(reporters, derpconsole.New())

		case "mongo":

			// RULE: The mongo logger writes to the common database, which may not be
			// connected yet in setup mode.  Skip (loudly) instead of panicking.
			if commonDatabase == nil {
				log.Warn().Msg("Cannot add mongo logger until the common database is connected")
				continue
			}

			log.Trace().Msg("Adding mongo logger to DERP...")
			reporters = append(reporters, derpmongo.New(
				commonDatabase.Collection("ErrorLog"),
				logger))

		default:
			log.Error().Str("loc", location).Str("type", logger.GetString("type")).Msg("Unknown logging type")
		}
	}

	// RULE: The application must NEVER run without an error sink.  A config that declares no
	// (valid) loggers — e.g. a hand-written file that omits "loggers" — would send every
	// derp.Report() into a black hole, silently swallowing failures like a domain that cannot
	// bootstrap.  Default to a console reporter so reported errors always reach stdout.
	if len(reporters) == 0 {
		log.Warn().Str("loc", location).Msg("No loggers configured; defaulting to a console error reporter so failures are visible")
		reporters = append(reporters, derpconsole.New())
	}

	// One swap: no moment ever exists with a partial or empty reporter list
	derp.SetPlugins(reporters...)
}

// refreshDomains synchronizes the in-memory domain registry with the configuration:
// new domains are created, existing ones are refreshed, and removed ones are deleted.
func (factory *factoryCore) refreshDomains(config config.Config) {

	const location = "server.factoryCore.refreshDomains"

	// First, mark ALL for deletion
	factory.domains.Range(func(_ string, domain *service.Factory) bool {
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

	// Install an inert placeholder queue, so that a task published before the first config
	// reload has somewhere to go instead of a nil pointer.  refreshQueue replaces it with a
	// real, consumer-bearing queue (queueReady stays FALSE until then).
	factory.rewire(func(value *wiring) {
		value.queue = queue.New()
	})

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
