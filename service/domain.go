package service

import (
	"html/template"
	"sync"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/external"
	"github.com/EmissarySocial/emissary/tools/domain"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/maps"
	"github.com/davecgh/go-spew/spew"
)

// Domain service manages all access to the singleton model.Domain in the database
type Domain struct {
	collection      data.Collection
	configuration   config.Domain
	externalService *External
	funcMap         template.FuncMap
	model           model.Domain
	lock            *sync.Mutex
}

// NewDomain returns a fully initialized Domain service
func NewDomain(collection data.Collection, configuration config.Domain, externalService *External, funcMap template.FuncMap) Domain {
	service := Domain{
		externalService: externalService,
		funcMap:         funcMap,
		lock:            &sync.Mutex{},
	}

	service.Refresh(collection, configuration)

	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Domain) Refresh(collection data.Collection, configuration config.Domain) {
	service.collection = collection
	service.configuration = configuration
	service.model = model.NewDomain()
}

// Close stops the subscription service watcher
func (service *Domain) Close() {
}

/*******************************************
 * Common Data Methods
 *******************************************/

// Load retrieves an Domain from the database (or in-memory cache)
func (service *Domain) Load(domain *model.Domain) error {

	// If the value is already cached, then return it
	if !service.model.DomainID.IsZero() {
		*domain = service.model
		return nil
	}

	// Initialize a new object (to avoid NPE errors)
	service.lock.Lock()
	defer service.lock.Unlock()

	service.model = model.NewDomain()

	// If not cached, try to load from database
	err := service.collection.Load(exp.All(), &service.model)

	// If present in database, return success
	if err == nil {
		*domain = service.model
		return nil
	}

	// If not in database, try to create a new record
	if derp.NotFound(err) {

		if err := service.Save(domain, "Create New Domain"); err != nil {
			return derp.Wrap(err, "service.Domain.Load", "Error creating new domain")
		}

		*domain = service.model
		return nil
	}

	// Otherwise, there's some bigger error happening, fail un-gracefully
	return derp.Wrap(err, "service.Domain.Load", "Error loading Domain")
}

// Save updates the value of this domain in the database (and in-memory cache)
func (service *Domain) Save(domain *model.Domain, note string) error {

	// Try to save the value to the database
	if err := service.collection.Save(domain, note); err != nil {
		return derp.Wrap(err, "service.Domain.Save", "Error saving Domain")
	}

	// Update the in-memory cache
	service.model = *domain

	return nil
}

/*******************************************
 * Generic Data Methods
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *Domain) ObjectNew() data.Object {
	result := model.NewDomain()
	return &result
}

func (service *Domain) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return nil, derp.NewBadRequestError("service.Domain.ObjectDelete", "Unsupported")
}

func (service *Domain) ObjectLoad(_ exp.Expression) (data.Object, error) {
	result := model.NewDomain()
	err := service.Load(&result)
	return &result, err
}

func (service *Domain) ObjectSave(object data.Object, note string) error {
	return service.Save(object.(*model.Domain), note)
}

func (service *Domain) ObjectDelete(object data.Object, note string) error {
	return derp.NewBadRequestError("service.Domain.ObjectDelete", "Unsupported")
}

func (service *Domain) Debug() maps.Map {
	return maps.Map{
		"service": "Domain",
	}
}

/*******************************************
 * OAuth Methods
 *******************************************/

// Adapter returns the external Adapter that matches the given providerID
func (service *Domain) Adapter(providerID string) (external.Adapter, bool) {
	return service.externalService.GetAdapter(providerID)
}

// ManualAdapter returns the external.ManualAdapter that matches the given providerID
func (service *Domain) ManualAdapter(providerID string) (external.ManualAdapter, bool) {

	if adapter, ok := service.Adapter(providerID); ok {

		if manualAdapter, ok := adapter.(external.ManualAdapter); ok {
			return manualAdapter, true
		}
	}

	return nil, false
}

// OAuthAdapter returns the external.OAuthAdapter that matches the given providerID
func (service *Domain) OAuthAdapter(providerID string) (external.OAuthAdapter, bool) {

	if adapter, ok := service.Adapter(providerID); ok {

		if oAuthAdapter, ok := adapter.(external.OAuthAdapter); ok {
			return oAuthAdapter, true
		}
	}

	return nil, false
}

// OAuthCodeURL generates a new (unique) OAuth state and AuthCodeURL for the specified provider
func (service *Domain) OAuthCodeURL(providerID string) (string, error) {

	// Get the adapter for this provider
	adapter, ok := service.OAuthAdapter(providerID)

	if !ok {
		return "", derp.NewBadRequestError("service.Domain.OAuthCodeURL", "Unknown OAuth Provider", providerID)
	}

	// Set a new "state" for this provider
	state, err := service.NewOAuthState(providerID)

	if err != nil {
		return "", derp.Wrap(err, "service.Domain.OAuthCodeURL", "Error generating new OAuth state")
	}

	// Generate and return the AuthCodeURL
	config := adapter.OAuthConfig()
	config.RedirectURL = service.OAuthCallbackURL()
	authCodeURL := config.AuthCodeURL(state)

	spew.Dump(config, authCodeURL)

	return authCodeURL, nil
}

// OAuthRed
func (service *Domain) OAuthCallbackURL() string {
	return domain.Protocol(service.configuration.Hostname) + service.configuration.Hostname + "/oauth/callback"
}

// NewOAuthState generates and returns a new OAuth state for the specified provider
func (service *Domain) NewOAuthState(providerID string) (string, error) {

	const location = "service.Domain.NewOAuthState"

	var domain model.Domain

	if err := service.Load(&domain); err != nil {
		return "", derp.Wrap(err, location, "Error loading domain")
	}

	// Try to find a matching client
	client, ok := domain.Clients[providerID]

	if !ok {
		client.ProviderID = providerID
	}

	// Try to generate a new state
	newState, err := random.GenerateString(32)

	if err != nil {
		return "", derp.Wrap(err, location, "Error generating random string")
	}

	// Assign the state to the client and save the domain
	client.Data["state"] = newState
	domain.Clients[providerID] = client

	if err := service.Save(&domain, "New OAuth State"); err != nil {
		return "", derp.Wrap(err, location, "Error saving domain")
	}

	return newState, nil
}

// ReadOAuthState returns the OAuth state for the specified provider WITHOUT changing the current value
func (service *Domain) ReadOAuthState(providerID string) (string, error) {

	const location = "service.Domain.NewOAuthState"

	var domain model.Domain

	if err := service.Load(&domain); err != nil {
		return "", derp.Wrap(err, location, "Error loading domain")
	}

	// Try to find a matching client
	client, ok := domain.Clients[providerID]

	if !ok {
		return "", derp.NewBadRequestError(location, "Unknown OAuth Provider", providerID)
	}

	return client.Data.GetString("state"), nil
}
