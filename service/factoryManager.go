package service

import (
	"strings"
	"sync"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/config"
	"github.com/labstack/echo/v4"
)

// FactoryManager manages all interactions with the FactoryManager collection
type FactoryManager struct {
	factories map[string]*Factory
	mutex     sync.RWMutex
}

// NewFactoryManager uses the provided configuration data to generate a new FactoryManager
// if there are any errors connecting to a domain's datasource, NewFactoryManager will derp.Report
// the error, but will continue loading without those domains.
func NewFactoryManager(c config.Global) *FactoryManager {

	service := &FactoryManager{
		factories: make(map[string]*Factory, len(c.Domains)),
		mutex:     sync.RWMutex{},
	}

	for _, domain := range c.Domains {
		if err := service.Add(domain); err != nil {
			derp.Report(err)
		}
	}

	return service
}

// Add appends a new domain into the domain service IF it does not already exist.  If the domain
// is already in the FactoryManager, then no additional action is taken.
func (service *FactoryManager) Add(domain config.Domain) *derp.Error {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	// If this factory DOES NOT EXIST in the registry...
	if _, ok := service.factories[domain.Hostname]; !ok {

		factory, err := NewFactory(domain)

		if err != nil {
			return derp.Wrap(err, "ghost.service.FactoryManager.New", "Error creating factory", domain)
		}

		// Assign the new factory to the registry
		service.factories[domain.Hostname] = factory
	}

	return nil
}

// DomainCount returns the number of domains currently configured by this manager.
func (service *FactoryManager) DomainCount() int {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	return len(service.factories)
}

// ByContext retrieves a domain using an echo.Context
func (service *FactoryManager) ByContext(ctx echo.Context) (*Factory, *derp.Error) {

	host := service.NormalizeHostname(ctx.Request().Host)
	return service.ByDomainName(host)
}

// ByDomainName retrieves a domain using a Domain Name
func (service *FactoryManager) ByDomainName(name string) (*Factory, *derp.Error) {

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
