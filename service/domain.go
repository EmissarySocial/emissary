package service

import (
	"context"
	"html/template"
	"sync"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/providers"
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
	userService     *User
	providerService *Provider
	funcMap         template.FuncMap
	model           model.Domain
	lock            *sync.Mutex
}

// NewDomain returns a fully initialized Domain service
func NewDomain(collection data.Collection, configuration config.Domain, userService *User, providerService *Provider, funcMap template.FuncMap) Domain {
	service := Domain{
		providerService: providerService,
		userService:     userService,
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

// Provider returns the external Provider that matches the given providerID
func (service *Domain) Provider(providerID string) (providers.Provider, bool) {
	return service.providerService.GetProvider(providerID)
}

// ManualProvider returns the external.ManualProvider that matches the given providerID
func (service *Domain) ManualProvider(providerID string) (providers.ManualProvider, bool) {

	if provider, ok := service.Provider(providerID); ok {

		if manualProvider, ok := provider.(providers.ManualProvider); ok {
			return manualProvider, true
		}
	}

	return nil, false
}

// OAuthProvider returns the external.OAuthProvider that matches the given providerID
func (service *Domain) OAuthProvider(providerID string) (providers.OAuthProvider, bool) {

	if provider, ok := service.Provider(providerID); ok {

		if oAuthProvider, ok := provider.(providers.OAuthProvider); ok {
			return oAuthProvider, true
		}
	}

	return nil, false
}

// OAuthCodeURL generates a new (unique) OAuth state and AuthCodeURL for the specified provider
func (service *Domain) OAuthCodeURL(providerID string) (string, error) {

	// Get the provider for this provider
	provider, ok := service.OAuthProvider(providerID)

	if !ok {
		return "", derp.NewBadRequestError("service.Domain.OAuthCodeURL", "Unknown OAuth Provider", providerID)
	}

	// Set a new "state" for this provider
	client, err := service.NewOAuthClient(providerID)

	if err != nil {
		return "", derp.Wrap(err, "service.Domain.OAuthCodeURL", "Error generating new OAuth client")
	}

	// Generate and return the AuthCodeURL
	config := provider.OAuthConfig()

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

	// Get the provider for this provider
	provider, ok := service.OAuthProvider(providerID)

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
	config := provider.OAuthConfig()

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

// ReadOAuthState returns the OAuth state for the specified provider WITHOUT changing the current value.
// THIS SHOULD NOT BE USED TO ACCESS OAUTH TOKENS because they may be expired.  Use GetOAuthToken for that.
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

	return domain, client, nil
}

// GetAuthToken retrieves the OAuth token for the specified provider.  If the token has expired
// then it is refreshed (and saved) automatically before returning.
func (service *Domain) GetOAuthToken(providerID string) (model.Domain, model.Client, *oauth2.Token, error) {

	// Get the provider for this OAuth provider
	provider, ok := service.OAuthProvider(providerID)

	if !ok {
		return model.Domain{}, model.Client{}, nil, derp.NewBadRequestError("service.Domain.GetOAuthToken", "Unknown OAuth Provider", providerID)
	}

	// Try to load the Domain and Client data
	domain, client, err := service.ReadOAuthClient(providerID)

	if err != nil {
		return model.Domain{}, model.Client{}, nil, derp.Wrap(err, "service.Domain.GetOAuthToken", "Error reading OAuth client")
	}

	// Retrieve the Token from the client
	token := client.Token

	if token == nil {
		return model.Domain{}, model.Client{}, token, derp.NewBadRequestError("service.Domain.GetOAuthToken", "No OAuth token found for provider", providerID)
	}

	// Use TokenSource to update tokens when they expire.
	config := provider.OAuthConfig()
	source := config.TokenSource(context.Background(), token)

	newToken, err := source.Token()

	if err != nil {
		return model.Domain{}, model.Client{}, token, derp.Wrap(err, "service.Domain.GetOAuthToken", "Error refreshing OAuth token")
	}

	// If the token has changed, save it
	if token.AccessToken != newToken.AccessToken {
		client.Token = newToken
		domain.Clients[providerID] = client
		if err := service.Save(&domain, "Refresh OAuth Token"); err != nil {
			return model.Domain{}, model.Client{}, token, derp.Wrap(err, "service.Domain.GetOAuthToken", "Error saving refreshed Token")
		}
	}

	// Success!
	return domain, client, newToken, nil
}
