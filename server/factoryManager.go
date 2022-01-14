package server

import (
	"strings"
	"sync"

	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/form/vocabulary"
	"github.com/benpate/ghost/config"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// FactoryManager manages all interactions with the FactoryManager collection
type FactoryManager struct {
	factories map[string]*domain.Factory
	mutex     sync.RWMutex
	config    config.Config
}

// NewFactoryManager uses the provided configuration data to generate a new FactoryManager
// if there are any errors connecting to a domain's datasource, NewFactoryManager will derp.Report
// the error, but will continue loading without those domains.
func NewFactoryManager(cfg config.Config) *FactoryManager {

	service := &FactoryManager{
		factories: make(map[string]*domain.Factory, len(cfg.Domains)),
		mutex:     sync.RWMutex{},
		config:    cfg,
	}

	for _, domain := range cfg.Domains {
		if err := service.start(domain); err != nil {
			derp.Report(err)
		}
	}

	if len(service.factories) == 0 {
		panic("no domains configured")
	}

	return service
}

// Add appends a new domain into the domain service IF it does not already exist.  If the domain
// is already in the FactoryManager, then no additional action is taken.
func (service *FactoryManager) start(d config.Domain) error {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	factory, err := domain.NewFactory(d)

	if err != nil {
		return derp.Wrap(err, "ghost.service.FactoryManager.New", "Error creating factory", d)
	}

	// Assign the new factory to the registry
	service.factories[d.Hostname] = factory

	return nil
}

/****************************
 * Domain Methods
 ****************************/

func (service *FactoryManager) ListDomains() []config.Domain {
	return service.config.Domains
}

func (service *FactoryManager) DomainByName(hostname string) (config.Domain, error) {

	hostname = service.NormalizeHostname(hostname)

	for _, domain := range service.config.Domains {

		if domain.Hostname == hostname {
			return domain, nil
		}
	}

	return config.Domain{}, derp.NewNotFoundError("ghost.factoryManager.DomainByName", "Domain not fount", hostname)
}

func (service *FactoryManager) write() error {

	// TODO: this hardcoded reference should be moved into the config file itself
	if err := config.Write(service.config, "./config.json"); err != nil {
		return derp.Wrap(err, "ghost.server.FactoryManager.write", "Error writing configuration")
	}

	return nil

}

func (service *FactoryManager) UpdateDomain(indexString string, domain config.Domain) error {

	if indexString == "new" {
		service.config.Domains = append(service.config.Domains, domain)
		return nil
	}

	index := convert.Int(indexString)

	service.config.Domains[index] = domain

	// TODO: this hardcoded reference should be moved into the config file itself
	if err := service.write(); err != nil {
		return derp.Wrap(err, "ghost.server.FactoryManager.WriteConfig", "Error writing configuration")
	}

	service.start(domain)
	return nil
}

// DomainCount returns the number of domains currently configured by this manager.
func (service *FactoryManager) DomainCount() int {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	return len(service.factories)
}

// DomainByIndex returns a domain from its index in the list
func (service *FactoryManager) DomainByIndex(domainID string) (config.Domain, error) {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	if domainID == "new" {
		return config.NewDomain(), nil
	}

	index := convert.Int(domainID)

	if (index < 0) || (index >= len(service.config.Domains)) {
		return config.Domain{}, derp.New(derp.CodeNotFoundError, "ghost.server.FactoryManager.DomainByIndex", "Index out of bounds", index)
	}

	return service.config.Domains[index], nil
}

func (service *FactoryManager) DeleteDomain(domain config.Domain) error {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	for index, d := range service.config.Domains {
		if d.Hostname == domain.Hostname {
			service.config.Domains = append(service.config.Domains[:index], service.config.Domains[index+1:]...)
		}
	}

	if err := service.write(); err != nil {
		return derp.Wrap(err, "ghost.server.FactoryManager.DeleteDomain", "Error saving configuration")
	}

	delete(service.factories, domain.Hostname)

	return nil
}

/****************************
 * Factory Methods
 ****************************/

// ByContext retrieves a domain using an echo.Context
func (service *FactoryManager) ByContext(ctx echo.Context) (*domain.Factory, error) {

	host := service.NormalizeHostname(ctx.Request().Host)
	return service.ByDomainName(host)
}

// ByDomainName retrieves a domain using a Domain Name
func (service *FactoryManager) ByDomainName(name string) (*domain.Factory, error) {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	if domain, ok := service.factories[name]; ok {
		return domain, nil
	}

	return nil, derp.New(404, "ghost.service.FactoryManager.Get", "Unrecognized FactoryManager Name", name)
}

// NormalizeHostname removes some inconsistencies in host names, including a leading "www", if present
func (service *FactoryManager) NormalizeHostname(hostname string) string {

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

// Steranko implements the steranko.Factory method, used for locating the specific
// steranko instance used by a domain.
func (service *FactoryManager) Steranko(ctx echo.Context) (*steranko.Steranko, error) {

	factory, err := service.ByContext(ctx)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.server.FactoryManager.Steranko", "Unable to locate factory for this domain")
	}

	return factory.Steranko(), nil
}

// FormLibrary returns a reference to the form widget library
func (service *FactoryManager) FormLibrary() form.Library {
	result := form.NewLibrary(nil)
	vocabulary.All(&result)
	return result
}
