package service

import (
	"context"
	"html/template"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
)

// Domain service manages all access to the singleton model.Domain in the database
type Domain struct {
	collection      data.Collection
	configuration   config.Domain
	themeService    *Theme
	userService     *User
	providerService *Provider
	funcMap         template.FuncMap
	domain          model.Domain
	ready           bool
}

// NewDomain returns a fully initialized Domain service
func NewDomain() Domain {
	return Domain{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Domain) Refresh(collection data.Collection, configuration config.Domain, themeService *Theme, userService *User, providerService *Provider, funcMap template.FuncMap) {

	service.collection = collection
	service.configuration = configuration
	service.themeService = themeService
	service.userService = userService
	service.providerService = providerService
	service.funcMap = funcMap

	service.domain = model.NewDomain()

	if _, err := service.LoadOrCreateDomain(); err != nil {
		derp.Report(derp.Wrap(err, "service.Domain.Refresh", "Domain Not Ready: Error loading domain record"))
		return
	}

	if err := queries.UpgradeMongoDB(configuration.ConnectString, configuration.DatabaseName, &service.domain); err != nil {
		derp.Report(derp.Wrap(err, "service.Domain.Refresh", "Domain Not Ready: Error upgrading domain record"))
		return
	}

	service.ready = true
}

// Close stops the following service watcher
func (service *Domain) Close() {
}

// Ready returns TRUE if the service is ready to use
func (service *Domain) Ready() bool {
	return service.ready
}

// LoadOrCreate domain guarantees that a domain record exists in the database.
// It returns A COPY of the service domain.
func (service *Domain) LoadOrCreateDomain() (model.Domain, error) {

	// If the domain has already been loaded, then just return it.
	if service.domain.NotEmpty() {
		return service.domain, nil
	}

	// Try to load the domain from the database
	err := service.collection.Load(exp.All(), &service.domain)

	// If loaded the domain successfully, then return
	if err == nil {
		return service.domain, nil
	}

	// If "Not Found" then initialize and return
	if derp.NotFound(err) {

		if err := service.Save(service.domain, "Created Domain Record"); err != nil {
			return service.domain, derp.Wrap(err, "service.Domain.Refresh", "Error creating new domain record")
		}

		return service.domain, nil
	}

	// Ouch.  This is really bad.  Return the error.
	return service.domain, derp.Wrap(err, "service.Domain.Refresh", "Domain Not Ready: Error loading domain record")
}

/******************************************
 * Common Data Methods
 ******************************************/

// Load retrieves an Domain from the database (or in-memory cache)
func (service *Domain) Get() model.Domain {
	return service.domain
}

func (service *Domain) GetPointer() *model.Domain {
	return &service.domain
}

// Save updates the value of this domain in the database (and in-memory cache)
func (service *Domain) Save(domain model.Domain, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(domain); err != nil {
		return derp.Wrap(err, "service.Domain.Save", "Error cleaning Domain", domain)
	}

	// Try to save the value to the database
	if err := service.collection.Save(&domain, note); err != nil {
		return derp.Wrap(err, "service.Domain.Save", "Error saving Domain")
	}

	// Update the in-memory cache
	service.domain = domain

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Domain) ObjectType() string {
	return "Domain"
}

// New returns a fully initialized model.Stream as a data.Object.
func (service *Domain) ObjectNew() data.Object {
	result := model.NewDomain()
	return &result
}

func (service *Domain) ObjectID(object data.Object) primitive.ObjectID {

	if domain, ok := object.(*model.Domain); ok {
		return domain.DomainID
	}

	return primitive.NilObjectID
}

func (service *Domain) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Domain) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return nil, derp.NewBadRequestError("service.Domain.ObjectDelete", "Unsupported")
}

func (service *Domain) ObjectLoad(_ exp.Expression) (data.Object, error) {
	return &service.domain, nil
}

func (service *Domain) ObjectSave(object data.Object, note string) error {
	if domain, ok := object.(*model.Domain); ok {
		return service.Save(*domain, note)
	}

	return derp.NewInternalError("service.Domain.ObjectSave", "Invalid Object Type", object)
}

func (service *Domain) ObjectDelete(object data.Object, note string) error {
	return derp.NewBadRequestError("service.Domain.ObjectDelete", "Unsupported")
}

func (service *Domain) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Domain", "Not Authorized")
}

func (service *Domain) Schema() schema.Schema {
	return schema.New(model.DomainSchema())
}

/******************************************
 * Provider Methods
 ******************************************/

func (service *Domain) Theme() model.Theme {
	return service.themeService.GetTheme(service.domain.ThemeID)
}

/******************************************
 * Provider Methods
 ******************************************/

// HasSignupForm returns TRUE if this domain allows new users to sign up.
func (service *Domain) HasSignupForm() bool {
	return service.domain.HasSignupForm()
}

// ActiveClients returns all active Clients for this domain
func (service *Domain) ActiveClients() []model.Client {

	// List all clients, filtering by "active" ones.
	result := make([]model.Client, 0, len(service.domain.Clients))

	for _, client := range service.domain.Clients {
		if client.Active {
			result = append(result, client)
		}
	}

	// Success
	return result
}

func (service *Domain) Client(providerID string) model.Client {

	if client, ok := service.domain.Clients[providerID]; ok {
		return client
	}

	return model.NewClient(providerID)
}

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

/******************************************
 * OAuth Handshake Methods
 ******************************************/

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

	config.RedirectURL = service.OAuthClientCallbackURL(providerID)
	/* TODO: MEDIUM: add hash value for challenge_method...
	codeChallengeBytes := sha256.Sum256([]byte(client.GetStringOK("code_challenge")))
	codeChallenge := oauth2.SetAuthURLParam("code_challenge", random.Base64URLEncode(codeChallengeBytes[:]))
	codeChallengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	*/

	codeChallenge := oauth2.SetAuthURLParam("code_challenge", client.Data.GetString("code_challenge"))
	codeChallengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "plain")
	authCodeURL := config.AuthCodeURL(client.Data.GetString("state"), codeChallenge, codeChallengeMethod)

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

	// The client must already be set up for this exchange to work.
	client, ok := service.domain.Clients.Get(providerID)

	if !ok {
		return derp.NewBadRequestError(location, "Unknown OAuth Provider", providerID)
	}

	// Validate the state across requests
	if newState, _ := client.Data.GetStringOK("state"); newState != state {
		return derp.NewBadRequestError(location, "Invalid OAuth State", state)
	}

	// Try to generate the OAuth token
	config := provider.OAuthConfig()

	token, err := config.Exchange(context.Background(), code,
		oauth2.SetAuthURLParam("code_verifier", client.Data.GetString("code_challenge")),
		oauth2.SetAuthURLParam("redirect_uri", service.OAuthClientCallbackURL(providerID)))

	if err != nil {
		return derp.NewInternalError(location, "Error exchanging OAuth code for token", err.Error())
	}

	// Try to update the client with the new token
	client.Token = token
	client.Data = mapof.NewAny()
	client.Active = true
	service.domain.SetClient(client)

	if service.Save(service.domain, "OAuth Exchange") != nil {
		return derp.NewInternalError(location, "Error saving domain")
	}

	// Success!
	return nil
}

// OAuthClientCallbackURL returns the specific callback URL to use for this host and provider.
func (service *Domain) OAuthClientCallbackURL(providerID string) string {
	return domain.Protocol(service.configuration.Hostname) + service.configuration.Hostname + "/oauth/clients/" + providerID + "/callback"
}

// NewOAuthState generates and returns a new OAuth state for the specified provider
func (service *Domain) NewOAuthClient(providerID string) (model.Client, error) {

	const location = "service.Domain.NewOAuthState"

	// Find or Create a client for this provider
	client, _ := service.domain.GetClient(providerID)

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
	service.domain.SetClient(client)

	// Save the domain
	if err := service.Save(service.domain, "New OAuth State"); err != nil {
		return model.Client{}, derp.Wrap(err, location, "Error saving domain")
	}

	return client, nil
}

// ReadOAuthState returns the OAuth state for the specified provider WITHOUT changing the current value.
// THIS SHOULD NOT BE USED TO ACCESS OAUTH TOKENS because they may be expired.  Use GetOAuthToken for that.
func (service *Domain) ReadOAuthClient(providerID string) (model.Client, bool) {
	return service.domain.GetClient(providerID)
}

// GetAuthToken retrieves the OAuth token for the specified provider.  If the token has expired
// then it is refreshed (and saved) automatically before returning.
func (service *Domain) GetOAuthToken(providerID string) (model.Client, *oauth2.Token, error) {

	// Get the provider for this OAuth provider
	provider, ok := service.OAuthProvider(providerID)

	if !ok {
		return model.Client{}, nil, derp.NewBadRequestError("service.Domain.GetOAuthToken", "Unknown OAuth Provider", providerID)
	}

	// Try to load the Domain and Client data
	client, ok := service.ReadOAuthClient(providerID)

	if !ok {
		return model.Client{}, nil, derp.NewBadRequestError("service.Domain.GetOAuthToken", "Error reading OAuth client")
	}

	// Retrieve the Token from the client
	token := client.Token

	if token == nil {
		return model.Client{}, token, derp.NewBadRequestError("service.Domain.GetOAuthToken", "No OAuth token found for provider", providerID)
	}

	// Use TokenSource to update tokens when they expire.
	config := provider.OAuthConfig()
	source := config.TokenSource(context.Background(), token)

	newToken, err := source.Token()

	if err != nil {
		return model.Client{}, token, derp.Wrap(err, "service.Domain.GetOAuthToken", "Error refreshing OAuth token")
	}

	// If the token has changed, save it
	if token.AccessToken != newToken.AccessToken {
		client.Token = newToken
		service.domain.SetClient(client)
		if err := service.Save(service.domain, "Refresh OAuth Token"); err != nil {
			return model.Client{}, token, derp.Wrap(err, "service.Domain.GetOAuthToken", "Error saving refreshed Token")
		}
	}

	// Success!
	return client, newToken, nil
}
