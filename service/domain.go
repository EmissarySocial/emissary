package service

import (
	"cmp"
	"context"
	"crypto/sha256"
	"html/template"
	"strings"
	"sync/atomic"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/exp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/oauth2"
)

// Domain service manages all access to the singleton Domain record in the database
type Domain struct {
	activityService     *ActivityStream
	configuration       config.Domain
	connectionService   *Connection
	cache               atomic.Pointer[model.WritableDomain] // the published Domain record.  Never modify a published value.
	funcMap             template.FuncMap
	database            func() *mongo.Database
	newSession          func(time.Duration) (data.Session, context.CancelFunc, error)
	withTransaction     func(context.Context, data.TransactionCallbackFunc) (any, error)
	providerService     *Provider
	registrationService *Registration
	steranko            func(data.Session) *steranko.Steranko
	themeService        *Theme
	userService         *User
	hostname            string
	host                string
}

// NewDomain returns a fully initialized Domain service
func NewDomain() Domain {
	return Domain{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// collection returns the Domain collection for the provided database session
func (service *Domain) collection(session data.Session) data.Collection {
	return session.Collection("Domain")
}

// Refresh updates any stateful data that is cached inside this service.
func (service *Domain) Refresh(factory *Factory) {

	// RULE: Never reset the cached Domain record here.  Refresh runs on every configuration reload,
	// but only Start and the watcher reload the record, so a blank here strands an empty Domain.
	// That blank has no CreateDate, so its next Save tries to INSERT a second record.

	service.activityService = factory.ActivityStream()
	service.configuration = factory.config
	service.connectionService = factory.Connection()
	service.funcMap = factory.FuncMap()
	service.database = factory.Database
	service.newSession = factory.Session
	service.withTransaction = factory.WithTransaction
	service.providerService = factory.Provider()
	service.registrationService = factory.Registration()
	service.steranko = factory.Steranko
	service.themeService = factory.Theme()
	service.userService = factory.User()
	service.hostname = factory.Hostname()
	service.host = factory.Host()
}

// Start loads and publishes the Domain record, creating it on the first run, then upgrades the
// database and syncs its indexes in the background.
func (service *Domain) Start() error {

	const location = "service.Domain.Start"

	session, cancel, err := service.newSession(10 * time.Minute)

	if err != nil {
		return derp.Wrap(err, location, "Connecting to database")
	}

	defer cancel()

	// Reset the cached record HERE, next to the Load that refills it, and never in Refresh.
	// A failed Load leaves this blank published.
	service.publish(model.NewWritableDomain())

	// Try to load the stored Domain record into memory
	writableDomain := model.NewWritableDomain()

	if err := service.Load(session, &writableDomain); err != nil {

		// Anything BUT a "Not Found" error is fatal.
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Loading domain record")
		}

		// Otherwise, this is the first run, so bootstrap the domain and owner into the same record.
		if err := service.bootstrap(session, &writableDomain); err != nil {
			return derp.Wrap(err, location, "Bootstrapping new domain")
		}
	}

	// Publish the loaded or bootstrapped domain record to the cache.
	service.publish(writableDomain)

	if err := service.stampHostname(session); err != nil {
		return derp.Wrap(err, location, "Updating domain hostname")
	}

	// ASYNC: Upgrade the database and sync its indexes through the connection the factory already
	// holds, read at call time so a reconnect is picked up.  Never dial a client here: one that is
	// never disconnected leaks per domain, at boot and on every domain change.
	go func() {

		// RULE: Bounded.  Nothing holds a lock here, but an unreachable server must not pin
		// this goroutine (and the migration it guards) forever.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		database := service.database()

		if database == nil {
			derp.Report(derp.Internal(location, "Domain Not Ready: no database connection for upgrades and index sync"))
			return
		}

		// Once we have the domain loaded, try to upgrade the database
		readOnlyDomain := service.Cached()
		if err := queries.UpgradeMongoDB(ctx, database, readOnlyDomain.DatabaseVersion); err != nil {
			derp.Report(derp.Wrap(err, location, "Domain Not Ready: Error upgrading domain record"))
			return
		}

		// After any necessary upgrades, sync the indexes for the domain collection
		queries.SyncDomainIndexes(ctx, database)
	}()

	return nil
}

// bootstrap fills domain with the initial Domain record and creates it, with the configured owner
// account, in ONE transaction.  The caller publishes the record once bootstrap returns.
func (service *Domain) bootstrap(session data.Session, writableDomain *model.WritableDomain) error {

	const location = "service.Domain.bootstrap"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Stamp the hostname BEFORE persist: Domain.Host() builds every derived URL from it.
	writableDomain.Hostname = service.hostname
	writableDomain.Label = service.configuration.Label

	var owner *model.User

	if _, err := service.withTransaction(ctx, func(txn data.Session) (any, error) {

		// Create the singleton domain record
		if err := service.persist(txn, writableDomain, "Created Domain Record"); err != nil {
			return nil, derp.Wrap(err, location, "Creating domain record")
		}

		// When configured, create the owner account in the SAME transaction
		newOwner, err := service.createOwner(txn)

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating owner account")
		}

		owner = newOwner
		return nil, nil

	}); err != nil {
		return derp.Wrap(err, location, "Initializing domain")
	}

	// POST-COMMIT: invite a non-localhost owner to set their password (see inviteOwner).
	// This runs outside the transaction because sending email is an external side effect
	// that cannot be rolled back.
	if owner != nil {
		service.inviteOwner(session, owner)
	}

	return nil
}

// stampHostname writes the configured hostname into the stored Domain record whenever the two
// disagree (see AGENTS.md).
func (service *Domain) stampHostname(session data.Session) error {

	const location = "service.Domain.stampHostname"

	// NO-OP: the cached record already agrees with the configuration
	if !needsHostnameStamp(service.Cached().Hostname, service.hostname) {
		return nil
	}

	// Start the write from the stored record, never from the cache
	writableDomain := model.NewWritableDomain()

	if err := service.Load(session, &writableDomain); err != nil {
		return derp.Wrap(err, location, "Loading Domain")
	}

	// NO-OP: another node stamped it since the cached record was read
	if !needsHostnameStamp(writableDomain.Hostname, service.hostname) {
		return nil
	}

	writableDomain.Hostname = service.hostname

	if err := service.Save(session, &writableDomain, "Updated Hostname"); err != nil {
		return derp.Wrap(err, location, "Saving Domain", service.hostname)
	}

	return nil
}

// needsHostnameStamp reports whether the stored hostname must be rewritten to match the configured
// one.  A blank configured hostname never overwrites a stored value.
func needsHostnameStamp(stored string, configured string) bool {

	if configured == "" {
		return false
	}

	return stored != configured
}

// createOwner creates the configured owner account, or returns nil when none is configured.  Call it
// inside the bootstrap transaction, so that the owner and the Domain record commit together.
func (service *Domain) createOwner(session data.Session) (*model.User, error) {

	const location = "service.Domain.createOwner"

	// Nothing to do unless the operator asked us to create an owner
	if !service.configuration.CreateOwner {
		return nil, nil
	}

	log.Trace().Str("hostname", service.hostname).Msg("Creating owner account")

	// Build the owner from the configured details, falling back to sensible defaults.
	owner := newOwnerFromConfig(service.configuration.Owner, service.hostname)

	// RULE: On localhost (e.g. the demo image) set a convenience password so the operator
	// can sign in immediately.  We NEVER ship a known default credential on a public host;
	// those owners set their own password via the emailed reset link (see inviteOwner).
	if service.IsLocalhost() {
		if err := service.steranko(session).SetPassword(&owner, "demo"); err != nil {
			return nil, derp.Wrap(err, location, "Setting owner password")
		}
	}

	// Save the owner account
	if err := service.userService.Save(session, &owner, "Created owner account"); err != nil {
		return nil, derp.Wrap(err, location, "Saving owner account")
	}

	log.Trace().Str("username", owner.Username).Msg("Created owner account")
	return &owner, nil
}

// newOwnerFromConfig builds the domain owner from the configured details, defaulting every blank
// field.  A blank email becomes "demo@<hostname>", because User.Save requires an address.
func newOwnerFromConfig(configured config.Owner, hostname string) model.User {

	owner := model.NewUser()
	owner.DisplayName = cmp.Or(strings.TrimSpace(configured.DisplayName), "Demo")
	owner.Username = cmp.Or(strings.TrimSpace(configured.Username), "demo")
	owner.EmailAddress = cmp.Or(strings.TrimSpace(configured.EmailAddress), "demo@"+hostname)
	owner.IsOwner = true
	owner.IsPublic = true

	return owner
}

// inviteOwner delivers a first password to a newly-bootstrapped owner: nothing on localhost, a
// password-reset link when an email is configured, and otherwise a loud warning to the operator.
func (service *Domain) inviteOwner(session data.Session, owner *model.User) {

	// Don't send emails on localhost
	if service.IsLocalhost() {
		return
	}

	// Don't send emails if there isn't an email configured
	if service.configuration.Owner.EmailAddress == "" {
		return
	}

	// Report-and-continue: owner bootstrap must not fail because the welcome email bounced.
	// The reset code is still issued, so the operator can recover once mail is fixed.
	if err := service.userService.SendPasswordResetEmail(session, owner, model.PasswordResetDurationWelcome); err != nil {
		derp.Report(derp.Wrap(err, "service.Domain.inviteOwner", "Sending owner welcome email", owner.Username))
	}
}

/******************************************
 * Common Data Methods
 ******************************************/

// Cached returns the cached, read-only Domain record.  A service that has never loaded one returns a
// blank Domain.  To change the record, Load a WritableDomain and Save it (see AGENTS.md).
func (service *Domain) Cached() *model.Domain {

	if result := service.cache.Load(); result != nil {
		return &result.Domain
	}

	// Publish a blank record only if nothing else got there first, so every caller shares one value
	blank := model.NewWritableDomain()
	service.cache.CompareAndSwap(nil, &blank)

	return &service.cache.Load().Domain
}

// Load reads the stored Domain record into writableDomain, which the caller builds with model.NewWritableDomain()
func (service *Domain) Load(session data.Session, writableDomain *model.WritableDomain) error {

	const location = "service.Domain.Load"

	if err := service.collection(session).Load(exp.All(), writableDomain); err != nil {
		return derp.Wrap(err, location, "Loading Domain record")
	}

	return nil
}

// publish replaces the cached Domain record with a copy of the provided value
func (service *Domain) publish(writableDomain model.WritableDomain) {

	// The caller keeps its value and may go on editing it, so the cache holds its own maps and slices.
	// Clone is the embedded Domain's; the Journal holds only scalars and is copied as-is.
	clone := model.WritableDomain{
		Domain:  writableDomain.Clone(),
		Journal: writableDomain.Journal,
	}

	service.cache.Store(&clone)
}

// Save updates the value of this domain in the database and refreshes the in-memory cache.
func (service *Domain) Save(session data.Session, writableDomain *model.WritableDomain, note string) error {

	// Write the (validated) value to the database
	if err := service.persist(session, writableDomain, note); err != nil {
		return derp.Wrap(err, "service.Domain.Save", "Saving Domain")
	}

	// Update the in-memory cache to match what was just written
	service.publish(*writableDomain)

	return nil
}

// persist validates a Domain and writes it to the database WITHOUT publishing it.  A caller inside
// a transaction publishes only after the commit (see AGENTS.md).
func (service *Domain) persist(session data.Session, writableDomain *model.WritableDomain, note string) error {

	const location = "service.Domain.persist"

	// Validate the value using the default domain schema
	if _, err := schema.New(model.DomainSchema()).Validate(writableDomain); err != nil {
		return derp.Wrap(err, location, "Validating Domain with standard Domain schema")
	}

	// Validate again with Schema(), which returns the same standard schema, not the Theme's
	if _, err := service.Schema().Validate(writableDomain); err != nil {
		return derp.Wrap(err, location, "Validating Domain with custom schema from Theme")
	}

	// If the MLS mode is not "Groups", then clear all group IDs
	if writableDomain.MLSMode != model.DomainMLSModeGroups {
		writableDomain.MLSGroupIDs = sliceof.NewString()
	}

	// Try to save the value to the database
	if err := service.collection(session).Save(writableDomain, note); err != nil {
		return derp.Wrap(err, location, "Saving Domain")
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// Count returns the number of records that match the provided criteria
func (service *Domain) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// ObjectType returns the type of object that this service manages
func (service *Domain) ObjectType() string {
	return "Domain"
}

// ObjectNew returns a fully initialized model.WritableDomain as a data.Object.
func (service *Domain) ObjectNew() data.Object {
	writableDomain := model.NewWritableDomain()
	return &writableDomain
}

// ObjectID returns the unique ID of the provided Domain. Implements the ModelService interface.
func (service *Domain) ObjectID(object data.Object) primitive.ObjectID {

	if writableDomain, ok := object.(*model.WritableDomain); ok {
		return writableDomain.DomainID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every Domain that matches the provided criteria. Implements the ModelService interface.
func (service *Domain) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad returns a copy of the stored Domain record, whatever the criteria. Implements the ModelService interface.
func (service *Domain) ObjectLoad(session data.Session, _ exp.Expression) (data.Object, error) {

	writableDomain := model.NewWritableDomain()

	if err := service.Load(session, &writableDomain); err != nil {
		return nil, derp.Wrap(err, "service.Domain.ObjectLoad", "Loading Domain")
	}

	return &writableDomain, nil
}

// ObjectSave adds or updates a Domain in the database. Implements the ModelService interface.
func (service *Domain) ObjectSave(session data.Session, object data.Object, note string) error {
	if writableDomain, ok := object.(*model.WritableDomain); ok {
		return service.Save(session, writableDomain, note)
	}

	return derp.Internal("service.Domain.ObjectSave", "Invalid Object Type", object)
}

// ObjectDelete refuses to delete the Domain. Implements the ModelService interface.
func (service *Domain) ObjectDelete(session data.Session, object data.Object, note string) error {
	return derp.BadRequest("service.Domain.ObjectDelete", "Unsupported")
}

// ObjectUserCan refuses every action on the Domain. Implements the ModelService interface.
func (service *Domain) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Domain", "Not Authorized")
}

// Schema returns the rosetta schema that describes a Domain
func (service *Domain) Schema() schema.Schema {
	return schema.New(model.DomainSchema())
}

/******************************************
 * Provider Methods
 ******************************************/

// Theme returns the Theme that this Domain is displayed with
func (service *Domain) Theme() model.Theme {
	return service.themeService.GetTheme(service.Cached().ThemeID)
}

// HasRegistrationForm returns TRUE if this domain allows new users to sign up.
func (service *Domain) HasRegistrationForm() bool {
	return service.Cached().HasRegistrationForm()
}

// LoadRegistration returns the sign-up Registration configured for this Domain
func (service *Domain) LoadRegistration() model.Registration {

	if registrationID := service.Cached().RegistrationID; registrationID != "" {
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

// ManualProvider returns the providers.ManualProvider that matches the given providerID
func (service *Domain) ManualProvider(providerID string) (providers.ManualProvider, bool) {

	if provider, ok := service.Provider(providerID); ok {

		if manualProvider, ok := provider.(providers.ManualProvider); ok {
			return manualProvider, true
		}
	}

	return nil, false
}

// OAuthProvider returns the providers.OAuthProvider that matches the given providerID
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
	return uri.IsLocalHostname(service.hostname)
}

/******************************************
 * OAuth Handshake Methods
 ******************************************/

// OAuthCodeURL generates a new (unique) OAuth state and AuthCodeURL for the specified provider
func (service *Domain) OAuthCodeURL(session data.Session, providerID string) (string, error) {

	const location = "service.Domain.OAuthCodeURL"

	// Get the OAuth provider for this providerID
	provider, ok := service.OAuthProvider(providerID)

	if !ok {
		return "", derp.BadRequest(location, "Unknown OAuth Provider", providerID)
	}

	// Set a new "state" for this provider
	connection, err := service.NewOAuthClient(session, providerID)

	if err != nil {
		return "", derp.Wrap(err, location, "Generating new OAuth connection")
	}

	// Generate and return the AuthCodeURL.  The code challenge is hashed with S256, because the
	// unhashed "plain" method is insecure.
	config := provider.OAuthConfig()

	config.RedirectURL = service.OAuthClientCallbackURL(providerID)
	codeChallengeBytes := sha256.Sum256([]byte(connection.Data.GetString("code_challenge")))
	codeChallenge := oauth2.SetAuthURLParam("code_challenge", random.Base64URLEncode(codeChallengeBytes[:]))
	codeChallengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	authCodeURL := config.AuthCodeURL(connection.Data.GetString("state"), codeChallenge, codeChallengeMethod)

	return authCodeURL, nil
}

// OAuthExchange trades a temporary OAuth code for a valid OAuth token
func (service *Domain) OAuthExchange(session data.Session, providerID string, state string, code string) error {

	const location = "service.Domain.OAuthExchange"

	// Get the OAuth provider for this providerID
	provider, ok := service.OAuthProvider(providerID)

	if !ok {
		return derp.BadRequest(location, "Unknown OAuth Provider", providerID)
	}

	// Load the Connection that holds the state and code challenge from OAuthCodeURL
	connection, err := service.connectionService.LoadOrCreateByProvider(session, providerID)

	if err != nil {
		return derp.BadRequest(location, "Unknown OAuth Provider", providerID)
	}

	// Validate the state across requests
	if newState, _ := connection.Data.GetStringOK("state"); newState != state {
		return derp.BadRequest(location, "Invalid OAuth State", state)
	}

	// Try to generate the OAuth token
	config := provider.OAuthConfig()

	token, err := config.Exchange(service.oauthHTTPContext(), code,
		oauth2.SetAuthURLParam("code_verifier", connection.Data.GetString("code_challenge")),
		oauth2.SetAuthURLParam("redirect_uri", service.OAuthClientCallbackURL(providerID)))

	if err != nil {
		return derp.Internal(location, "Unable to exchange OAuth code for token", err.Error())
	}

	// Try to update the connection with the new token
	connection.Token = token
	connection.Data = mapof.NewAny()
	connection.Active = true

	if service.connectionService.Save(session, &connection, "OAuth Exchange") != nil {
		return derp.Internal(location, "Unable to save domain")
	}

	// Success!
	return nil
}

// oauthHTTPContext returns a context carrying an SSRF-hardened HTTP client for the oauth2 handshake,
// which otherwise falls back to the unguarded http.DefaultClient (see AGENTS.md).
func (service *Domain) oauthHTTPContext() context.Context {
	client := remote.NewHTTPClient(service.activityService.AllowPrivateIPs())
	return context.WithValue(context.Background(), oauth2.HTTPClient, client)
}

// OAuthClientCallbackURL returns the specific callback URL to use for this host and provider.
func (service *Domain) OAuthClientCallbackURL(providerID string) string {
	return uri.GuessProtocolForHostname(service.configuration.Hostname) + service.configuration.Hostname + "/oauth/connections/" + providerID + "/callback"
}

// NewOAuthClient generates and returns a new OAuth state for the specified provider
func (service *Domain) NewOAuthClient(session data.Session, providerID string) (model.Connection, error) {

	const location = "service.Domain.NewOAuthClient"

	// Find or Create a connection for this provider
	connection, err := service.connectionService.LoadOrCreateByProvider(session, providerID)

	// RULE: The error must not be discarded.  Its failure paths return a ZERO Connection whose
	// Data map is nil, and the assignments below write into that map -- so a dropped error here
	// is a panic, not a degraded result.
	if err != nil {
		return model.Connection{}, derp.Wrap(err, location, "Loading Connection", providerID)
	}

	// Try to generate a new state
	newState, err := random.GenerateString(32)

	if err != nil {
		return model.Connection{}, derp.Wrap(err, location, "Generating random string")
	}

	codeChallenge, err := random.GenerateString(64)

	if err != nil {
		return model.Connection{}, derp.Wrap(err, location, "Generating random string")
	}

	// Assign the state to the connection and put into the domain
	connection.Data["state"] = newState
	connection.Data["code_challenge"] = codeChallenge

	// Save the domain
	if err := service.connectionService.Save(session, &connection, "New OAuth State"); err != nil {
		return model.Connection{}, derp.Wrap(err, location, "Saving domain")
	}

	return connection, nil
}

// GetOAuthToken retrieves the OAuth token for the specified provider.  If the token has expired
// then it is refreshed (and saved) automatically before returning.
func (service *Domain) GetOAuthToken(session data.Session, providerID string) (model.Connection, *oauth2.Token, error) {

	// Get the provider for this OAuth provider
	provider, ok := service.OAuthProvider(providerID)

	if !ok {
		return model.Connection{}, nil, derp.BadRequest("service.Domain.GetOAuthToken", "Unknown OAuth Provider", providerID)
	}

	// Try to load the Connection config
	connection := model.NewConnection()
	if err := service.connectionService.LoadByProvider(session, providerID, &connection); err != nil {
		return model.Connection{}, nil, derp.BadRequest("service.Domain.GetOAuthToken", "Unable to read OAuth connection")
	}

	// Retrieve the Token from the connection
	token := connection.Token

	if token == nil {
		return model.Connection{}, token, derp.BadRequest("service.Domain.GetOAuthToken", "No OAuth token found for provider", providerID)
	}

	// Use TokenSource to update tokens when they expire.
	config := provider.OAuthConfig()
	source := config.TokenSource(service.oauthHTTPContext(), token)

	newToken, err := source.Token()

	if err != nil {
		return model.Connection{}, token, derp.Wrap(err, "service.Domain.GetOAuthToken", "Refreshing OAuth token")
	}

	// If the token has changed, save it
	if token.AccessToken != newToken.AccessToken {
		connection.Token = newToken
		if err := service.connectionService.Save(session, &connection, "Refresh OAuth Token"); err != nil {
			return model.Connection{}, token, derp.Wrap(err, "service.Domain.GetOAuthToken", "Saving refreshed Token")
		}
	}

	// Success!
	return connection, newToken, nil
}

/******************************************
 * WebFinger Behavior
 ******************************************/

// LoadWebFinger returns the WebFinger resource that describes this Domain's service Actor
func (service *Domain) LoadWebFinger(username string) (digit.Resource, error) {

	const location = "service.User.LoadWebFinger"

	if username != "service@"+service.hostname {
		return digit.Resource{}, derp.BadRequest(location, "Invalid username", username)
	}

	profileURL := uri.PrependProtocol(service.hostname) + "/@application"

	// Make a WebFinger resource for the service actor
	result := digit.NewResource("acct:service@"+service.hostname).
		Alias(profileURL).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, profileURL)

	return result, nil
}
