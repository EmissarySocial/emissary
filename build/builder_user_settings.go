package build

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/form"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Settings is a builder for the @user/inbox page
type Settings struct {
	_user *model.User
	CommonWithTemplate
}

// NewSettings returns a fully initialized `Settings` builder
func NewSettings(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, user *model.User, actionID string) (Settings, error) {

	const location = "build.NewSettings"

	// Load the Template
	templateService := factory.Template()
	template, err := templateService.Load("user-settings") // TODO: Users should get to select their inbox template

	if err != nil {
		return Settings{}, derp.Wrap(err, location, "Unable to load template")
	}

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, user, actionID)

	if err != nil {
		return Settings{}, derp.Wrap(err, location, "Unable to create common builder")
	}

	return Settings{
		_user:              user,
		CommonWithTemplate: common,
	}, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Settings
func (w Settings) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		return "", derp.Wrap(status.Error, "build.Settings.Render", "Unable to generate HTML", w._request.URL.String())
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Settings
func (w Settings) View(actionID string) (template.HTML, error) {

	builder, err := NewSettings(w._factory, w._session, w._request, w._response, w._user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.Settings.View", "Unable to create Settings builder")
	}

	return builder.Render()
}

// NavigationID returns the ID to use for highlighing navigation menus
func (w Settings) NavigationID() string {
	return "settings"
}

func (w Settings) PageTitle() string {
	return "Settings"
}

func (w Settings) BasePath() string {
	return "/@me/settings"
}

func (w Settings) Permalink() string {
	return w.Host() + "/@me/settings"
}

func (w Settings) Token() string {
	return "users"
}

func (w Settings) object() data.Object {
	return w._user
}

func (w Settings) objectID() primitive.ObjectID {
	return w._user.UserID
}

func (w Settings) objectType() string {
	return "User"
}

func (w Settings) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

func (w Settings) service() service.ModelService {
	return w._factory.User()
}

func (w Settings) templateRole() string {
	return "inbox"
}

func (w Settings) clone(action string) (Builder, error) {
	return NewSettings(w._factory, w._session, w._request, w._response, w._user, action)
}

/******************************************
 * Data Accessors
 ******************************************/

func (w Settings) UserID() string {
	return w._user.UserID.Hex()
}

// Myself returns TRUE if the current user is viewing their own profile
func (w Settings) Myself() bool {
	return w._authorization.UserID == w._user.UserID
}

func (w Settings) Username() string {
	return w._user.Username
}

func (w Settings) FollowerCount() int {
	return w._user.FollowerCount
}

func (w Settings) FollowingCount() int {
	return w._user.FollowingCount
}

func (w Settings) RuleCount() int {
	return w._user.RuleCount
}

func (w Settings) DisplayName() string {
	return w._user.DisplayName
}

func (w Settings) ProfileURL() string {
	return w._user.ProfileURL
}

func (w Settings) IconURL() string {
	return w._user.ActivityPubIconURL()
}

/******************************************
 * Settings Methods
 ******************************************/

// Stream returns a stream object - if it is owned by the current user
func (w Settings) Stream(token string) (model.Stream, error) {

	// Load the stream from the database
	streamService := w._factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByToken(w._session, token, &stream); err != nil {
		return model.Stream{}, derp.Wrap(err, "build.Settings.Stream", "Unable to load stream", token)
	}

	// RULE: Stream must be owned by the current user
	if stream.AttributedTo.UserID != w._user.UserID {
		return model.Stream{}, derp.UnauthorizedError("build.Settings.Stream", "You do not have permission to view this stream")
	}

	// uWu
	return stream, nil
}

// Template returns the named Template object
func (w Settings) Template(templateID string) (model.Template, error) {
	templateService := w._factory.Template()
	return templateService.Load(templateID)
}

// Circles returns a QueryBuilder for Circles owned by the current user
func (w Settings) Circles() QueryBuilder[model.Circle] {

	// Define inbound parameters
	expressionBuilder := builder.NewBuilder().
		String("label")

	// Calculate criteria
	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w.AuthenticatedID()),
	)

	// Return the query builder
	return NewQueryBuilder[model.Circle](w._factory.Circle(), w._session, criteria)
}

func (w Settings) Followers() QueryBuilder[model.FollowerSummary] {

	// Define inbound parameters
	expressionBuilder := builder.NewBuilder().
		String("search", builder.WithAlias("actor.name"), builder.WithDefaultOpContains()).
		String("name", builder.WithAlias("actor.name"))

	// Calculate criteria
	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("parentId", w.AuthenticatedID()),
	)

	// Return the query builder
	return NewQueryBuilder[model.FollowerSummary](w._factory.Follower(), w._session, criteria)
}

func (w Settings) Following() QueryBuilder[model.FollowingSummary] {

	expressionBuilder := builder.NewBuilder().
		String("search", builder.WithAlias("label"), builder.WithDefaultOpContains()).
		String("label")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w.AuthenticatedID()),
	)

	return NewQueryBuilder[model.FollowingSummary](w._factory.Following(), w._session, criteria)
}

func (w Settings) FollowingByFolder(token string) ([]model.FollowingSummary, error) {

	// Get the UserID from the authentication scope
	userID := w.AuthenticatedID()

	if userID.IsZero() {
		return nil, derp.UnauthorizedError("build.Settings.FollowingByFolder", "Must be signed in to view following")
	}

	// Get the followingID from the token
	followingID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return nil, derp.Wrap(err, "build.Settings.FollowingByFolder", "Invalid following ID", token)
	}

	// Try to load the matching records
	followingService := w._factory.Following()
	return followingService.QueryByFolder(w._session, userID, followingID)
}

func (w Settings) FollowingByToken(followingToken string) (model.Following, error) {

	userID := w.AuthenticatedID()

	followingService := w._factory.Following()

	following := model.NewFollowing()

	if err := followingService.LoadByToken(w._session, userID, followingToken, &following); err != nil {
		return model.Following{}, derp.Wrap(err, "build.Settings.FollowingByID", "Unable to load following")
	}

	return following, nil
}

func (w Settings) OAuthClients() (sliceof.Object[model.OAuthClient], error) {
	userID := w.AuthenticatedID()
	oauthClientService := w._factory.OAuthClient()
	return oauthClientService.QueryByUserID(w._session, userID)
}

func (w Settings) OAuthUserTokens() (sliceof.MapOfAny, error) {
	userID := w.AuthenticatedID()
	return queries.OAuthUserTokens(w._session, userID)
}

func (w Settings) Rules() QueryBuilder[model.Rule] {

	expressionBuilder := builder.NewBuilder().
		String("search", builder.WithAlias("trigger"), builder.WithDefaultOpContains()).
		String("trigger")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w.AuthenticatedID()),
	)

	result := NewQueryBuilder[model.Rule](w._factory.Rule(), w._session, criteria)

	return result
}

func (w Settings) RuleByToken(token string) model.Rule {
	ruleService := w._factory.Rule()
	rule := model.NewRule()

	if err := ruleService.LoadByToken(w._session, w.AuthenticatedID(), token, &rule); err != nil {
		derp.Report(derp.Wrap(err, "build.Settings.RuleByToken", "Unable to load rule", token))
	}

	return rule
}

func (w Settings) Imports() (sliceof.Object[model.Import], error) {
	importService := w._factory.Import()
	return importService.QueryByUser(w._session, w.AuthenticatedID())
}

// ImportPlan generates an import plan for a given actor.
func (w Settings) ImportPlan(actor streams.Document) sliceof.Object[form.LookupCode] {
	return w.factory().Import().CalcImportPlan(actor)
}

func (w Settings) Privileges() QueryBuilder[model.Privilege] {

	expressionBuilder := builder.NewBuilder().
		String("search", builder.WithAlias("emailAddress"), builder.WithDefaultOpBeginsWith())

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w._user.UserID),
	)

	return NewQueryBuilder[model.Privilege](w._factory.Privilege(), w._session, criteria)
}

func (w Settings) MerchantAccounts() QueryBuilder[model.MerchantAccount] {

	expressionBuilder := builder.NewBuilder().
		String("search", builder.WithAlias("name"), builder.WithDefaultOpBeginsWith())

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w._user.UserID),
	)

	return NewQueryBuilder[model.MerchantAccount](w._factory.MerchantAccount(), w._session, criteria)
}

func (w Settings) MerchantAccount(merchantAccountID string) (model.MerchantAccount, error) {
	result := model.NewMerchantAccount()
	err := w._factory.MerchantAccount().LoadByUserAndToken(w._session, w._user.UserID, merchantAccountID, &result)
	return result, err
}

// RemoteProducts syncs the remote products from all MerchantAccounts and returns the full Product catalog.
func (w Settings) RemoteProducts() (sliceof.Object[model.Product], error) {

	_, remoteProducts, err := w._factory.Product().SyncRemoteProducts(w._session, w._user.UserID)

	if err != nil {
		return nil, derp.Wrap(err, "build.Common.Products", "Unable to load products for user", w._user.UserID)
	}

	return remoteProducts, nil
}

// FilteredByFollowing returns the Following record that is being used to filter the Settings
func (w Settings) FilteredByFollowing() model.Following {

	result := model.NewFollowing()

	if !w.IsAuthenticated() {
		return result
	}

	token := w._request.URL.Query().Get("origin.followingId")

	if followingID, err := primitive.ObjectIDFromHex(token); err == nil {
		followingService := w._factory.Following()

		if err := followingService.LoadByID(w._session, w.AuthenticatedID(), followingID, &result); err == nil {
			return result
		}
	}

	return result
}

// SubBuilder creates a new builder for a child object.  This function works
// with Rule, Folder, Follower, Following, and Stream objects.  It will return
// an error if the object is not one of those types.
func (w Settings) SubBuilder(object any) (Builder, error) {

	var result Builder
	var err error

	switch typed := object.(type) {

	case model.Rule:
		result, err = NewModel(w._factory, w._session, w._request, w._response, w._template, &typed, w._actionID)

	case model.Folder:
		result, err = NewModel(w._factory, w._session, w._request, w._response, w._template, &typed, w._actionID)

	case model.Follower:
		result, err = NewFollower(w._factory, w._session, w._request, w._response, w._template, &typed, w._actionID)

	case model.Following:
		result, err = NewModel(w._factory, w._session, w._request, w._response, w._template, &typed, w._actionID)

	case model.Stream:
		result, err = NewStream(w._factory, w._session, w._request, w._response, w._template, &typed, w._actionID)

	default:
		result, err = nil, derp.InternalError("build.Common.SubBuilder", "Invalid object type", object)
	}

	if err != nil {
		err = derp.Wrap(err, "build.Common.SubBuilder", "Unable to create sub-builder for object", object)
		derp.Report(err)
	}

	return result, err
}

func (w Settings) AmFollowing(url string) model.Following {

	// Get following service and new following record
	followingService := w._factory.Following()
	following := model.NewFollowing()

	// Null check
	if w._user == nil {
		return following
	}

	// Retrieve following record. Discard errors
	// nolint:errcheck
	_ = followingService.LoadByURL(w._session, w._user.UserID, url, &following)

	// Return the (possibly empty) Following record
	return following
}

// HasRule returns a rule that matches the current user, rule type, and trigger.
// If no rule is found, then an empty rule is returned.
func (w Settings) HasRule(ruleType string, trigger string) model.Rule {

	// Get following service and new following record
	ruleService := w._factory.Rule()
	rule := model.NewRule()

	// Null check
	if w._user == nil {
		return rule
	}

	// Retrieve rule record.  "Not Found" is acceptable, but "legitimate" errors are not.
	// In either case, do not halt the request
	if err := ruleService.LoadByTrigger(w._session, w._user.UserID, ruleType, trigger, &rule); err != nil {
		if !derp.IsNotFound(err) {
			derp.Report(derp.Wrap(err, "build.Settings.HasRule", "Unable to load rule", ruleType, trigger))
		}
	}

	// Return the (possibly empty) Rule record
	return rule
}

func (w Settings) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Settings")
}
