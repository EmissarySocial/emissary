package server

import (
	"context"
	"embed"
	"html/template"
	"strings"
	"sync"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/EmissarySocial/emissary/tools/ascacherules"
	"github.com/EmissarySocial/emissary/tools/ascontextmaker"
	"github.com/EmissarySocial/emissary/tools/ashash"
	"github.com/EmissarySocial/emissary/tools/asnormalizer"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/queue"
	"github.com/benpate/icon"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/sherlock"
	"github.com/benpate/steranko"
	"github.com/davidscottmills/goeditorjs"
	"github.com/labstack/echo/v4"
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

	// Server-level services
	themeService    service.Theme
	templateService service.Template
	widgetService   service.Widget
	contentService  service.Content
	providerService service.Provider
	emailService    service.ServerEmail
	taskQueue       queue.Queue
	activityService service.ActivityStream
	embeddedFiles   embed.FS

	activityStreamCache *mongo.Client
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs

	domains   map[string]*domain.Factory
	refreshed chan bool
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
		taskQueue:     queue.NewSimpleQueue(128, 16),
		refreshed:     make(chan bool, 1),
	}

	// Global Theme service
	factory.themeService = *service.NewTheme(
		factory.Template(),
		factory.Content(),
		factory.FuncMap(),
	)

	// Global Widget Service
	factory.widgetService = *service.NewWidget(
		factory.FuncMap(),
	)

	// Global Template Service
	factory.templateService = *service.NewTemplate(
		factory.Filesystem(),
		factory.Theme(),
		factory.Widget(),
		factory.FuncMap(),
		sliceof.NewObject[mapof.String](),
	)

	factory.contentService = service.NewContent(factory.EditorJS())
	factory.providerService = service.NewProvider(factory.config.Providers)

	factory.emailService = service.NewServerEmail(
		factory.Filesystem(),
		factory.FuncMap(),
		sliceof.NewObject[mapof.String](),
	)

	factory.activityService = service.NewActivityStream()

	go factory.start()

	return &factory
}

func (factory *Factory) start() {

	log.Debug().Msg("Factory: waiting for configuration file...")

	filesystemService := factory.Filesystem()

	// Read configuration files from the channel
	for config := range factory.storage.Subscribe() {

		log.Debug().Msg("Factory: received new configuration...")

		if attachmentOriginals, err := filesystemService.GetAfero(config.AttachmentOriginals); err == nil {
			factory.attachmentOriginals = attachmentOriginals
		} else {
			derp.Report(derp.Wrap(err, "server.Factory.start", "Error getting attachment original directory", config.AttachmentOriginals))
		}

		if attachmentCache, err := filesystemService.GetAfero(config.AttachmentCache); err == nil {
			factory.attachmentCache = attachmentCache
		} else {
			derp.Report(derp.Wrap(err, "server.Factory.start", "Error getting attachment cache directory", config.AttachmentCache))
		}

		factory.config = config

		// Mark all domains for deletion (then unmark them later)
		for index := range factory.domains {
			factory.domains[index].MarkForDeletion = true
		}

		// Refresh cached values in global services
		factory.templateService.Refresh(config.Templates)
		factory.emailService.Refresh(config.Emails)
		factory.providerService.Refresh(config.Providers)
		factory.RefreshActivityService(config.ActivityPubCache)

		// Insert/Update a factory for each domain in the configuration
		for _, domainConfig := range config.Domains {

			factory.mutex.Lock()
			if err := factory.refreshDomain(config, domainConfig); err != nil {
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

		factory.refreshed <- true
	}
}

// Refreshed returns the channel that is notified whenever the configuration is refreshed
func (factory *Factory) Refreshed() <-chan bool {
	return factory.refreshed
}

// refreshDomain attempts to refresh an existing domain, or creates a new one if it doesn't exist
// CALLS TO THIS MUST BE LOCKED
func (factory *Factory) refreshDomain(config config.Config, domainConfig config.Domain) error {

	// Try to find the domain
	if existing := factory.domains[domainConfig.Hostname]; existing != nil {

		// Even if there's an error "refreshing" the domain, we don't want to delete it
		existing.MarkForDeletion = false

		// Try to refresh the domain
		if err := existing.Refresh(domainConfig, config.Providers, factory.attachmentOriginals, factory.attachmentCache); err != nil {
			return derp.Wrap(err, "server.Factory.start", "Error refreshing domain", domainConfig.Hostname)
		}

		return nil
	}

	// Fall through means that the domain does not exist, so we need to create it
	newDomain, err := domain.NewFactory(
		domainConfig,
		config.Providers,
		&factory.activityService,
		&factory.emailService,
		&factory.themeService,
		&factory.templateService,
		&factory.widgetService,
		&factory.contentService,
		&factory.providerService,
		factory.taskQueue,
		factory.attachmentOriginals,
		factory.attachmentCache,
	)

	if err != nil {
		return derp.Wrap(err, "server.Factory.start", "Unable to start domain", domainConfig)
	}

	// If there are no errors, then add the domain to the list.
	factory.domains[newDomain.Hostname()] = newDomain

	return nil
}

/****************************
 * Server Config Methods
 ****************************/

func (factory *Factory) Version() string {
	return "0.1.0"
}

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
		return derp.Wrap(err, "server.Factory.UpdateConfig", "Error writing configuration", value)
	}

	return nil
}

/****************************
 * Domain Methods
 ****************************/

// ListDomains returns a list of all domains in the Factory
func (factory *Factory) ListDomains() []config.Domain {
	return factory.config.Domains
}

// PutDomain adds a domain to the Factory
func (factory *Factory) PutDomain(configuration config.Domain) error {

	// Save the domain info ant write a new configuration to the storage service
	if err := factory.putDomain(configuration); err != nil {
		return derp.Wrap(err, "server.Factory.PutDomain", "Error adding domain", configuration)
	}

	// The storage service will trigger a new configuration via the Subscrbe() channel,
	// But we still want to call the owner update manually.

	domainFactory, err := factory.ByDomainName(configuration.Hostname)

	if err != nil {
		return derp.Wrap(err, "server.Factory.PutDomain", "Error getting domain factory", configuration.Hostname)
	}

	userService := domainFactory.User()
	if err := userService.SetOwner(configuration.Owner); err != nil {
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
	if err := factory.refreshDomain(factory.config, configuration); err != nil {
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
	return config.NewDomain(), derp.NewNotFoundError("server.Factory.DomainByID", "DomainID not found", domainID)
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

/****************************
 * OAuth Connection Methods
 ****************************/

// PutConnection adds a domain to the Factory
func (factory *Factory) PutProvider(oauthClient config.Provider) error {

	factory.mutex.Lock()
	defer factory.mutex.Unlock()

	// Add the domain to the collection
	factory.config.Providers.Put(oauthClient)

	// Try to write the configuration to the storage service
	if err := factory.storage.Write(factory.config); err != nil {
		return derp.Wrap(err, "server.Factory.WriteConfig", "Error writing configuration")
	}

	// The storage service will trigger a new configuration via the Subscrbe() channel

	return nil
}

// DeleteConnection removes a domain from the Factory
func (factory *Factory) DeleteProvider(providerID string) error {

	factory.mutex.Lock()
	defer factory.mutex.Unlock()

	// Delete the connection from the collection
	factory.config.Providers.Delete(providerID)

	// Write changes to the storage engine.
	if err := factory.storage.Write(factory.config); err != nil {
		return derp.Wrap(err, "server.Factory.DeleteDomain", "Error saving configuration")
	}

	return nil
}

/****************************
 * Factory Methods
 ****************************/

// ByContext retrieves a domain using an echo.Context
func (factory *Factory) ByContext(ctx echo.Context) (*domain.Factory, error) {

	host := factory.NormalizeHostname(ctx.Request().Host)
	result, err := factory.ByDomainName(host)

	if err != nil {
		return nil, derp.Wrap(err, "server.Factory.ByContext", "Error finding domain", host)
	}

	return result, nil
}

// ByDomainID retrieves a domain using a DomainID
func (factory *Factory) ByDomainID(domainID string) (config.Domain, *domain.Factory, error) {

	// Look up the domain name for this domainID
	domainConfig, err := factory.DomainByID(domainID)

	if err != nil {
		return config.Domain{}, nil, derp.Wrap(err, "server.Factory.ByDomainID", "Error finding domain configuration", domainID)
	}

	// Return the domain
	result, err := factory.ByDomainName(domainConfig.Hostname)

	if err != nil {
		return config.Domain{}, nil, derp.Wrap(err, "server.Factory.ByDomainID", "Error finding domain", domainConfig.Hostname)
	}

	return domainConfig, result, nil
}

// ByDomainName retrieves a domain using a Domain Name
func (factory *Factory) ByDomainName(name string) (*domain.Factory, error) {

	factory.mutex.RLock()
	defer factory.mutex.RUnlock()

	if domain, ok := factory.domains[name]; ok {
		return domain, nil
	}

	return nil, derp.NewNotFoundError("server.Factory.ByDomainName", "Unrecognized domain name", name)
}

// NormalizeHostname removes some inconsistencies in host names, including a leading "www", if present
func (factory *Factory) NormalizeHostname(hostname string) string {

	hostname = strings.ToLower(hostname)
	hostname = list.Head(hostname, ':')

	if dotIndex := strings.Index(hostname, "."); dotIndex > 0 {

		if subdomain := hostname[0 : dotIndex-1]; subdomain == "www" {
			return hostname[dotIndex+1:]
		}
	}

	return hostname
}

/****************************
 * Other Global Services
 ****************************/

// Contet returns the global content service
func (factory *Factory) Content() *service.Content {
	return &factory.contentService
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
	return render.FuncMap(factory.Icons())
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

// Steranko implements the steranko.Factory method, used for locating the specific
// steranko instance used by a domain.
func (factory *Factory) Steranko(ctx echo.Context) (*steranko.Steranko, error) {

	result, err := factory.ByContext(ctx)

	if err != nil {
		return nil, derp.Wrap(err, "server.Factory.Steranko", "Unable to locate factory for domain", ctx.Request().Host)
	}

	return result.Steranko(), nil
}

func (factory *Factory) RefreshActivityService(connection mapof.String) {

	// If there is already a cache connection in place,
	// then close it before we open a new one
	if factory.activityStreamCache != nil {
		go func(client *mongo.Client) {
			if err := client.Disconnect(context.Background()); err != nil {
				derp.Report(derp.Wrap(err, "server.Factory.RefreshActivityService", "Unable to disconnect from database"))
			}
		}(factory.activityStreamCache)
	}

	// Collect arguments from the connection config
	uri := connection.GetString("connectString")
	database := connection.GetString("database")

	// ActivityStream cache is not configured.
	if uri == "" || database == "" {
		return
	}

	// Try to connect to the cache database
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))

	if err != nil {
		derp.Report(derp.Wrap(err, "server.Factory.RefreshActivityService", "Unable to connect to database", uri))
		return
	}

	collection := client.Database(database).Collection("Document")

	// Build a new client stack
	sherlockClient := sherlock.NewClient(
		sherlock.WithUserAgent("Emissary Social: https://emissary.social"),
	)

	normalizerClient := asnormalizer.New(sherlockClient)       // enforce opinionated data formats
	contextMakerClient := ascontextmaker.New(normalizerClient) // compute document context (if missing)
	cacheRulesClient := ascacherules.New(contextMakerClient)   // apply custom caching rules to documents

	cacheClient := ascache.New(cacheRulesClient, collection, ascache.WithIgnoreHeaders()) // cache data in MongoDB
	hashClient := ashash.New(cacheClient)                                                 // Traverse hash values within documents

	factory.activityService.Refresh(hashClient, cacheClient, mongodb.NewCollection(collection))

	// This is breaking somehow.  Test thoroughly before re-enabling.
	// writableCache := ascache.New(contextMakerClient, collection, ascache.WithWriteOnly())
	// crawlerClient := ascrawler.New(writableCache, ascrawler.WithMaxDepth(4))
	// readOnlyCache := ascache.New(crawlerClient, collection, ascache.WithReadOnly())
	// factory.activityService.Refresh(readOnlyCache, mongodb.NewCollection(collection))
}
