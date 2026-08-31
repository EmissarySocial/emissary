package build

import (
	"html/template"
	"net"
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/EmissarySocial/emissary/tools/asrules"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/form"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/sherlock"
	"github.com/benpate/sniff"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Common provides common building functions that are needed by ALL builders
type Common struct {
	_factory       Factory             // Factory interface is required for locating other services.
	_session       data.Session        // Database session for all db requests
	_request       *http.Request       // Pointer to the HTTP request we are serving
	_response      http.ResponseWriter // ResponseWriter for this request
	_authorization model.Authorization // Authorization information for the current website visitor
	_user          *model.User         // User information for the current website User (if any)
	_identity      *model.Identity     // Identity information for the current website visitor (if any)

	arguments mapof.String // Temporary data scope for this request
}

// NewCommon returns the Common builder that every other Builder embeds
func NewCommon(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter) Common {

	// Retrieve the user's authorization information
	steranko := factory.Steranko(session)
	authorization := getAuthorization(steranko, request)

	// Return a new Common builder
	return Common{
		_factory:       factory,
		_session:       session,
		_request:       request,
		_response:      response,
		_authorization: authorization,
		arguments:      make(mapof.String),
	}
}

/******************************************
 * Builder Interface
 ******************************************/

// context returns request context embedded in this builder.
func (w Common) factory() Factory {
	return w._factory
}

// session returns the database session that this Builder is using.
func (w Common) session() data.Session {
	return w._session
}

// request returns the original http.Request that we are responding to.
func (w Common) request() *http.Request {
	return w._request
}

// response returns the original http.ResponseWriter that we are writing to.
func (w Common) response() http.ResponseWriter {
	return w._response
}

// authorization returns the user's authorization data from the context.
func (w Common) authorization() model.Authorization {
	return w._authorization
}

/******************************************
 * Page Defaults
 ******************************************/

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w Common) PageTitle() string {
	return ""
}

// Summary returns the human-friendly summary to display at the top of the page. Implements the Builder interface.
func (w Common) Summary() string {
	return ""
}

/******************************************
 * Request Info
 ******************************************/

// Method returns the HTTP method of the current request
func (w Common) Method() string {
	return w._request.Method
}

// Host is the absolute origin of this domain: protocol + hostname + port.
// e.g. "http://localhost:8080" (dev) or "https://example.com" (prod).
//
// The port comes from the browser's request -- NOT server config -- so absolute
// URLs stay correct behind a proxy or port-map, where the public port differs
// from the internal HTTP port (e.g. Docker 8080:80). The hostname always comes
// from the validated domain config, never the request, so a spoofed Host header
// cannot inject a foreign domain into generated links.
func (w Common) Host() string {
	return w.Protocol() + w.Hostname() + w.requestPort()
}

// requestPort returns the ":port" the browser used for this request (from the
// Host header), or "" for the standard port (80/443) or when no port is present.
// Reading the port from the request -- rather than server config -- is what keeps
// Host() correct when the public and internal ports differ.
//
// NOTE: behind a reverse proxy that rewrites the Host header, the proxy must
// preserve the original Host (standard practice) for this to be correct.
// Honoring X-Forwarded-Host / X-Forwarded-Port is a follow-up.
func (w Common) requestPort() string {

	if w._request == nil {
		return ""
	}

	_, port, err := net.SplitHostPort(w._request.Host)

	if err != nil {
		return "" // No explicit port in the Host header -> standard port
	}

	switch port {
	case "", "80", "443":
		return ""
	default:
		return ":" + port
	}
}

// Hostname is the bare domain only -- no protocol, no port. Use for federation
// identity (@user@hostname) and comparisons. e.g. "localhost".
func (w Common) Hostname() string {
	return w._factory.Hostname()
}

// Protocol is the scheme for this domain, including "://". e.g. "http://".
func (w Common) Protocol() string {
	return uri.GuessProtocolForHostname(w.Hostname())
}

// URL is the current page as an ABSOLUTE url (Host + path + query). Use for
// canonical links, og:url, and email -- anywhere a full URL is required.
// e.g. "http://localhost:8080/@me/settings".
func (w Common) URL() string {
	return w.Host() + w._request.URL.RequestURI()
}

// RelativeURL is the current page as a ROOT-RELATIVE ref -- path + query, no origin.
// Use for form actions and htmx targets: the browser supplies the origin, so it is
// correct on any port or proxy (unlike URL, which hard-codes the server's own port).
// e.g. "/@me/settings".
func (w Common) RelativeURL() string {
	return w._request.URL.RequestURI()
}

// Path is the request path only -- no origin, no query. e.g. "/@me/settings".
func (w Common) Path() string {
	return w._request.URL.Path
}

// PathList returns the HTTP Request path as a List
// of strings
func (w Common) PathList() list.List {
	return list.BySlash(w.Path()).Tail()
}

// SetQueryParam updates the HTTP request, setting a new value
// for an individual query parameter.
func (w Common) SetQueryParam(name string, value string) string {
	query := w._request.URL.Query()
	query.Set(name, value)
	w._request.URL.RawQuery = query.Encode()
	return ""
}

// DefaultQueryParam sets a query parameter only if it is not already present. Implements the Builder interface.
func (w Common) DefaultQueryParam(name string, value string) string {
	query := w._request.URL.Query()

	if query.Get(name) != "" {
		return ""
	}

	query.Set(name, value)
	w._request.URL.RawQuery = query.Encode()
	return ""
}

// QueryParam returns the named query string parameter.  If the parameter has
// multiple values, then only the first value is returned.
func (w Common) QueryParam(param string) string {
	return w._request.URL.Query().Get(param)
}

// QueryString returns the raw query string (encoded as a template.URL)
// to be re-embedded in a template link.
func (w Common) QueryString() template.URL {
	return template.URL(w._request.URL.RawQuery)
}

// RawQuery returns the raw query string (encoded as a string)
func (w Common) RawQuery() string {
	return w._request.URL.RawQuery
}

// IsPartialRequest returns TRUE if this is a partial page request from htmx.
func (w Common) IsPartialRequest() bool {
	return w._request.Header.Get("HX-Request") != ""
}

// templateRole returns the the role that the current Template performs in the system.
// Used for selecting eligible child templates.
func (w Common) templateRole() string {
	return ""
}

// UserCan returns TRUE if the current user has the specified permission.
// Default implementation returns FALSE for all requests.
func (w Common) UserCan(_ string) bool {
	return false
}

// UserCanMLS returns TRUE if the current user has permission to use MLS E2EE messaging
func (w Common) UserCanMLS() bool {
	if user, err := w.getUser(); err == nil {
		result := w._factory.Domain().Get().UserCanMLS(user)
		return result
	}

	return false
}

// HasUnreadNotifications returns TRUE if the authenticated User has at least one unread
// notification.  It is available on every builder (via Common) so the global nav dot can
// render on any page.  No counts — the dot is a simple yes/no.
func (w Common) HasUnreadNotifications() bool {

	if w.AuthenticatedID().IsZero() {
		return false
	}

	hasUnread, err := w._factory.Notification().HasUnread(w._session, w.AuthenticatedID())

	if err != nil {
		derp.Report(derp.Wrap(err, "build.Common.HasUnreadNotifications", "Checking unread notifications", w.AuthenticatedID()))
		return false
	}

	return hasUnread
}

// WebPushPublicKey returns this domain's VAPID public key (generating it on first use), so the
// browser can subscribe to Web Push.  Returns "" if the user is not authenticated or key generation
// fails (the UI degrades to no push).
func (w Common) WebPushPublicKey() string {

	if w.AuthenticatedID().IsZero() {
		return ""
	}

	publicKey, err := w._factory.WebPush().PublicKey(w._session)

	if err != nil {
		derp.Report(derp.Wrap(err, "build.Common.WebPushPublicKey", "Loading VAPID public key"))
		return ""
	}

	return publicKey
}

// UserCanBridgeToBluesky returns TRUE if the current user has permission to bridge to Bluesky
func (w Common) UserCanBridgeToBluesky() bool {
	if user, err := w.getUser(); err == nil {
		result := w._factory.Domain().Get().UserCanBridgeToBluesky(user)
		return result
	}

	return false
}

// HasConnectionProvider returns TRUE if this domain has an active connection for the named provider
func (w Common) HasConnectionProvider(provider string) bool {
	return w.factory().Domain().Get().HasConnectionProvider(provider)
}

// ThemeID returns the ID of the Theme that this Domain has selected.
func (w Common) ThemeID() string {
	return w.factory().Domain().Get().ThemeID
}

// Theme returns the Theme with the provided ID, or this Domain's default Theme if
// the requested Theme does not exist.
func (w Common) Theme(themeID string) model.Theme {
	return w.factory().Theme().GetTheme(themeID)
}

// ThemeData returns a single custom value from this Domain's theme data.
// If the Domain has no value for the token, the Theme's declared default is used instead,
// and if the Theme does not declare the token either, it returns an empty string.
func (w Common) ThemeData(token string) string {

	// RULE: Read the Domain RECORD.  model.Theme.Data is a process-wide singleton shared
	// by every Domain on this server, and model.Domain.Data holds secrets (the VAPID
	// private key) that must never reach a page.
	domain := w.factory().Domain().Get()

	if value, exists := domain.ThemeData[token]; exists {
		return convert.String(value)
	}

	// RULE: Fall back to the SCHEMA default, not to the empty string.  A Domain begins with
	// no themeData keys at all, so a setting that has never been saved has no stored value --
	// and a settings toggle that is meant to start ON has to read as ON here too, or the page
	// and the form that configures it disagree until the owner's first save.  The Theme's
	// schema is the single place that default is declared; the form widget reads the same one.
	theme := w.Theme(domain.ThemeID)
	element, exists := theme.Schema.GetElement("themeData." + token)

	if !exists {
		return ""
	}

	return convert.String(element.DefaultValue())
}

// Now returns the current time in milliseconds since the Unix epoch
func (w Common) Now() int64 {
	return time.Now().Unix()
}

// NavigationID returns the the identifier of the top-most stream in the
// navigation.  The "common" builder just returns a default value that
// other builders should override.
func (w Common) NavigationID() string {
	return ""
}

/******************************************
 * Request Data (Getters and Setters)
 ******************************************/

// getArguments returns the arguments passed to this action. Implements the Builder interface.
func (w Common) getArguments() map[string]string {
	return w.arguments
}

// setArguments merges the provided arguments into this Builder's temporary data scope
func (w *Common) setArguments(arguments map[string]string) {
	for key, value := range arguments {
		w.arguments.SetString(key, value)
	}
}

// GetBool returns the named argument as a boolean. Implements the Builder interface.
func (w Common) GetBool(name string) bool {
	return convert.Bool(w.GetString(name))
}

// GetFloat returns the named argument as a float. Implements the Builder interface.
func (w Common) GetFloat(name string) float64 {
	return convert.Float(w.GetString(name))
}

// GetHTML returns the named argument as trusted HTML. Implements the Builder interface.
func (w Common) GetHTML(name string) template.HTML {
	return template.HTML(w.GetString(name))
}

// GetInt returns the named argument as an integer. Implements the Builder interface.
func (w Common) GetInt(name string) int {
	return convert.Int(w.GetString(name))
}

// GetInt64 returns the named argument as a 64-bit integer. Implements the Builder interface.
func (w Common) GetInt64(name string) int64 {
	return convert.Int64(w.GetString(name))
}

// GetString returns the named argument as a string. Implements the Builder interface.
func (w Common) GetString(name string) string {
	return w.arguments.GetString(name)
}

// setString writes a named argument into this Builder's temporary data scope
func (w Common) setString(name string, value string) {
	w.arguments.SetString(name, value)
}

// SetContent writes the "content" argument, which the next step in the pipeline renders
func (w Common) SetContent(value string) {
	w.setString("content", value)
}

// GetContent returns the "content" argument, as trusted HTML
func (w Common) GetContent() template.HTML {
	return w.GetHTML("content")
}

// IsIndexable returns TRUE if this page may be indexed by search engines.
// The default is TRUE; builders whose content can opt out of indexing (such as
// public user profiles) override this method. Shared page templates use it to
// decide whether to emit a "noindex" robots directive in the document <head>.
func (w Common) IsIndexable() bool {
	return true
}

/******************************************
 * Domain Data
 ******************************************/

// DomainStateID returns the lifecycle state of this Domain
func (w Common) DomainStateID() string {
	return w._factory.Domain().Get().StateID
}

// DomainLabel returns the human-readable name of this Domain
func (w Common) DomainLabel() string {
	return w._factory.Domain().Get().Label
}

// DomainIcon returns the URL of this Domain's icon image
func (w Common) DomainIcon() string {
	return w._factory.Domain().Get().IconURL()
}

// DomainImage returns the URL of this Domain's banner image
func (w Common) DomainImage() string {
	return w._factory.Domain().Get().ImageURL()
}

// DomainHasRegistrationForm returns TRUE if this Domain accepts new sign-ups
func (w Common) DomainHasRegistrationForm() bool {
	return w._factory.Domain().Get().HasRegistrationForm()
}

// IsDomainStartup returns TRUE if this Domain has not finished its first-run setup
func (w Common) IsDomainStartup() bool {
	return (w._factory.Domain().Get().StateID == model.DomainStateStartup)
}

// NotDomainStartup returns TRUE if this Domain has finished its first-run setup
func (w Common) NotDomainStartup() bool {
	return (w._factory.Domain().Get().StateID != model.DomainStateStartup)
}

/***************************
 * Access Permissions
 **************************/

// IsAuthenticated returns TRUE if the user is signed in
func (w Common) IsAuthenticated() bool {
	authorization := w.authorization()
	return authorization.IsAuthenticated()
}

// IsIdentity returns TRUE if the caller is signed in as a guest Identity
func (w Common) IsIdentity() bool {
	authorization := w.authorization()
	return authorization.IsIdentity()
}

// NotAuthenticatedOrIdentity returns TRUE if the caller is neither an authenticated user nor a guest identity
func (w Common) NotAuthenticatedOrIdentity() bool {

	if w.IsAuthenticated() {
		return false
	}

	if w.IsIdentity() {
		return false
	}

	return true
}

// IsOwner returns TRUE if the user is a Domain Owner
func (w Common) IsOwner() bool {
	authorization := w.authorization()
	return authorization.DomainOwner
}

// IsMasquerading returns TRUE if the user is a Domain Owner currently masquerading as another user
func (w Common) IsMasquerading() bool {
	authorization := w.authorization()
	return authorization.Masquerade
}

// IsAdminBuilder returns TRUE if the current builder is an Admin
// route.  By default, all other builders return FALSE.
func (w Common) IsAdminBuilder() bool {
	return false
}

// AuthenticatedID returns the unique ID of the currently logged in user (may be nil).
func (w Common) AuthenticatedID() primitive.ObjectID {
	authorization := w.authorization()
	return authorization.UserID
}

// UserName returns the DisplayName of the user
func (w Common) UserName() (string, error) {

	const location = "build.Stream.UserName"

	user, err := w.getUser()

	if err != nil {
		return "", derp.Wrap(err, location, "Loading User")
	}

	return user.DisplayName, nil
}

// UserImage returns the avatar image of the signed-in User
func (w Common) UserImage() (string, error) {

	const location = "build.Stream.UserImage"

	user, err := w.getUser()

	if err != nil {
		return "", derp.Wrap(err, location, "Loading User")
	}

	return user.ActivityPubIconURL(), nil
}

/******************************************
 * ActivityStreams / ActivityPub
 ******************************************/

// ActivityStream returns an ActivityStream document for the provided URL.  The
// returned document uses Emissary's custom ActivityStream service, which uses
// document values and rules from the server's shared cache.
func (w Common) ActivityStream(url string) streams.Document {

	const location = "build.Common.ActivityStream"

	// RULE: ?reveal=true is the D2 click-to-reveal override. It only lifts the REFUSAL of documents
	// the viewer's own rules hide -- the verdict is still stamped, so labels stay visible. It grants
	// nothing: an anonymous or other-user viewer has different rules, not fewer.
	options := make([]any, 0, 1)

	if w.QueryParam("reveal") == "true" {
		options = append(options, asrules.WithReveal(true))
	}

	// Load the document from the Interwebs. The asrules layer in this client stack stamps the
	// viewer's verdict into result.Metadata.Labels -- hides, annotations, and attribution alike.
	activityService := w._factory.ActivityStream()
	result, err := activityService.UserClient(w.AuthenticatedID()).Load(url, options...)

	if err != nil {

		// RULE: a rules refusal is the expected shape of a hidden document, not an error. The
		// refused document still carries the verdict, so templates render the D2 placeholder.
		if !derp.IsForbidden(err) {
			derp.Report(derp.Wrap(err, location, "Loading ActivityStream. Returning empty document to template."))
		}
	}

	// Return the result
	return result
}

// ActivityStreamCollection returns the item URLs in the remote ActivityPub collection at the provided URL
func (w Common) ActivityStreamCollection(url string) sliceof.String {

	const location = "build.Common.ActivityStreamCollection"

	// Load the collection from the Interwebs
	activityService := w._factory.ActivityStream()
	object, err := activityService.UserClient(w.AuthenticatedID()).Load(url, ascache.WithWriteOnly())

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading ActivityStream Collection. Returning empty collection to template."))
		return sliceof.NewString()
	}

	// Copy item IDs into the result slice
	result := sliceof.NewString()
	for item := range collections.RangeDocuments(object) {
		result = append(result, item.ID())
	}

	return result
}

// ActivityStreamActor returns an ActivityStream actor document for the provided URL.  The
// returned document uses Emissary's custom ActivityStream service, which uses
// document values and rules from the server's shared cache.
func (w Common) ActivityStreamActor(url string) streams.Document {
	activityService := w._factory.ActivityStream()
	result, err := activityService.UserClient(w.AuthenticatedID()).Load(url, sherlock.AsActor())

	if err != nil {
		derp.Report(err)
	}

	return result
}

// ActivityStreamActors returns the cached remote Actors that match the provided search text
func (w Common) ActivityStreamActors(search string) (sliceof.Object[model.ActorSummary], error) {
	activityService := w._factory.ActivityStream()
	return activityService.QueryActors(search)
}

// AmFollowing returns a Following record for the current user and the given URL
// If the user is not authenticated, or the URL is not valid, then an empty Following record is returned.
// The UX uses this to label "mutual" follows
func (w Common) AmFollowing(url string) model.Following {

	if !w._authorization.IsAuthenticated() {
		return model.NewFollowing()
	}

	// Get following service and new following record
	followingService := w._factory.Following()
	following := model.NewFollowing()

	// Retrieve following record. Discard errors
	// nolint:errcheck
	_ = followingService.LoadByURL(w._session, w._authorization.UserID, url, &following)

	// Return the (possibly empty) Following record
	return following
}

// IsFollower returns the signed-in User's Follower record for the provided URL, or an empty record
func (w Common) IsFollower(url string) model.Follower {

	followerService := w._factory.Follower()
	follower := model.NewFollower()

	_ = followerService.LoadByActor(w._session, w.AuthenticatedID(), url, &follower)
	return follower
}

/******************************************
 * Misc Helper Methods
 ******************************************/

// IsMobile returns TRUE if the request was made by a mobile device
func (w Common) IsMobile() bool {
	return sniff.IsMobile(w.request().UserAgent())
}

// IsDesktop returns TRUE if the request was made by a desktop device
func (w Common) IsDesktop() bool {
	return !sniff.IsMobile(w.request().UserAgent())
}

// IsLocalhost returns TRUE if the request was made to a local domain (localhost, 127.0.0.1, etc.)
func (w Common) IsLocalhost() bool {
	return uri.IsLocalHostname(w.Hostname())
}

// IsMe returns TRUE if the provided URI is the profileURL of the current user
func (w Common) IsMe(url string) bool {
	if user, err := w.getUser(); err == nil {
		return url == user.ActivityPubURL()
	}
	return false
}

// NotMe returns TRUE if the provided URI is NOT the ProfileURL of the current user
func (w Common) NotMe(url string) bool {
	return !w.IsMe(url)
}

// GetFollowingID returns the FollowingID that connects the signed-in User to the
// document at the provided URL (or to the actor who created it)
func (w Common) GetFollowingID(url string) string {

	const location = "build.Common.GetFollowingID"

	followingService := w._factory.Following()
	result, err := followingService.GetFollowingID(w._session, w.AuthenticatedID(), url)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Getting following status", url))
		return ""
	}

	return result
}

// lookupProvider returns the LookupProvider service, which can return form.LookupGroups
func (w Common) lookupProvider() form.LookupProvider {
	userID := w.AuthenticatedID()
	return w._factory.LookupProvider(w._request, w._session, userID)
}

// Dataset returns a single form.LookupGroup from the LookupProvider
func (w Common) Dataset(name string) form.LookupGroup {
	return w.lookupProvider().Group(name)
}

// DatasetValue returns a single form.LookupCode from the LookupProvider
func (w Common) DatasetValue(name string, value string) form.LookupCode {

	if dataset := w.Dataset(name); dataset != nil {
		for _, item := range dataset.Get() {
			if item.Value == value {
				return item
			}
		}
	}

	return form.LookupCode{}
}

// withinPublishDate returns the criteria fragment limiting a query to the current publish window. Implements the Builder interface.
func (w Common) withinPublishDate() exp.Expression {
	return exp.LessThan("publishDate", time.Now().Unix()).
		AndGreaterThan("unpublishDate", time.Now().Unix())
}

// defaultAllowed augments a query criteria to include the
// group authorizations of the currently signed in user.
func (w Common) defaultAllowed() exp.Expression {

	const location = "build.Common.defaultAllowed"

	var result exp.Expression = exp.Equal("deleteDate", 0) // Stream must not be deleted

	// If the user IS NOT a domain owner, then we must also
	// check their permission to VIEW this stream
	authorization := w.authorization()

	if authorization.DomainOwner {
		return result
	}

	// Fall through means this is a regular user, so standard permissions apply

	// Retrieve the Identity of the current website guest (if any)
	var identity *model.Identity
	identity, err := w.getIdentity()

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading Identity"))
	}

	// Get the access list for this user
	permissionService := w._factory.Permission()
	if permissions := permissionService.Permissions(&authorization, identity); permissions.NotZero() {
		result = result.AndIn("defaultAllow", permissions)
	}

	// Done.
	return result
}

// getUser loads/caches the currently-signed-in user to be used by other functions in this builder
func (w Common) getUser() (*model.User, error) {

	const location = "build.Common.getUser"

	// If we already have a cached User, then return that
	if w._user != nil {
		return w._user, nil
	}

	// Otherwise, try to load the User from the database
	userService := w._factory.User()
	steranko := w._factory.Steranko(w._session)
	authorization := getAuthorization(steranko, w._request)

	user := model.NewUser()
	if err := userService.LoadByID(w._session, authorization.UserID, &user); err != nil {
		return nil, derp.Wrap(err, location, "Loading user from database", authorization.UserID)
	}

	// Save the User in the builder to use it later
	w._user = &user

	// Return the User to the caller
	return w._user, nil
}

// getIdentity returns the currently signed-in guest Identity. Implements the Builder interface.
func (w Common) getIdentity() (*model.Identity, error) {

	const location = "build.Common.getIdentity"

	// If no Identity is provided, then return nil
	if !w._authorization.IsIdentity() {
		return nil, nil
	}

	// If Identity exists in the cache, then use it.
	if w._identity != nil {
		return w._identity, nil
	}

	// Otherwise, try to load the Identity from the database
	identity := model.NewIdentity()
	if err := w._factory.Identity().LoadByID(w._session, w._authorization.IdentityID, &identity); err != nil {
		return nil, derp.Wrap(err, location, "Loading Identity from database", w._authorization.IdentityID)
	}

	// Save the Identity in the builder to use it later
	w._identity = &identity

	// Return the Identity to the caller.
	return w._identity, nil
}

/******************************************
 * Common Queries
 ******************************************/

// Navigation returns an array of Streams that have a Zero ParentID
func (w Common) Navigation() (sliceof.Object[model.StreamSummary], error) {
	criteria := w.defaultAllowed().
		And(w.withinPublishDate()).
		AndEqual("parentId", primitive.NilObjectID)

	builder := NewQueryBuilder[model.StreamSummary](w._factory.Stream(), w._session, criteria)

	result, err := builder.Top60().ByRank().Slice()
	return result, err
}

// GetResponseID returns the ID of the signed-in User's response of the provided type to a URL
func (w Common) GetResponseID(responseType string, url string) string {

	// If the user is not signed in, then they can't have responded.
	if !w.IsAuthenticated() {
		return ""
	}

	if len(url) == 0 {
		return ""
	}

	// If the user is signed in, then we need to check the database to see if they've responded.
	responseService := w._factory.Response()
	response := model.NewResponse()

	if err := responseService.LoadByUserAndObject(w._session, w.AuthenticatedID(), url, responseType, &response); err == nil {
		return response.ResponseID.Hex()
	}

	return ""
}

// GetResponseSummary returns which responses the signed-in User has made to the provided URL
func (w Common) GetResponseSummary(url string) model.UserResponseSummary {

	result := model.NewUserResponseSummary()

	// If the user is not signed in, then they can't have responded.
	if !w.IsAuthenticated() {
		return result
	}

	if len(url) == 0 {
		return result
	}

	// If the user is signed in, then we need to check the database to see if they've responded.
	responseService := w._factory.Response()

	if responses, err := responseService.QueryByUserAndObject(w._session, w.AuthenticatedID(), url); err == nil {
		for _, response := range responses {
			result.SetResponse(response.Type, true)
		}
	}

	return result
}

// AvailableMerchantAccounts returns the payment providers this Domain has connected, as LookupCodes
func (w Common) AvailableMerchantAccounts() (sliceof.Object[form.LookupCode], error) {
	merchantAccountService := w._factory.MerchantAccount()
	return merchantAccountService.AvailableMerchantAccounts(w._session)
}

/******************************************
 * Search Engine
 ******************************************/

// Search returns a SearchBuilder seeded from this request's query string. Implements the Builder interface.
func (w Common) Search() SearchBuilder {

	// Collect required values
	searchTagService := w._factory.SearchTag()
	searchResultService := w._factory.SearchResult()
	textQuery := w.QueryParam("q")

	// Evaluate query string
	b := builder.NewBuilder().
		String("tags", builder.WithFilter(model.ToToken)).
		Time("date").
		Polygon("location")

	criteria := b.Evaluate(w._request.URL.Query())

	// Create the SearchBuilder for this request
	return NewSearchBuilder(searchTagService, searchResultService, w._factory.Rule(), w.AuthenticatedID(), w._session, criteria, textQuery)
}

// SearchTag returns the SearchTag with the provided name, or an empty tag if it does not exist
func (w Common) SearchTag(tagName string) model.SearchTag {

	const location = "build.Common.SearchTag"

	result := model.NewSearchTag()

	if err := w._factory.SearchTag().LoadByValue(w._session, tagName, &result); err != nil {
		derp.Report(derp.Wrap(err, location, "Loading SearchTag", tagName))
	}

	return result
}

// MapTiles returns the map-tile provider that this Domain draws maps with
func (w Common) MapTiles() form.LookupCode {
	return w.factory().GeocodeTiles().GetTileURL(w.session())
}

// MerchantAccount returns the MerchantAccount with the provided token
func (w Common) MerchantAccount(merchantAccountID string) (model.MerchantAccount, error) {
	result := model.NewMerchantAccount()
	err := w._factory.MerchantAccount().LoadByToken(w._session, merchantAccountID, &result)
	return result, err
}

// FeaturedSearchTags returns a QueryBuilder over this Domain's featured SearchTags
func (w Common) FeaturedSearchTags() *QueryBuilder[model.SearchTag] {

	criteria := exp.And(
		exp.Equal("stateId", model.SearchTagStateFeatured),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.SearchTag](w._factory.SearchTag(), w._session, criteria)
	result.CaseInsensitive()
	result.ByRank()

	return &result
}

// AllowedSearchTags returns a query builder for all SearchTags that are
// marked "Allowed" by the domain admin.
func (w Common) AllowedSearchTags() *QueryBuilder[model.SearchTag] {

	query := builder.NewBuilder().
		String("q", builder.WithAlias("value"), builder.WithDefaultOpContains(), builder.WithFilter(model.ToToken))

	criteria := exp.And(
		query.Evaluate(w._request.URL.Query()),
		exp.In("stateId", []int{model.SearchTagStateAllowed, model.SearchTagStateFeatured}),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.SearchTag](w._factory.SearchTag(), w._session, criteria)
	result.CaseInsensitive()
	result.ByName()

	return &result
}

/******************************************
 * Additional Data
 ******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (w Common) AdminSections() []form.LookupCode {
	return []form.LookupCode{
		{
			Value: "domain",
			Label: "General",
		},
		{
			Value: "navigation",
			Label: "Navigation",
		},
		{
			Value: "groups",
			Label: "Groups",
		},
		{
			Value: "users",
			Label: "Users",
		},
		{
			Value: "rules",
			Label: "Rules",
		},
		{
			Value: "tags",
			Label: "Tags",
		},
		{
			Value: "connections",
			Label: "Connections",
		},
		{
			Value: "webhooks",
			Label: "Webhooks",
		},
		{
			Value: "syndication",
			Label: "Syndication",
		},
	}
}
