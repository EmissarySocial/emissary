package build

import (
	"bytes"
	"html/template"
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Inbox is a builder for the @user/inbox page
type Inbox struct {
	_user *model.User
	CommonWithTemplate
}

// NewInbox returns a fully initialized `Inbox` builder
func NewInbox(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, user *model.User, actionID string) (Inbox, error) {

	const location = "build.NewInbox"

	// Load the Template
	templateService := factory.Template()
	template, err := templateService.Load(user.InboxTemplate)

	if err != nil {
		return Inbox{}, derp.Wrap(err, location, "Loading template")
	}

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, user, actionID)

	if err != nil {
		return Inbox{}, derp.Wrap(err, location, "Creating common builder")
	}

	// Enforce user permissions on the requested action
	if !common.UserCan(actionID) {
		if common._authorization.IsAuthenticated() {
			return Inbox{}, derp.Forbidden(location, "Forbidden", "User is authenticated, but this action is not allowed", actionID)
		} else {
			return Inbox{}, derp.Unauthorized(location, "Anonymous user is not authorized to perform this action", user.ProfileURL, actionID)
		}
	}

	return Inbox{
		_user:              user,
		CommonWithTemplate: common,
	}, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Inbox
func (w Inbox) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		return "", derp.Wrap(status.Error, "build.NewsFeed.Render", "Generating HTML", w._request.URL.String())
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Inbox
func (w Inbox) View(actionID string) (template.HTML, error) {

	builder, err := NewInbox(w._factory, w._session, w._request, w._response, w._user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.NewsFeed.View", "Creating Inbox builder")
	}

	return builder.Render()
}

// NavigationID returns the ID to use for highlighing navigation menus
func (w Inbox) NavigationID() string {
	return "inbox"
}

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w Inbox) PageTitle() string {
	return w._user.DisplayName
}

// BasePath returns the URL path of this object, without any action. Implements the Builder interface.
func (w Inbox) BasePath() string {
	return "/@me/newsfeed"
}

// Permalink returns the permanent URL of the record being built. Implements the Builder interface.
func (w Inbox) Permalink() string {

	if newsItem := w.Message(); !newsItem.IsNew() {
		return newsItem.URL
	}

	if url := w._request.URL.Query().Get("url"); url != "" {
		return url
	}

	return w.Host() + "/@me/newsfeed"
}

// Token returns the URL token of the record being built. Implements the Builder interface.
func (w Inbox) Token() string {
	return "users"
}

// object returns the model object being built. Implements the Builder interface.
func (w Inbox) object() data.Object {
	return w._user
}

// objectID returns the unique ID of the object being built. Implements the Builder interface.
func (w Inbox) objectID() primitive.ObjectID {
	return w._user.UserID
}

// objectType returns the name of the model type being built. Implements the Builder interface.
func (w Inbox) objectType() string {
	return "User"
}

// schema returns the rosetta schema that validates this object. Implements the Builder interface.
func (w Inbox) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

// service returns the ModelService that backs this Builder. Implements the Builder interface.
func (w Inbox) service() service.ModelService {
	return w._factory.User()
}

// templateRole returns the role this Template plays, which chooses the child template. Implements the Builder interface.
func (w Inbox) templateRole() string {
	return "inbox"
}

// clone returns a copy of this Builder, pointed at a different action. Implements the Builder interface.
func (w Inbox) clone(action string) (Builder, error) {
	return NewInbox(w._factory, w._session, w._request, w._response, w._user, action)
}

/******************************************
 * Data Accessors
 ******************************************/

// UserID returns the unique ID of the User whose inbox is being built, as a string
func (w Inbox) UserID() string {
	return w._user.UserID.Hex()
}

// ActorURL returns the ActivityPub URL of the User whose inbox is being built
func (w Inbox) ActorURL() string {
	return w.Host() + "/@" + w._user.UserID.Hex()
}

// Myself returns TRUE if the current user is viewing their own profile
func (w Inbox) Myself() bool {
	return w._authorization.UserID == w._user.UserID
}

// Username returns the username of the Inbox being built
func (w Inbox) Username() string {
	return w._user.Username
}

// FollowerCount returns the follower count of the Inbox being built
func (w Inbox) FollowerCount() int {
	return w._user.FollowerCount
}

// FollowingCount returns the following count of the Inbox being built
func (w Inbox) FollowingCount() int {
	return w._user.FollowingCount
}

// RuleCount returns the rule count of the Inbox being built
func (w Inbox) RuleCount() int {
	return w._user.RuleCount
}

// DisplayName returns the display name of the Inbox being built
func (w Inbox) DisplayName() string {
	return w._user.DisplayName
}

// ProfileURL returns the profile URL of the Inbox being built
func (w Inbox) ProfileURL() string {
	return w._user.ProfileURL
}

// IconURL returns the icon URL of the Inbox being built
func (w Inbox) IconURL() string {
	return w._user.ActivityPubIconURL()
}

/******************************************
 * Inbox Methods
 ******************************************/

// Followers returns a QueryBuilder that lists the signed-in User's Followers
func (w Inbox) Followers() QueryBuilder[model.FollowerSummary] {

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

// Following returns a QueryBuilder that lists the actors the signed-in User follows
func (w Inbox) Following() QueryBuilder[model.FollowingSummary] {

	expressionBuilder := builder.NewBuilder().
		String("search", builder.WithAlias("label"), builder.WithDefaultOpContains()).
		String("label")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w.AuthenticatedID()),
	)

	return NewQueryBuilder[model.FollowingSummary](w._factory.Following(), w._session, criteria)
}

// FollowingByFolder returns every Following that the signed-in User routes into the named Folder
func (w Inbox) FollowingByFolder(token string) ([]model.FollowingSummary, error) {

	// Get the UserID from the authentication scope
	userID := w.AuthenticatedID()

	if userID.IsZero() {
		return nil, derp.Unauthorized("build.NewsFeed.FollowingByFolder", "Must be signed in to view following")
	}

	// Get the followingID from the token
	followingID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return nil, derp.Wrap(err, "build.NewsFeed.FollowingByFolder", "Invalid following ID", token)
	}

	// Try to load the matching records
	followingService := w._factory.Following()
	return followingService.QueryByFolder(w._session, userID, followingID)
}

// FollowingByToken returns the signed-in User's Following record with the provided token
func (w Inbox) FollowingByToken(followingToken string) (model.Following, error) {

	userID := w.AuthenticatedID()

	followingService := w._factory.Following()

	following := model.NewFollowing()

	if err := followingService.LoadByToken(w._session, userID, followingToken, &following); err != nil {
		return model.Following{}, derp.Wrap(err, "build.NewsFeed.FollowingByID", "Loading following")
	}

	return following, nil
}

// Rules returns a QueryBuilder that lists the signed-in User's Rules
func (w Inbox) Rules() QueryBuilder[model.Rule] {

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

// RuleByToken returns the signed-in User's Rule with the provided token, or an empty Rule
func (w Inbox) RuleByToken(token string) model.Rule {
	ruleService := w._factory.Rule()
	rule := model.NewRule()

	if err := ruleService.LoadByToken(w._session, w.AuthenticatedID(), token, &rule); err != nil {
		derp.Report(derp.Wrap(err, "build.NewsFeed.RuleByToken", "Loading rule", token))
	}

	return rule
}

// Inbox returns a QueryBuilder for current User's inbox
func (w Inbox) Inbox() (QueryBuilder[model.NewsItem], error) {

	const location = "build.NewsFeed.Inbox"

	// Must be signed in to view inbox
	if w.AuthenticatedID().IsZero() {
		return QueryBuilder[model.NewsItem]{}, derp.Unauthorized(location, "Must be signed in to view inbox")
	}

	queryString := w._request.URL.Query()

	expBuilder := builder.NewBuilder().
		ObjectID("origin.followingId").
		ObjectID("followingId", builder.WithAlias("origin.followingId")).
		Int("rank").
		Int("readDate").
		Int("createDate")

	var criteria exp.Expression = exp.And(
		exp.Equal("userId", w.AuthenticatedID()),
		exp.Equal("deleteDate", 0),
		expBuilder.Evaluate(queryString),
	)

	// If we have a NON-ZERO folderID, then include it in the criteria
	if folderID, err := primitive.ObjectIDFromHex(queryString.Get("folderId")); err == nil {

		if !folderID.IsZero() {
			criteria = criteria.AndEqual("folderId", folderID)
		}
	}

	return NewQueryBuilder[model.NewsItem](w._factory.NewsFeed(), w._session, criteria), nil
}

// IsInboxEmpty returns TRUE if the inbox has no results and there are no filters applied
// This corresponds to there being NOTHING in the inbox, instead of just being filtered out.
func (w Inbox) IsInboxEmpty(inbox []model.NewsItem) bool {

	if len(inbox) > 0 {
		return false
	}

	if w._request.URL.Query().Get("rank") != "" {
		return false
	}

	return true
}

// FilteredByFollowing returns the Following record that is being used to filter the Inbox
func (w Inbox) FilteredByFollowing() model.Following {

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

// Folders returns a slice of all folders owned by the current User
func (w Inbox) Folders() (model.FolderList, error) {

	result := model.NewFolderList()

	// User must be authenticated to view any folders
	if !w.IsAuthenticated() {
		return result, derp.Forbidden("build.NewsFeed.Folders", "Not authenticated")
	}

	folderService := w._factory.Folder()
	folders, err := folderService.QueryByUserID(w._session, w.AuthenticatedID())

	if err != nil {
		return result, derp.Wrap(err, "build.NewsFeed.Folders", "Loading folders")
	}

	result.Folders = folders
	result.HasUnreadNotifications = w.HasUnreadNotifications()
	return result, nil
}

// FoldersWithSelection returns the signed-in User's Folder list, with the named section marked as selected
func (w Inbox) FoldersWithSelection(section string) (model.FolderList, error) {

	const location = "build.NewsFeed.FoldersWithSelection"

	// Get Folder List
	result, err := w.Folders()

	if err != nil {
		return result, derp.Wrap(err, location, "Loading folders")
	}

	// If the "Conversations" section is selected, then we are done.
	if section == model.FolderListSectionConversations {
		result.Section = section
		return result, nil
	}

	// If the "Notifications" section is selected, then we are done.
	if section == model.FolderListSectionNotifications {
		result.Section = section
		return result, nil
	}

	// Otherwise, we are in the "Folder" section
	result.Section = model.FolderListSectionFolder

	// If we don't have any folders, then we MUST be looking at
	// the "All Folders" view.
	if result.Folders.IsEmpty() {
		result.SelectedID = primitive.NilObjectID
		return result, nil
	}

	// Find/Mark the Selected FolderID.  A missing or invalid folderId EXPLICITLY
	// selects the synthetic "News Feed" (all folders) view — never an arbitrary
	// folder.  (Requests that lose the query string, like sidebar refetches, must
	// not silently jump the selection to the first folder.)
	token := w._request.URL.Query().Get("folderId")

	if folderID, err := primitive.ObjectIDFromHex(token); err == nil {
		result.SelectedID = folderID
	} else {
		result.SelectedID = primitive.NilObjectID
	}

	// Update the query string to reflect the selected folder
	w.SetQueryParam("folderId", result.SelectedID.Hex())

	// Return the result
	return result, nil
}

// SubBuilder creates a new builder for a child object.  This function works
// with Rule, Folder, Follower, Following, and Stream objects.  It will return
// an error if the object is not one of those types.
func (w Inbox) SubBuilder(object any) (Builder, error) {

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
		result, err = nil, derp.Internal("build.Common.SubBuilder", "Invalid object type", object)
	}

	if err != nil {
		err = derp.Wrap(err, "build.Common.SubBuilder", "Creating sub-builder for object", object)
		derp.Report(err)
	}

	return result, err
}

/******************************************
 * Message Methods
 ******************************************/

// Message uses the queryString ?newsItemId= parameter to load a Message from the database
// If the newsItemId parameter does not exist, is malformed, or if the newsItem does not exist, then
// a new, empty Message is returned.
// In addition, if there is a "sibling" URL parameter (either "next" or "prev") then the next/previous
// newsItem is loaded instead.
func (w Inbox) Message() model.NewsItem {

	const location = "build.NewsFeed.Message"

	// Get the newsItemID from the query string
	newsItemID, err := primitive.ObjectIDFromHex(w._request.URL.Query().Get("newsItemId"))

	if err != nil {
		return model.NewNewsItem()
	}

	// Load the newsItem from the database
	inboxService := w._factory.NewsFeed()
	newsItem := model.NewNewsItem()

	if err := inboxService.LoadByID(w._session, w.AuthenticatedID(), newsItemID, &newsItem); err != nil {
		derp.Report(derp.Wrap(err, location, "Loading newsItem", newsItemID))
		return model.NewNewsItem()
	}

	// If sibling (prev/next) is specified, then try to look that up before returning.
	if sibling := w._request.URL.Query().Get("sibling"); sibling != "" {

		// Otherwise, look up the next/previous newsItem
		criteria := exp.Equal("userId", w.AuthenticatedID()).AndEqual("folderId", newsItem.FolderID)
		options := []option.Option{option.MaxRows(1)}

		if sibling == "next" {
			criteria = criteria.And(exp.GreaterThan("rank", newsItem.Rank))
			options = append(options, option.SortAsc("rank"))
		} else {
			criteria = criteria.And(exp.LessThan("rank", newsItem.Rank))
			options = append(options, option.SortDesc("rank"))
		}

		// Limit results to a particular origin, if specified
		if followingID := w._request.URL.Query().Get("origin.followingId"); followingID != "" {
			criteria = criteria.And(exp.Equal("origin.followingId", followingID))
		}

		// Get results from the database
		result, err := inboxService.Query(w._session, criteria, options...)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Querying sibling newsItem", sibling, newsItem.MessageID))
			return model.NewNewsItem()
		}

		// If we have (a) result, then return it.
		if len(result) > 0 {
			newsItem = result[0]
		}

		// Update the QueryString to reflect the "correct" newsItem
		w.SetQueryParam("newsItemId", newsItem.ID())
		w.SetQueryParam("sibling", "")
	}

	// Icky side effect to update the URI parameter to use the new Message
	w.SetQueryParam("newsItemId", newsItem.ID())

	if url := w.QueryParam("url"); url == "" {
		w.SetQueryParam("url", newsItem.URL)
	}

	if folderID := w.QueryParam("folderId"); folderID == "" {
		w.SetQueryParam("folderId", newsItem.FolderID.Hex())
	}

	// Otherwise, there was some error (likely 404 Not Found) so return the original newsItem instead.
	return newsItem
}

// QueryByContext returns the cached documents in a conversation, each labeled with the viewer's rule verdict
func (w Inbox) QueryByContext(contextID string, afterDate int64, maxRows int) (sliceof.Object[streams.Document], error) {

	activityService := w._factory.ActivityStream()
	result, err := activityService.QueryByContext(w._request.Context(), contextID, afterDate, maxRows)

	if err != nil {
		return result, err
	}

	// Stamp each document with the viewer's rule verdict (D2 placeholders + label chips)
	w._factory.Rule().LabelDocuments(w._session, w.AuthenticatedID(), result)

	return result, nil
}

// LikesBefore returns the actors who "Liked" the specified URL, before the specified date.
// Unlike the Stream builder's LikeLinksAfter (which reads a LOCAL Likes collection), the inbox
// object may be remote, so likes are drawn from the federated ActivityStream cache instead.
func (w Inbox) LikesBefore(url string, dateString string, maxRows int) sliceof.Object[streams.Document] {

	done := make(channel.Done)

	// Get all "Like" activities that target the provided URL
	activityService := w._factory.ActivityStream()
	maxDate := convert.Int64Default(dateString, math.MaxInt)
	likes := activityService.QueryLikesBeforeDate(w._request.Context(), url, maxDate, done)

	// Collect into a slice, newest-first
	result := slice.Reverse(channel.Slice(likes))

	// RULE: likes from actors the viewer's rules hide are dropped, not placeheld -- a likes
	// list is a roll call, not thread structure. Aggregate counts are unaffected (R9).
	w._factory.Rule().LabelDocuments(w._session, w.AuthenticatedID(), result)

	return slice.Filter(result, func(document streams.Document) bool {
		return !document.Metadata.Labels.IsHidden()
	})
}

// RepliesAfter returns replies to the specified URL after the specified date
func (w Inbox) RepliesAfter(url string, dateString string, maxRows int) sliceof.Object[streams.Document] {

	activityService := w._factory.ActivityStream()
	minDate := convert.Int64(dateString)
	values := activityService.QueryRepliesAfterDate(w._request.Context(), url, minDate, int64(maxRows))

	// Map cached values into documents, then stamp each with the viewer's rule verdict, so
	// templates can render D2 placeholders for hidden replies without a per-reply rules query.
	result := make(sliceof.Object[streams.Document], 0, len(values))

	for _, value := range values {
		result = append(result, value.AsDocument())
	}

	w._factory.Rule().LabelDocuments(w._session, w.AuthenticatedID(), result)

	return result
}

// AmFollowing returns the Following record for the specified URL.
// If the current user is not following the specified URL, then
// an empty Following record is returned.
func (w Inbox) AmFollowing(url string) model.Following {

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
func (w Inbox) HasRule(ruleType string, trigger string) model.Rule {

	// Get following service and new following record
	ruleService := w._factory.Rule()
	rule := model.NewRule()

	// Null check
	if w._user == nil {
		return rule
	}

	// Retrieve rule record.  "Not Found" is acceptable, but "legitimate" errors are not.
	// In either case, do not halt the request
	if err := ruleService.LoadByMatchKey(w._session, w._user.UserID, ruleType, trigger, &rule); err != nil {
		if !derp.IsNotFound(err) {
			derp.Report(derp.Wrap(err, "build.NewsFeed.HasRule", "Loading rule", ruleType, trigger))
		}
	}

	// Return the (possibly empty) Rule record
	return rule
}

// debug writes this Builder's state to the console. Implements the Builder interface.
func (w Inbox) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Inbox")
}
