package build

import (
	"html/template"
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/form"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/sherlock"
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

	// Cached values, do not populate unless needed
	domain model.Domain // This is a value because we expect to use it in every request.
}

func NewCommon(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter) Common {

	// Retrieve the user's authorization information
	steranko := factory.Steranko()
	authorization := getAuthorization(steranko, request)

	// Return a new Common builder
	return Common{
		_factory:       factory,
		_session:       session,
		_request:       request,
		_response:      response,
		_authorization: authorization,
		arguments:      make(mapof.String),
		domain:         model.NewDomain(),
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

func (w Common) PageTitle() string {
	return ""
}

func (w Common) Summary() string {
	return ""
}

/******************************************
 * Request Info
 ******************************************/

// Returns the request method
func (w Common) Method() string {
	return w._request.Method
}

// Host returns the protocol + the Hostname
func (w Common) Host() string {
	return w.Protocol() + w.Hostname()
}

// URL returns the originally requested URL
func (w Common) URL() string {
	return w.Host() + w._request.URL.RequestURI()
}

// Protocol returns http:// or https:// used for this request
func (w Common) Protocol() string {
	return dt.Protocol(w.Hostname())
}

// Hostname returns the configured hostname for this request
func (w Common) Hostname() string {
	return dt.Hostname(w._request)
}

// Path returns the HTTP Request path
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

// Returns the designated request parameter.  If there are
// multiple values for the parameter, then only the first
// value is returned.
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

func (w Common) getArguments() map[string]string {
	return w.arguments
}

func (w *Common) setArguments(arguments map[string]string) {
	for key, value := range arguments {
		w.arguments.SetString(key, value)
	}
}

func (w Common) GetBool(name string) bool {
	return convert.Bool(w.GetString(name))
}

func (w Common) GetFloat(name string) float64 {
	return convert.Float(w.GetString(name))
}

func (w Common) GetHTML(name string) template.HTML {
	return template.HTML(w.GetString(name))
}

func (w Common) GetInt(name string) int {
	return convert.Int(w.GetString(name))
}

func (w Common) GetInt64(name string) int64 {
	return convert.Int64(w.GetString(name))
}

func (w Common) GetString(name string) string {
	return w.arguments.GetString(name)
}

func (w Common) setString(name string, value string) {
	w.arguments.SetString(name, value)
}

func (w Common) SetContent(value string) {
	w.setString("content", value)
}

func (w Common) GetContent() template.HTML {
	return w.GetHTML("content")
}

/******************************************
 * Domain Data
 ******************************************/

func (w Common) DomainLabel() string {
	return w._factory.Domain().Get().Label
}

func (w Common) DomainHasRegistrationForm() bool {
	return w._factory.Domain().Get().HasRegistrationForm()
}

/***************************
 * Access Permissions
 **************************/

// IsAuthenticated returns TRUE if the user is signed in
func (w Common) IsAuthenticated() bool {
	authorization := w.authorization()
	return authorization.IsAuthenticated()
}

func (w Common) IsIdentity() bool {
	authorization := w.authorization()
	return authorization.IsIdentity()
}

// IsOwner returns TRUE if the user is a Domain Owner
func (w Common) IsOwner() bool {
	authorization := w.authorization()
	return authorization.DomainOwner
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
		return "", derp.Wrap(err, location, "Error loading User")
	}

	return user.DisplayName, nil
}

// UserAvatar returns the avatar image of the user
func (w Common) UserImage() (string, error) {

	const location = "build.Stream.UserImage"

	user, err := w.getUser()

	if err != nil {
		return "", derp.Wrap(err, location, "Error loading User")
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
	// Load the document from the Interwebs
	result, err := w._factory.ActivityStream().Load(url)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading ActivityStream"))
	}

	// Search for rules that might add a LABEL to this document.
	ruleService := w._factory.Rule()
	filter := ruleService.Filter(w.AuthenticatedID(), service.WithLabelsOnly())
	filter.Allow(w._session, &result)

	// Return the result
	return result
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

func (w Common) IsFollower(url string) model.Follower {

	followerService := w._factory.Follower()
	follower := model.NewFollower()

	_ = followerService.LoadByActor(w._session, w.AuthenticatedID(), url, &follower)
	return follower
}

// ActivityStreamActor returns an ActivityStream actor document for the provided URL.  The
// returned document uses Emissary's custom ActivityStream service, which uses
// document values and rules from the server's shared cache.
func (w Common) ActivityStreamActor(url string) streams.Document {
	result, err := w._factory.ActivityStream().Load(url, sherlock.AsActor())

	if err != nil {
		derp.Report(err)
	}

	return result
}

func (w Common) ActivityStreamActors(search string) ([]model.ActorSummary, error) {
	return w._factory.ActivityStream().SearchActors(w._session, search)
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

/******************************************
 * Misc Helper Methods
 ******************************************/

// IsFollowing returns TRUE if the curren user is following the
// document at a specific URI (or the actor who created the document)
func (w Common) GetFollowingID(url string) string {

	const location = "build.Common.GetFollowingID"

	followingService := w._factory.Following()
	result, err := followingService.GetFollowingID(w._session, w.AuthenticatedID(), url)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error getting following status", url))
		return ""
	}

	return result
}

// lookupProvider returns the LookupProvider service, which can return form.LookupGroups
func (w Common) lookupProvider() form.LookupProvider {
	userID := w.AuthenticatedID()
	return w._factory.LookupProvider(w._request, userID)
}

// Dataset returns a single form.LookupGroup from the LookupProvider
func (w Common) Dataset(name string) form.LookupGroup {
	return w.lookupProvider().Group(name)
}

// DatasetValue returns a single form.LookupCode from the LookupProvider
func (w Common) DatasetValue(name string, value string) form.LookupCode {

	dataset := w.Dataset(name)

	if dataset != nil {
		for _, item := range dataset.Get() {
			if item.Value == value {
				return item
			}
		}
	}

	return form.LookupCode{}
}

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
		derp.Report(derp.Wrap(err, location, "Error loading Identity"))
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
	steranko := w._factory.Steranko()
	authorization := getAuthorization(steranko, w._request)

	user := model.NewUser()
	if err := userService.LoadByID(w._session, authorization.UserID, &user); err != nil {
		return nil, derp.Wrap(err, location, "Error loading user from database", authorization.UserID)
	}

	// Save the User in the builder to use it later
	w._user = &user

	// Return the User to the caller
	return w._user, nil
}

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
		return nil, derp.Wrap(err, location, "Error loading Identity from database", w._authorization.IdentityID)
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

func (w Common) AvailableMerchantAccounts() (sliceof.Object[form.LookupCode], error) {
	merchantAccountService := w._factory.MerchantAccount()
	return merchantAccountService.AvailableMerchantAccounts(w._session)
}

/******************************************
 * Search Engine
 ******************************************/

func (w Common) Search() SearchBuilder {

	// Collect required values
	searchTagService := w._factory.SearchTag()
	searchResultService := w._factory.SearchResult()
	textQuery := w.QueryParam("q")

	// Evaluate query string
	b := builder.NewBuilder().
		String("tags", builder.WithFilter(model.ToToken)).
		Time("date").
		Location("place")

	criteria := b.Evaluate(w._request.URL.Query())

	// Create the SearchBuilder for this request
	return NewSearchBuilder(searchTagService, searchResultService, w._session, criteria, textQuery)
}

func (w Common) SearchTag(tagName string) model.SearchTag {

	const location = "build.Common.SearchTag"

	result := model.NewSearchTag()

	if err := w._factory.SearchTag().LoadByValue(w._session, tagName, &result); err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading SearchTag", tagName))
	}

	return result
}

func (w Common) MerchantAccount(merchantAccountID string) (model.MerchantAccount, error) {
	result := model.NewMerchantAccount()
	err := w._factory.MerchantAccount().LoadByToken(w._session, merchantAccountID, &result)
	return result, err
}

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
