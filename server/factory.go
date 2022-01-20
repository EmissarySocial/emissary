package server

import (
	"fmt"
	"strings"
	"sync"

	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/form/vocabulary"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/config"
	"github.com/whisperverse/whisperverse/domain"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/service"
)

// Factory manages all server-level services, and generates individual
// domain factories for each domain
type Factory struct {
	config config.Config
	mutex  sync.RWMutex

	// Server-level services
	layoutService   service.Layout
	templateService service.Template
	templateChannel chan string

	domains map[string]*domain.Factory
}

// NewFactory uses the provided configuration data to generate a new Factory
// if there are any errors connecting to a domain's datasource, NewFactory will derp.Report
// the error, but will continue loading without those domains.
func NewFactory(cfg config.Config) *Factory {

	fmt.Println("Starting Server...")

	factory := &Factory{
		config:          cfg,
		mutex:           sync.RWMutex{},
		templateChannel: make(chan string),
		domains:         make(map[string]*domain.Factory, len(cfg.Domains)),
	}

	// Global Layout Service
	factory.layoutService = service.NewLayout(
		cfg.TemplateAdapter,
		cfg.TemplatePath,
		render.FuncMap(),
	)

	go factory.layoutService.Watch()

	// Global Template Service
	factory.templateService = service.NewTemplate(
		factory.Layout(),
		render.FuncMap(),
		cfg.TemplateAdapter,
		cfg.TemplatePath,
		factory.templateChannel,
	)

	if err := factory.templateService.Watch(); err != nil {
		derp.Report(err)
		panic(err)
	}

	for _, domain := range cfg.Domains {
		if err := factory.start(domain); err != nil {
			derp.Report(err)
		}
	}

	if len(factory.domains) == 0 {
		panic("no domains configured")
	}

	return factory
}

// Add appends a new domain into the domain service IF it does not already exist.  If the domain
// is already in the Factory, then no additional action is taken.
func (factory *Factory) start(d config.Domain) error {

	result, err := domain.NewFactory(d, &factory.layoutService, &factory.templateService)

	if err != nil {
		return derp.Wrap(err, "whisper.factory.Factory.New", "Error creating factory", d)
	}

	// Assign the new factory to the registry
	factory.domains[d.Hostname] = result

	return nil
}

/****************************
 * Services
 ****************************/

func (factory *Factory) Layout() *service.Layout {
	return &factory.layoutService
}

/****************************
 * Domain Methods
 ****************************/

func (factory *Factory) ListDomains() []config.Domain {
	return factory.config.Domains
}

func (factory *Factory) DomainByName(hostname string) (config.Domain, error) {

	hostname = factory.NormalizeHostname(hostname)

	for _, domain := range factory.config.Domains {

		if domain.Hostname == hostname {
			return domain, nil
		}
	}

	return config.Domain{}, derp.NewNotFoundError("whisper.factoryManager.DomainByName", "Domain not fount", hostname)
}

func (factory *Factory) write() error {

	// TODO: this hardcoded reference should be moved into the config file itself
	if err := config.Write(factory.config, "./config.json"); err != nil {
		return derp.Wrap(err, "whisper.server.Factory.write", "Error writing configuration")
	}

	return nil

}

func (factory *Factory) UpdateDomain(indexString string, domain config.Domain) error {

	if indexString == "new" {
		factory.config.Domains = append(factory.config.Domains, domain)
		return nil
	}

	index := convert.Int(indexString)

	factory.config.Domains[index] = domain

	// TODO: this hardcoded reference should be moved into the config file itself
	if err := factory.write(); err != nil {
		return derp.Wrap(err, "whisper.server.Factory.WriteConfig", "Error writing configuration")
	}

	factory.start(domain)
	return nil
}

// DomainCount returns the number of domains currently configured by this manager.
func (factory *Factory) DomainCount() int {

	factory.mutex.RLock()
	defer factory.mutex.RUnlock()

	return len(factory.domains)
}

// DomainByIndex returns a domain from its index in the list
func (factory *Factory) DomainByIndex(domainID string) (config.Domain, error) {

	factory.mutex.RLock()
	defer factory.mutex.RUnlock()

	if domainID == "new" {
		return config.NewDomain(), nil
	}

	index := convert.Int(domainID)

	if (index < 0) || (index >= len(factory.config.Domains)) {
		return config.Domain{}, derp.New(derp.CodeNotFoundError, "whisper.server.Factory.DomainByIndex", "Index out of bounds", index)
	}

	return factory.config.Domains[index], nil
}

func (factory *Factory) DeleteDomain(domain config.Domain) error {

	factory.mutex.Lock()
	defer factory.mutex.Unlock()

	for index, d := range factory.config.Domains {
		if d.Hostname == domain.Hostname {
			factory.config.Domains = append(factory.config.Domains[:index], factory.config.Domains[index+1:]...)
		}
	}

	if err := factory.write(); err != nil {
		return derp.Wrap(err, "whisper.server.Factory.DeleteDomain", "Error saving configuration")
	}

	delete(factory.domains, domain.Hostname)

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

	return nil, derp.New(404, "whisper.factory.Factory.Get", "Unrecognized Factory Name", name)
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

// Steranko implements the steranko.Factory method, used for locating the specific
// steranko instance used by a domain.
func (factory *Factory) Steranko(ctx echo.Context) (*steranko.Steranko, error) {

	result, err := factory.ByContext(ctx)

	if err != nil {
		return nil, derp.Wrap(err, "whisper.server.Factory.Steranko", "Unable to locate factory for this domain")
	}

	return result.Steranko(), nil
}

// FormLibrary returns a reference to the form widget library
func (factory *Factory) FormLibrary() form.Library {
	result := form.NewLibrary(nil)
	vocabulary.All(&result)
	return result
}
