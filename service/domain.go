package service

import (
	"cmp"
	"context"
	"crypto/sha256"
	"html/template"
	"strings"
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

// Domain service manages all access to the singleton model.Domain in the database
type Domain struct {
	activityService     *ActivityStream
	configuration       config.Domain
	connectionService   *Connection
	domain              model.Domain
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
	return Domain{
		domain: model.NewDomain(),
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// collection returns the Domain collection for the provided database session
func (service *Domain) collection(session data.Session) data.Collection {
	return session.Collection("Domain")
}

// Refresh updates any stateful data that is cached inside this service.
//
// RULE: This method must NOT reset the cached Domain record.  Refresh runs on every configuration
// reload, but Start (the only thing that reloads the record from the database) runs only when the
// database connection or the hostname changes.  Blanking here therefore strands the service holding
// an empty Domain -- no Label, no PrivateKey, and a zero DomainID that makes the next Save INSERT a
// second record instead of updating the real one.  Start does the resetting, next to its Load.
func (service *Domain) Refresh(factory *Factory) {

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

// Start initializes the database, by:
// 1. Guaranteeing that a domain record exists in the db
// 2. synchronizing indexes
func (service *Domain) Start() error {

	const location = "service.Domain.Start"

	session, cancel, err := service.newSession(10 * time.Minute)

	if err != nil {
		return derp.Wrap(err, location, "Connecting to database")
	}

	defer cancel()

	// Reset the cached record HERE -- immediately before the Load that refills it -- so that this
	// service is never left holding a blank Domain.  See the RULE on Refresh.
	service.domain = model.NewDomain()

	// Try to load the domain model into memory
	err = service.collection(session).Load(exp.All(), &service.domain)

	switch {

	// If the domain record already exists, then bring its hostname up to date.
	case err == nil:
		if err := service.stampHostname(session); err != nil {
			return derp.Wrap(err, location, "Updating domain hostname")
		}

	// If "Not Found", then this is the first run, so bootstrap the domain and owner.
	case derp.IsNotFound(err):
		if err := service.bootstrap(session); err != nil {
			return derp.Wrap(err, location, "Bootstrapping new domain")
		}

	// Any other error is fatal.
	default:
		return derp.Wrap(err, location, "Loading domain record")
	}

	// ASYNC: Update database tables and indexes.  Both steps work through the connection the
	// factory already holds (read at call time, so a reconnect is picked up) -- the old
	// connect-string signatures dialed a fresh client per call and never disconnected it,
	// leaking one client per domain at boot and per domain change after.
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
		if err := queries.UpgradeMongoDB(ctx, database, &service.domain); err != nil {
			derp.Report(derp.Wrap(err, location, "Domain Not Ready: Error upgrading domain record"))
			return
		}

		// After any necessary upgrades, sync the indexes for the domain collection
		queries.SyncDomainIndexes(ctx, database)
	}()

	return nil
}

// bootstrap creates the initial domain record and, when configured, the owner account.
// Both writes happen inside a SINGLE transaction so the operation is atomic: if the owner
// cannot be created, the domain record is rolled back too.  The next server start then
// finds no domain record and cleanly retries the whole bootstrap, instead of stranding a
// domain record with no owner (which would leave the operator permanently locked out).
func (service *Domain) bootstrap(session data.Session) error {

	const location = "service.Domain.bootstrap"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Build the new domain record locally.  We publish it to the in-memory cache only
	// after the transaction commits, so a rollback never leaves the cache out of sync.
	//
	// The hostname must be stamped BEFORE persist(): Domain.Host() builds every derived URL from
	// it, and persist validates iconUrl/imageUrl as absolute URLs.  Without it, the very first
	// save of a new domain fails with "https:///..." -- a scheme and no authority.
	domain := service.domain
	domain.Hostname = service.hostname
	domain.Label = service.configuration.Label

	var owner *model.User

	if _, err := service.withTransaction(ctx, func(txn data.Session) (any, error) {

		// Create the singleton domain record
		if err := service.persist(txn, &domain, "Created Domain Record"); err != nil {
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

	// The transaction committed, so the in-memory cache can now reflect durable state.
	service.domain = domain

	// POST-COMMIT: invite a non-localhost owner to set their password (see inviteOwner).
	// This runs outside the transaction because sending email is an external side effect
	// that cannot be rolled back.
	if owner != nil {
		service.inviteOwner(session, owner)
	}

	return nil
}

// stampHostname writes the configured hostname into the stored Domain record whenever the two
// disagree.  The server configuration owns the hostname -- an operator can rename a domain in the
// setup tool at any time -- but the record needs its own copy because Domain.Host() builds every
// derived URL (federation actor, OAuth client metadata, oEmbed, email links) from it.  Writing it
// back on every Start is what keeps the stored copy from going stale after a rename.
func (service *Domain) stampHostname(session data.Session) error {

	const location = "service.Domain.stampHostname"

	// NO-OP: the stored record already agrees with the configuration
	if !needsHostnameStamp(service.domain.Hostname, service.hostname) {
		return nil
	}

	domain := service.domain
	domain.Hostname = service.hostname

	if err := service.Save(session, domain, "Updated Hostname"); err != nil {
		return derp.Wrap(err, location, "Saving Domain", service.hostname)
	}

	return nil
}

// needsHostnameStamp reports whether the stored hostname must be rewritten to match the configured
// one.  A blank configured hostname never overwrites a stored value: the setup console builds
// factories before its configuration is complete, and clearing a good hostname would break every
// URL the domain derives from it.
func needsHostnameStamp(stored string, configured string) bool {

	if configured == "" {
		return false
	}

	return stored != configured
}

// createOwner creates the domain owner account when the configuration requests one,
// returning the created User (or nil when owner creation is not configured).  It must be
// called inside the bootstrap transaction so the owner and domain record commit together.
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

// newOwnerFromConfig builds a domain owner User from the configured owner details,
// filling in default values for any field the operator left blank.  A blank email falls
// back to "admin@<hostname>" because User.Save requires a non-empty address.  Kept as a
// pure function (no database, no receiver) so the fallback rules are unit-testable.
func newOwnerFromConfig(configured config.Owner, hostname string) model.User {

	owner := model.NewUser()
	owner.DisplayName = cmp.Or(strings.TrimSpace(configured.DisplayName), "Demo")
	owner.Username = cmp.Or(strings.TrimSpace(configured.Username), "demo")
	owner.EmailAddress = cmp.Or(strings.TrimSpace(configured.EmailAddress), "demo@"+hostname)
	owner.IsOwner = true
	owner.IsPublic = true

	return owner
}

// ownerInviteMethod decides how a newly-bootstrapped owner receives their first password.
// Kept pure (no receiver, no database) so the policy is unit-testable.
type ownerInviteMethod int

const (
	// ownerInviteLocalhost means the convenience password is already set, so nothing needs to be sent
	ownerInviteLocalhost ownerInviteMethod = iota // convenience password already set; nothing to send
	// ownerInviteEmail means a password-reset link should be emailed to the owner
	ownerInviteEmail // public host + configured email: send a reset link
	// ownerInviteManual means the operator must set the owner's password by hand
	ownerInviteManual // public host + no email: operator must set one manually
)

// calcOwnerInviteMethod decides how a new Domain's owner is invited to set their password
func calcOwnerInviteMethod(isLocalhost bool, ownerEmail string) ownerInviteMethod {

	switch {

	case isLocalhost:
		return ownerInviteLocalhost

	case strings.TrimSpace(ownerEmail) != "":
		return ownerInviteEmail

	default:
		return ownerInviteManual
	}
}

// inviteOwner delivers a first-time password to a newly-bootstrapped owner.  Localhost
// owners already have the convenience password set in createOwner; public-host owners
// either receive a password-reset link (when an email was configured) or a clear, loud
// message pointing the operator at the setup console to set a password manually.
func (service *Domain) inviteOwner(session data.Session, owner *model.User) {

	switch calcOwnerInviteMethod(service.IsLocalhost(), service.configuration.Owner.EmailAddress) {

	case ownerInviteEmail:
		// Report-and-continue: owner bootstrap must not fail because the welcome email bounced.
		// The reset code is still issued, so the operator can recover once mail is fixed.
		if err := service.userService.SendPasswordResetEmail(session, owner, model.PasswordResetDurationWelcome); err != nil {
			derp.Report(derp.Wrap(err, "service.Domain.inviteOwner", "Sending owner welcome email", owner.Username))
		}

	case ownerInviteManual:
		// There is no password and no way to deliver one.  Surface a clear, loud message
		// so the operator is not silently locked out of their own server.
		log.Warn().
			Str("hostname", service.hostname).
			Str("username", owner.Username).
			Msg("Owner account created without a password. Configure an owner email address, or set a password from the server setup console (Domains > Users).")

	case ownerInviteLocalhost:
		// Nothing to do -- the convenience password is already set (see createOwner).
	}
}

/******************************************
 * Common Data Methods
 ******************************************/

// Get returns a pointer to the domain model object
func (service *Domain) Get() *model.Domain {
	return &service.domain
}

// Save updates the value of this domain in the database and refreshes the in-memory cache.
func (service *Domain) Save(session data.Session, domain model.Domain, note string) error {

	// Write the (validated) value to the database
	if err := service.persist(session, &domain, note); err != nil {
		return derp.Wrap(err, "service.Domain.Save", "Saving Domain")
	}

	// Update the in-memory cache to match what was just written
	service.domain = domain

	return nil
}

// persist validates a Domain and writes it to the database WITHOUT touching the
// in-memory cache.  Callers that write inside a transaction use this directly and
// publish to the cache themselves only after the transaction commits, so a rolled-back
// write never leaves the cache holding a record that isn't in the database.
func (service *Domain) persist(session data.Session, domain *model.Domain, note string) error {

	const location = "service.Domain.persist"

	// Validate the value using the default domain schema
	if _, err := schema.New(model.DomainSchema()).Validate(domain); err != nil {
		return derp.Wrap(err, location, "Validating Domain with standard Domain schema")
	}

	// Validate the value using the custom schema for this domain
	if _, err := service.Schema().Validate(domain); err != nil {
		return derp.Wrap(err, location, "Validating Domain with custom schema from Theme")
	}

	// If the MLS mode is not "Groups", then clear all group IDs
	if domain.MLSMode != model.DomainMLSModeGroups {
		domain.MLSGroupIDs = sliceof.NewString()
	}

	// Try to save the value to the database
	if err := service.collection(session).Save(domain, note); err != nil {
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

// ObjectNew returns a fully initialized model.Domain as a data.Object.
func (service *Domain) ObjectNew() data.Object {
	result := model.NewDomain()
	return &result
}

// ObjectID returns the unique ID of the provided Domain. Implements the ModelService interface.
func (service *Domain) ObjectID(object data.Object) primitive.ObjectID {

	if domain, ok := object.(*model.Domain); ok {
		return domain.DomainID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every Domain that matches the provided criteria. Implements the ModelService interface.
func (service *Domain) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single Domain as a data.Object. Implements the ModelService interface.
func (service *Domain) ObjectLoad(_ data.Session, _ exp.Expression) (data.Object, error) {
	return &service.domain, nil
}

// ObjectSave adds or updates a Domain in the database. Implements the ModelService interface.
func (service *Domain) ObjectSave(session data.Session, object data.Object, note string) error {
	if domain, ok := object.(*model.Domain); ok {
		return service.Save(session, *domain, note)
	}

	return derp.Internal("service.Domain.ObjectSave", "Invalid Object Type", object)
}

// ObjectDelete marks a Domain as deleted. Implements the ModelService interface.
func (service *Domain) ObjectDelete(session data.Session, object data.Object, note string) error {
	return derp.BadRequest("service.Domain.ObjectDelete", "Unsupported")
}

// ObjectUserCan reports whether the provided Authorization may run an action on a Domain. Implements the ModelService interface.
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
	return service.themeService.GetTheme(service.domain.ThemeID)
}

// HasRegistrationForm returns TRUE if this domain allows new users to sign up.
func (service *Domain) HasRegistrationForm() bool {
	return service.domain.HasRegistrationForm()
}

// LoadRegistration returns the sign-up Registration configured for this Domain
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
	return uri.IsLocalHostname(service.hostname)
}

/******************************************
 * OAuth Handshake Methods
 ******************************************/

// OAuthCodeURL generates a new (unique) OAuth state and AuthCodeURL for the specified provider
func (service *Domain) OAuthCodeURL(session data.Session, providerID string) (string, error) {

	const location = "service.Domain.OAuthCodeURL"

	// Get the provider for this provider
	provider, ok := service.OAuthProvider(providerID)

	if !ok {
		return "", derp.BadRequest(location, "Unknown OAuth Provider", providerID)
	}

	// Set a new "state" for this provider
	connection, err := service.NewOAuthClient(session, providerID)

	if err != nil {
		return "", derp.Wrap(err, location, "Generating new OAuth connection")
	}

	// Generate and return the AuthCodeURL
	config := provider.OAuthConfig()

	config.RedirectURL = service.OAuthClientCallbackURL(providerID)
	codeChallengeBytes := sha256.Sum256([]byte(connection.Data.GetString("code_challenge")))
	codeChallenge := oauth2.SetAuthURLParam("code_challenge", random.Base64URLEncode(codeChallengeBytes[:]))
	codeChallengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "S256")

	// INSECURE? Unhashed Code challenge method
	// codeChallenge := oauth2.SetAuthURLParam("code_challenge", connection.Data.GetString("code_challenge"))
	// codeChallengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "plain")
	authCodeURL := config.AuthCodeURL(connection.Data.GetString("state"), codeChallenge, codeChallengeMethod)

	return authCodeURL, nil
}

// OAuthExchange trades a temporary OAuth code for a valid OAuth token
func (service *Domain) OAuthExchange(session data.Session, providerID string, state string, code string) error {

	const location = "service.Domain.OAuthExchange"

	// Get the provider for this provider
	provider, ok := service.OAuthProvider(providerID)

	if !ok {
		return derp.BadRequest(location, "Unknown OAuth Provider", providerID)
	}

	// The connection must already be set up for this exchange to work.
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

// oauthHTTPContext returns a context carrying an SSRF-hardened HTTP client for the
// golang.org/x/oauth2 handshake. oauth2 POSTs to the provider's TokenURL using the client
// found in the context (else http.DefaultClient, which is UNGUARDED). Provider endpoints are
// admin-configured (not attacker-controlled), so this is defense-in-depth, but it keeps every
// oauth2 handshake on the same guarded transport. Private addresses are allowed only when this
// instance allows them, matching the ActivityStream client's policy.
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
	connection, _ := service.connectionService.LoadOrCreateByProvider(session, providerID)

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

	// Make a WebFinger resource for this user.
	result := digit.NewResource("acct:service@"+service.hostname).
		Alias(profileURL).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, profileURL)

	return result, nil
}
