package build

import (
	"html/template"
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
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
	_request       *http.Request       // Pointer to the HTTP request we are serving
	_response      http.ResponseWriter // ResponseWriter for this request
	_authorization model.Authorization // Authorization information for the current user

	arguments mapof.String // Temporary data scope for this request

	// Cached values, do not populate unless needed
	domain model.Domain // This is a value because we expect to use it in every request.
}

func NewCommon(factory Factory, request *http.Request, response http.ResponseWriter) Common {

	// Retrieve the user's authorization information
	steranko := factory.Steranko()
	authorization := getAuthorization(steranko, request)

	// Return a new Common builder
	return Common{
		_factory:       factory,
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

func (w Common) request() *http.Request {
	return w._request
}

func (w Common) response() http.ResponseWriter {
	return w._response
}

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
	return domain.Protocol(w.Hostname())
}

// Hostname returns the configured hostname for this request
func (w Common) Hostname() string {
	return w._request.Host
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
func (w Common) SetQueryParam(name string, value string) {
	query := w._request.URL.Query()
	query.Set(name, value)
	w._request.URL.RawQuery = query.Encode()
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

func (w Common) DomainLabel() (string, error) {
	if domain, err := w.getDomain(); err != nil {
		return "", err
	} else {
		return domain.Label, nil
	}
}

func (w Common) DomainHasRegistrationForm() (bool, error) {
	if domain, err := w.getDomain(); err != nil {
		return false, err
	} else {
		return domain.HasRegistrationForm(), nil
	}
}

/***************************
 * Access Permissions
 **************************/

// IsAuthenticated returns TRUE if the user is signed in
func (w Common) IsAuthenticated() bool {
	authorization := w.authorization()
	return authorization.IsAuthenticated()
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
	user, err := w.getUser()

	if err != nil {
		return "", derp.Wrap(err, "build.Stream.UserName", "Error loading User")
	}

	return user.DisplayName, nil
}

// UserAvatar returns the avatar image of the user
func (w Common) UserImage() (string, error) {
	user, err := w.getUser()

	if err != nil {
		return "", derp.Wrap(err, "build.Stream.UserAvatar", "Error loading User")
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

	// Load the document from the Interwebs
	result, err := w._factory.ActivityStream().Load(url)

	if err != nil {
		derp.Report(derp.Wrap(err, "build.Common.ActivityStream", "Error loading ActivityStream"))
	}

	// Search for rules that might add a LABEL to this document.
	ruleService := w._factory.Rule()
	filter := ruleService.Filter(w.AuthenticatedID(), service.WithLabelsOnly())
	filter.Allow(&result)

	// Return the result
	return result
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
	return w._factory.ActivityStream().SearchActors(search)
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

	followingService := w._factory.Following()
	result, err := followingService.GetFollowingID(w.AuthenticatedID(), url)

	if err != nil {
		derp.Report(derp.Wrap(err, "build.Common.GetFollowingID", "Error getting following status", url))
		return ""
	}

	return result
}

// lookupProvider returns the LookupProvider service, which can return form.LookupGroups
func (w Common) lookupProvider() form.LookupProvider {

	userID := w.AuthenticatedID()
	return w._factory.LookupProvider(userID)
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

// withViewPermission augments a query criteria to include the
// group authorizations of the currently signed in user.
func (w Common) withViewPermission(criteria exp.Expression) exp.Expression {

	result := criteria.
		And(exp.Equal("deleteDate", 0)) // Stream must not be deleted

	// If the user IS NOT a domain owner, then we must also
	// check their permission to VIEW this stream
	authorization := w.authorization()

	if !authorization.DomainOwner {
		result = result.And(exp.In("defaultAllow", authorization.AllGroupIDs())).
			And(exp.LessThan("publishDate", time.Now().Unix())) // Stream must be published
	}

	return result
}

// getUser loads/caches the currently-signed-in user to be used by other functions in this builder
func (w Common) getUser() (model.User, error) {

	userService := w._factory.User()
	steranko := w._factory.Steranko()
	result := model.NewUser()
	authorization := getAuthorization(steranko, w._request)

	if err := userService.LoadByID(authorization.UserID, &result); err != nil {
		return model.User{}, derp.Wrap(err, "build.Common.getUser", "Error loading user from database", authorization.UserID)
	}

	return result, nil
}

// getDomain retrieves the current domain model object from the domain service cache
func (w *Common) getDomain() (model.Domain, error) {

	domainService := w._factory.Domain()

	if !domainService.Ready() {
		return model.Domain{}, derp.NewInternalError("build.Common.getDomain", "Domain service not ready", nil)
	}

	return domainService.Get(), nil
}

/******************************************
 * Common Queries
 ******************************************/

// Navigation returns an array of Streams that have a Zero ParentID
func (w Common) Navigation() (sliceof.Object[model.StreamSummary], error) {
	criteria := w.withViewPermission(exp.Equal("parentId", primitive.NilObjectID))
	builder := NewQueryBuilder[model.StreamSummary](w._factory.Stream(), criteria)

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

	if err := responseService.LoadByUserAndObject(w.AuthenticatedID(), url, responseType, &response); err == nil {
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

	if responses, err := responseService.QueryByUserAndObject(w.AuthenticatedID(), url); err == nil {
		for _, response := range responses {
			result.SetResponse(response.Type, true)
		}
	}

	return result
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
	return NewSearchBuilder(searchTagService, searchResultService, criteria, textQuery)
}

func (w Common) SearchTag(tagName string) model.SearchTag {

	result := model.NewSearchTag()

	if err := w._factory.SearchTag().LoadByValue(tagName, &result); err != nil {
		derp.Report(derp.Wrap(err, "build.Common.SearchTag", "Error loading SearchTag", tagName))
	}

	return result
}

func (w Common) FeaturedSearchTags() *QueryBuilder[model.SearchTag] {

	criteria := exp.And(
		exp.Equal("stateId", model.SearchTagStateFeatured),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.SearchTag](w._factory.SearchTag(), criteria)
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

	result := NewQueryBuilder[model.SearchTag](w._factory.SearchTag(), criteria)
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
