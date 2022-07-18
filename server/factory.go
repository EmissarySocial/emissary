package server

import (
	"fmt"
	"strings"
	"sync"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/form/vocabulary"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
	"github.com/spf13/afero"
)

// Factory manages all server-level services, and generates individual
// domain factories for each domain
type Factory struct {
	storage config.Storage
	config  config.Config
	mutex   sync.RWMutex

	// Server-level services
	layoutService   service.Layout
	templateService service.Template
	templateChannel chan string

	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs

	domains set.Map[string, *domain.Factory]
}

// NewFactory uses the provided configuration data to generate a new Factory
// if there are any errors connecting to a domain's datasource, NewFactory will derp.Report
// the error, but will continue loading without those domains.
func NewFactory(storage config.Storage) *Factory {

	fmt.Println("Starting Emissary Server...")

	factory := Factory{
		storage:         storage,
		mutex:           sync.RWMutex{},
		templateChannel: make(chan string),
		domains:         make(map[string]*domain.Factory, 0),
	}

	go factory.start()
	return &factory
}

func (factory *Factory) start() {

	// Read configuration files from the channel
	for config := range factory.storage.Subscribe() {

		spew.Dump(factory.config)

		if attachmentOriginals, err := config.AttachmentOriginals.GetFilesystem(); err == nil {
			factory.attachmentOriginals = attachmentOriginals
		} else {
			derp.Report(derp.Wrap(err, "server.Factory.start", "Error getting attachment original directory", config.AttachmentOriginals))
		}

		if attachmentCache, err := config.AttachmentCache.GetFilesystem(); err == nil {
			factory.attachmentCache = attachmentCache
		} else {
			derp.Report(derp.Wrap(err, "server.Factory.start", "Error getting attachment cache directory", config.AttachmentCache))
		}

		factory.config = config

		// Global Layout Service
		factory.layoutService = service.NewLayout(
			config.Layouts,
			render.FuncMap(),
		)

		go factory.layoutService.Watch()

		// Global Template Service
		factory.templateService = service.NewTemplate(
			factory.Layout(),
			render.FuncMap(),
			config.Templates,
			factory.templateChannel,
		)

		if err := factory.templateService.Watch(); err != nil {
			derp.Report(err)
			panic(err)
		}

		// Insert/Update a factory for each domain in the configuration
		for _, cfg := range config.Domains {

			// Try to find the domain
			d, err := factory.domains.Get(cfg.Hostname)

			// If the domain already exists, then update configuration info.
			if err == nil {
				d.UpdateConfig(cfg, &factory.layoutService, &factory.templateService, factory.attachmentOriginals, factory.attachmentCache)
				continue
			}

			// Fall through means that the domain does not exist, so we need to create it
			d, err = domain.NewFactory(cfg, &factory.layoutService, &factory.templateService, factory.attachmentOriginals, factory.attachmentCache)

			if err != nil {
				derp.Report(derp.Wrap(err, "server.Factory.start", "Unable to start domain", cfg))
				continue
			}

			factory.domains.Put(d)
		}

		// Unceremoniously remove domains that are no longer in the configuration
		for domainID := range factory.domains {
			if _, err := factory.domains.Get(domainID); err != nil {
				factory.domains.Delete(domainID)
			}
		}
	}
}

/****************************
 * Server Config Methods
 ****************************/

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
func (factory *Factory) PutDomain(domain config.Domain) error {

	factory.mutex.Lock()
	defer factory.mutex.Unlock()

	// Add the domain to the collection
	factory.config.Domains.Put(domain)

	// Try to write the configuration to the storage service
	if err := factory.storage.Write(factory.config); err != nil {
		return derp.Wrap(err, "server.Factory.WriteConfig", "Error writing configuration")
	}

	// The storage service will trigger a new configuration via the Subscrbe() channel

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
	if domain, err := factory.config.Domains.Get(domainID); err == nil {
		return domain, nil
	}

	// Not found, so return an error
	return config.Domain{}, derp.NewNotFoundError("server.Factory.DomainByIndex", "DomainID not found", domainID)
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
 * Factory Methods
 ****************************/

// ByContext retrieves a domain using an echo.Context
func (factory *Factory) ByContext(ctx echo.Context) (*domain.Factory, error) {

	host := factory.NormalizeHostname(ctx.Request().Host)
	return factory.ByDomainName(host)
}

// ByDomainName retrieves a domain using a Domain Name
func (factory *Factory) ByDomainName(name string) (*domain.Factory, error) {

	factory.mutex.RLock()
	defer factory.mutex.RUnlock()

	if domain, ok := factory.domains[name]; ok {
		return domain, nil
	}

	return nil, derp.New(404, "factory.Factory.Get", "Unrecognized Factory Name", name)
}

// NormalizeHostname removes some inconsistencies in host names, including a leading "www", if present
func (factory *Factory) NormalizeHostname(hostname string) string {

	hostname = strings.ToLower(hostname)

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

// Layout returns the global layout service
func (factory *Factory) Layout() *service.Layout {
	return &factory.layoutService
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

// FormLibrary returns a reference to the form widget library
func (factory *Factory) FormLibrary() form.Library {
	result := form.NewLibrary(nil)
	vocabulary.All(&result)
	return result
}

// StaticPath returns the configured path to the "static"
// files for this website.
func (factory *Factory) StaticPath() string {
	return factory.config.Static.Location
}
