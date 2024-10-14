package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"html/template"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
)

// Domain service manages all access to the singleton model.Domain in the database
type Domain struct {
	collection          data.Collection
	configuration       config.Domain
	connectionService   *Connection
	providerService     *Provider
	registrationService *Registration
	themeService        *Theme
	userService         *User
	funcMap             template.FuncMap
	domain              model.Domain
	hostname            string // domain-only name (no protocol)
	ready               bool
}

// NewDomain returns a fully initialized Domain service
func NewDomain() Domain {
	return Domain{
		domain: model.NewDomain(),
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Domain) Refresh(collection data.Collection, configuration config.Domain, connectionService *Connection, providerService *Provider, registrationService *Registration, themeService *Theme, userService *User, funcMap template.FuncMap, hostname string) {

	service.collection = collection
	service.configuration = configuration
	service.connectionService = connectionService
	service.providerService = providerService
	service.registrationService = registrationService
	service.themeService = themeService
	service.userService = userService
	service.funcMap = funcMap
	service.hostname = hostname

	service.ready = true
}

// Init domain guarantees that a domain record exists in the database.
// It returns A COPY of the service domain.
func (service *Domain) Start() error {

	const location = "service.Domain.Start"

	// Try to load the domain model into memory
	_, err := service.LoadDomain()

	// In this process, some errors (like 404's) are okay,
	// so let's look at THIS error a little more closely.
	if err != nil {

		// If it's a "real" error, then we can't continue.
		if !derp.NotFound(err) {
			return derp.Wrap(err, location, "Error loading domain record")
		}

		// If "Not Found", then this is the first run.  Create a new domain record.
		service.domain.Label = service.configuration.Label

		if err := service.Save(service.domain, "Created Domain Record"); err != nil {
			return derp.Wrap(err, location, "Error creating new domain record")
		}

		// If this is a localhost server with "createOwner" set, then create a new owner
		if service.configuration.CreateOwner && service.IsLocalhost() {

			log.Trace().Msg("Creating admin user for local host")

			admin := model.NewUser()
			admin.DisplayName = "Admin"
			admin.Username = "admin"
			admin.EmailAddress = "admin@localhost"
			admin.SetPassword("admin")
			admin.IsOwner = true
			admin.IsPublic = true

			if err := service.userService.Save(&admin, "Create admin user for local host"); err != nil {
				return derp.Wrap(err, "service.Domain.Save", "Error creating admin user for local host")
			}

			log.Trace().Msg("Added admin user for local host")
		}
	}

	// Once we have the domain loaded, try to upgrade the database
	if err := queries.UpgradeMongoDB(service.configuration.ConnectString, service.configuration.DatabaseName, &service.domain); err != nil {
		return derp.Wrap(err, location, "Domain Not Ready: Error upgrading domain record")
	}

	return nil
}

// Close stops the following service watcher
func (service *Domain) Close() {
}

// Ready returns TRUE if the service is ready to use
func (service *Domain) Ready() bool {
	return service.ready
}

func (service *Domain) LoadDomain() (model.Domain, error) {

	// If the domain has already been loaded, then just return it.
	if service.domain.NotEmpty() {
		return service.domain, nil
	}

	// Try to load the domain from the database
	if err := service.collection.Load(exp.All(), &service.domain); err != nil {
		return service.domain, derp.Wrap(err, "service.Domain.LoadDomain", "Domain Not Ready: Error loading domain record")
	}

	// Attach the hostname to the domain
	// (in the future, this should probably be kept in the DB)
	service.domain.Hostname = service.hostname

	// Success.
	return service.domain, nil
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

	// Validate the value using the default domain schema
	if err := model.DomainSchema().Validate(&domain); err != nil {
		return derp.Wrap(err, "service.Domain.Save", "Error validating Domain with standard Domain schema", domain)
	}

	// Validate the value using the custom schema for this domain
	if err := service.Schema().Validate(&domain); err != nil {
		return derp.Wrap(err, "service.Domain.Save", "Error validating Domain with custom schema from Theme", domain)
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

// Count returns the number of records that match the provided criteria
func (service *Domain) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

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

// HasRegistrationForm returns TRUE if this domain allows new users to sign up.
func (service *Domain) HasRegistrationForm() bool {
	return service.domain.HasRegistrationForm()
}

func (service *Domain) LoadRegistration() model.Registration {

	if registrationID := service.domain.RegistrationID; registrationID != "" {
		if registration, err := service.registrationService.Load(registrationID); err == nil {
			return registration
		}
	}

	return model.NewRegistration("", nil)
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

// IsLocalhost returns TRUE if the current domain is a local domain
// (localhost, 127.0.0.1, *.local, etc.)
func (service *Domain) IsLocalhost() bool {
	return domain.IsLocalhost(service.hostname)
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
	connection, err := service.NewOAuthClient(providerID)

	if err != nil {
		return "", derp.Wrap(err, "service.Domain.OAuthCodeURL", "Error generating new OAuth connection")
	}

	// Generate and return the AuthCodeURL
	config := provider.OAuthConfig()

	config.RedirectURL = service.OAuthClientCallbackURL(providerID)
	/* TODO: MEDIUM: add hash value for challenge_method...
	codeChallengeBytes := sha256.Sum256([]byte(connection.GetStringOK("code_challenge")))
	codeChallenge := oauth2.SetAuthURLParam("code_challenge", random.Base64URLEncode(codeChallengeBytes[:]))
	codeChallengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	*/

	codeChallenge := oauth2.SetAuthURLParam("code_challenge", connection.Data.GetString("code_challenge"))
	codeChallengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "plain")
	authCodeURL := config.AuthCodeURL(connection.Data.GetString("state"), codeChallenge, codeChallengeMethod)

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

	// The connection must already be set up for this exchange to work.
	connection, err := service.connectionService.LoadOrCreateByProvider(providerID)

	if err != nil {
		return derp.NewBadRequestError(location, "Unknown OAuth Provider", providerID)
	}

	// Validate the state across requests
	if newState, _ := connection.Data.GetStringOK("state"); newState != state {
		return derp.NewBadRequestError(location, "Invalid OAuth State", state)
	}

	// Try to generate the OAuth token
	config := provider.OAuthConfig()

	token, err := config.Exchange(context.Background(), code,
		oauth2.SetAuthURLParam("code_verifier", connection.Data.GetString("code_challenge")),
		oauth2.SetAuthURLParam("redirect_uri", service.OAuthClientCallbackURL(providerID)))

	if err != nil {
		return derp.NewInternalError(location, "Error exchanging OAuth code for token", err.Error())
	}

	// Try to update the connection with the new token
	connection.Token = token
	connection.Data = mapof.NewString()
	connection.Active = true

	if service.connectionService.Save(&connection, "OAuth Exchange") != nil {
		return derp.NewInternalError(location, "Error saving domain")
	}

	// Success!
	return nil
}

// OAuthClientCallbackURL returns the specific callback URL to use for this host and provider.
func (service *Domain) OAuthClientCallbackURL(providerID string) string {
	return domain.Protocol(service.configuration.Hostname) + service.configuration.Hostname + "/oauth/connections/" + providerID + "/callback"
}

// NewOAuthState generates and returns a new OAuth state for the specified provider
func (service *Domain) NewOAuthClient(providerID string) (model.Connection, error) {

	const location = "service.Domain.NewOAuthState"

	// Find or Create a connection for this provider
	connection, _ := service.connectionService.LoadOrCreateByProvider(providerID)

	// Try to generate a new state
	newState, err := random.GenerateString(32)

	if err != nil {
		return model.Connection{}, derp.Wrap(err, location, "Error generating random string")
	}

	codeChallenge, err := random.GenerateString(64)

	if err != nil {
		return model.Connection{}, derp.Wrap(err, location, "Error generating random string")
	}

	// Assign the state to the connection and put into the domain
	connection.Data["state"] = newState
	connection.Data["code_challenge"] = codeChallenge

	// Save the domain
	if err := service.connectionService.Save(&connection, "New OAuth State"); err != nil {
		return model.Connection{}, derp.Wrap(err, location, "Error saving domain")
	}

	return connection, nil
}

// GetAuthToken retrieves the OAuth token for the specified provider.  If the token has expired
// then it is refreshed (and saved) automatically before returning.
func (service *Domain) GetOAuthToken(providerID string) (model.Connection, *oauth2.Token, error) {

	// Get the provider for this OAuth provider
	provider, ok := service.OAuthProvider(providerID)

	if !ok {
		return model.Connection{}, nil, derp.NewBadRequestError("service.Domain.GetOAuthToken", "Unknown OAuth Provider", providerID)
	}

	// Try to load the Connection config
	connection := model.NewConnection()
	if err := service.connectionService.LoadByProvider(providerID, &connection); err != nil {
		return model.Connection{}, nil, derp.NewBadRequestError("service.Domain.GetOAuthToken", "Error reading OAuth connection")
	}

	// Retrieve the Token from the connection
	token := connection.Token

	if token == nil {
		return model.Connection{}, token, derp.NewBadRequestError("service.Domain.GetOAuthToken", "No OAuth token found for provider", providerID)
	}

	// Use TokenSource to update tokens when they expire.
	config := provider.OAuthConfig()
	source := config.TokenSource(context.Background(), token)

	newToken, err := source.Token()

	if err != nil {
		return model.Connection{}, token, derp.Wrap(err, "service.Domain.GetOAuthToken", "Error refreshing OAuth token")
	}

	// If the token has changed, save it
	if token.AccessToken != newToken.AccessToken {
		connection.Token = newToken
		if err := service.connectionService.Save(&connection, "Refresh OAuth Token"); err != nil {
			return model.Connection{}, token, derp.Wrap(err, "service.Domain.GetOAuthToken", "Error saving refreshed Token")
		}
	}

	// Success!
	return connection, newToken, nil
}

/******************************************
 * Domain/Actor Methods
 ******************************************/

// Hostname returns the domain-only name (no protocol)
func (service *Domain) Hostname() string {
	return service.hostname
}

// ActorID returns the URL for this domain/actor
func (service *Domain) ActorID() string {
	return domain.AddProtocol(service.hostname) + "/@service"
}

// PublicKeyID returns the URL for the public key for this domain/actor
func (service *Domain) PublicKeyID() string {
	return service.ActorID() + "#main-key"
}

// PublicKeyPEM returns the PEM-encoded public key for this domain/actor
func (service *Domain) PublicKeyPEM() (string, error) {

	// Try to retrieve the private key for this domain
	privateKey, err := service.PrivateKey()

	if err != nil {
		return "", derp.Wrap(err, "service.Domain.PublicKeyPEM", "Error getting public key")
	}

	// Encode the public key portion
	publicKeyPEM := sigs.EncodePublicPEM(privateKey)
	return publicKeyPEM, nil
}

// PrivateKey returns the private key for this domain/actor
func (service *Domain) PrivateKey() (*rsa.PrivateKey, error) {

	const location = "service.Domain.PrivateKey"

	// Get the Domain record
	domain, err := service.LoadDomain()

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading Domain record")
	}

	// Try to use the existing private key
	if domain.PrivateKey != "" {

		privateKey, err := sigs.DecodePrivatePEM(domain.PrivateKey)

		if rsaKey, ok := privateKey.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}

		// Fall through means that we have a value for "domain.PrivateKey" but it's not
		// valid.  So, let's log the error and try to make a new one.
		derp.Report(derp.Wrap(err, location, "Error decoding private key. Creating a new key"))
	}

	// Otherwise, create a new private key, save it, and return it to the caller.
	privateKey, err := rsa.GenerateKey(rand.Reader, encryptionKeyBits)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error generating RSA key")
	}

	// Save the new private key into the Domain record
	domain.PrivateKey = sigs.EncodePrivatePEM(privateKey)

	if err := service.Save(domain, "Generated Private Key"); err != nil {
		return nil, derp.Wrap(err, location, "Error saving new EncryptionKey")
	}

	// Success??
	return privateKey, nil
}

/******************************************
 * WebFinger Behavior
 ******************************************/

func (service *Domain) LoadWebFinger(username string) (digit.Resource, error) {

	const location = "service.User.LoadWebFinger"

	if username != "acct:service@"+service.hostname {
		return digit.Resource{}, derp.NewBadRequestError(location, "Invalid username", username)
	}

	profileURL := domain.AddProtocol(service.hostname) + "/@service"

	// Make a WebFinger resource for this user.
	result := digit.NewResource("acct:service@"+service.hostname).
		Alias(profileURL).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, profileURL)

	return result, nil
}
