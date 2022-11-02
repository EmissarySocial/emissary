package service

import (
	"context"
	"html/template"
	"sync"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/external"
	"github.com/EmissarySocial/emissary/tools/domain"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/maps"
	"golang.org/x/oauth2"
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
	client, err := service.NewOAuthClient(providerID)

	if err != nil {
		return "", derp.Wrap(err, "service.Domain.OAuthCodeURL", "Error generating new OAuth client")
	}

	// Generate and return the AuthCodeURL
	config := adapter.OAuthConfig()

	config.RedirectURL = service.OAuthCallbackURL(providerID)
	/*
		codeChallengeBytes := sha256.Sum256([]byte(client.GetString("code_challenge")))
		codeChallenge := oauth2.SetAuthURLParam("code_challenge", random.Base64URLEncode(codeChallengeBytes[:]))
		codeChallengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	*/

	codeChallenge := oauth2.SetAuthURLParam("code_challenge", client.GetString("code_challenge"))
	codeChallengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "plain")
	authCodeURL := config.AuthCodeURL(client.GetString("state"), codeChallenge, codeChallengeMethod)

	return authCodeURL, nil
}

// OAuthExchange trades a temporary OAuth code for a valid OAuth token
func (service *Domain) OAuthExchange(providerID string, state string, code string) error {

	const location = "service.Domain.OAuthExchange"

	// Get the adapter for this provider
	adapter, ok := service.OAuthAdapter(providerID)

	if !ok {
		return derp.NewBadRequestError(location, "Unknown OAuth Provider", providerID)
	}

	// Try to load the domain from the database
	domain := model.NewDomain()
	if err := service.Load(&domain); err != nil {
		return derp.Wrap(err, location, "Error loading domain")
	}

	// The client must already be set up for this exchange to work.
	client, ok := domain.Clients.Get(providerID)

	if !ok {
		return derp.NewBadRequestError(location, "Unknown OAuth Provider", providerID)
	}

	// Validate the state across requests
	if client.Data.GetString("state") != state {
		return derp.NewBadRequestError(location, "Invalid OAuth State", state)
	}

	// Try to generate the OAuth token
	config := adapter.OAuthConfig()

	token, err := config.Exchange(context.Background(), code,
		oauth2.SetAuthURLParam("code_verifier", client.GetString("code_challenge")),
		oauth2.SetAuthURLParam("redirect_uri", service.OAuthCallbackURL(providerID)))

	if err != nil {
		return derp.NewInternalError(location, "Error exchanging OAuth code for token", err.Error())
	}

	// Try to update the client with the new token
	client.Token = token
	client.Data = maps.New()
	client.Active = true
	domain.Clients.Put(client)

	if service.Save(&domain, "OAuth Exchange") != nil {
		return derp.NewInternalError(location, "Error saving domain")
	}

	// Success!
	return nil
}

// OAuthCallbackURL returns the specific callback URL to use for this host and provider.
func (service *Domain) OAuthCallbackURL(providerID string) string {
	return domain.Protocol(service.configuration.Hostname) + service.configuration.Hostname + "/oauth/" + providerID + "/callback"
}

// NewOAuthState generates and returns a new OAuth state for the specified provider
func (service *Domain) NewOAuthClient(providerID string) (model.Client, error) {

	const location = "service.Domain.NewOAuthState"

	domain := model.NewDomain()

	if err := service.Load(&domain); err != nil {
		return model.Client{}, derp.Wrap(err, location, "Error loading domain")
	}

	if domain.Clients == nil {
		domain.Clients = make(set.Map[model.Client])
	}

	// Try to find a matching client
	client, ok := domain.Clients[providerID]

	if !ok {
		client = model.NewClient(providerID)
	}

	// Try to generate a new state
	newState, err := random.GenerateString(32)

	if err != nil {
		return model.Client{}, derp.Wrap(err, location, "Error generating random string")
	}

	codeChallenge, err := random.GenerateString(64)

	if err != nil {
		return model.Client{}, derp.Wrap(err, location, "Error generating random string")
	}

	// Assign the state to the client and put into the domain
	client.Data["state"] = newState
	client.Data["code_challenge"] = codeChallenge
	domain.Clients[providerID] = client

	// Save the domain
	if err := service.Save(&domain, "New OAuth State"); err != nil {
		return model.Client{}, derp.Wrap(err, location, "Error saving domain")
	}

	return client, nil
}

// ReadOAuthState returns the OAuth state for the specified provider WITHOUT changing the current value
func (service *Domain) ReadOAuthClient(providerID string) (model.Domain, model.Client, error) {

	const location = "service.Domain.NewOAuthState"

	var domain model.Domain

	if err := service.Load(&domain); err != nil {
		return model.Domain{}, model.Client{}, derp.Wrap(err, location, "Error loading domain")
	}

	// Try to find a matching client
	client, ok := domain.Clients[providerID]

	if !ok {
		return model.Domain{}, model.Client{}, derp.NewBadRequestError(location, "Unknown OAuth Provider", providerID)
	}

	// TODO: Renew client if the token has expired

	return domain, client, nil
}
