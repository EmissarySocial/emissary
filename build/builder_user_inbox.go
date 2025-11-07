package build

import (
	"bytes"
	"html/template"
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
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
		return Inbox{}, derp.Wrap(err, location, "Unable to load template")
	}

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, user, actionID)

	if err != nil {
		return Inbox{}, derp.Wrap(err, location, "Unable to create common builder")
	}

	// Enforce user permissions on the requested action
	if !common.UserCan(actionID) {
		if common._authorization.IsAuthenticated() {
			return Inbox{}, derp.ForbiddenError(location, "Forbidden", "User is authenticated, but this action is not allowed", actionID)
		} else {
			return Inbox{}, derp.UnauthorizedError(location, "Anonymous user is not authorized to perform this action", user.ProfileURL, actionID)
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
		return "", derp.Wrap(status.Error, "build.Inbox.Render", "Error generating HTML", w._request.URL.String())
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Inbox
func (w Inbox) View(actionID string) (template.HTML, error) {

	builder, err := NewInbox(w._factory, w._session, w._request, w._response, w._user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.Inbox.View", "Unable to create Inbox builder")
	}

	return builder.Render()
}

// NavigationID returns the ID to use for highlighing navigation menus
func (w Inbox) NavigationID() string {
	return "inbox"
}

func (w Inbox) PageTitle() string {
	return w._user.DisplayName
}

func (w Inbox) BasePath() string {
	return "/@me/inbox"
}

func (w Inbox) Permalink() string {

	if message := w.Message(); !message.IsNew() {
		return message.URL
	}

	if url := w._request.URL.Query().Get("url"); url != "" {
		return url
	}

	return w.Host() + "/@me/inbox"
}

func (w Inbox) Token() string {
	return "users"
}

func (w Inbox) object() data.Object {
	return w._user
}

func (w Inbox) objectID() primitive.ObjectID {
	return w._user.UserID
}

func (w Inbox) objectType() string {
	return "User"
}

func (w Inbox) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

func (w Inbox) service() service.ModelService {
	return w._factory.User()
}

func (w Inbox) templateRole() string {
	return "inbox"
}

func (w Inbox) clone(action string) (Builder, error) {
	return NewInbox(w._factory, w._session, w._request, w._response, w._user, action)
}

/******************************************
 * Data Accessors
 ******************************************/

func (w Inbox) UserID() string {
	return w._user.UserID.Hex()
}

// Myself returns TRUE if the current user is viewing their own profile
func (w Inbox) Myself() bool {
	return w._authorization.UserID == w._user.UserID
}

func (w Inbox) Username() string {
	return w._user.Username
}

func (w Inbox) FollowerCount() int {
	return w._user.FollowerCount
}

func (w Inbox) FollowingCount() int {
	return w._user.FollowingCount
}

func (w Inbox) RuleCount() int {
	return w._user.RuleCount
}

func (w Inbox) DisplayName() string {
	return w._user.DisplayName
}

func (w Inbox) ProfileURL() string {
	return w._user.ProfileURL
}

func (w Inbox) IconURL() string {
	return w._user.ActivityPubIconURL()
}

/******************************************
 * Inbox Methods
 ******************************************/

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

func (w Inbox) FollowingByFolder(token string) ([]model.FollowingSummary, error) {

	// Get the UserID from the authentication scope
	userID := w.AuthenticatedID()

	if userID.IsZero() {
		return nil, derp.UnauthorizedError("build.Inbox.FollowingByFolder", "Must be signed in to view following")
	}

	// Get the followingID from the token
	followingID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return nil, derp.Wrap(err, "build.Inbox.FollowingByFolder", "Invalid following ID", token)
	}

	// Try to load the matching records
	followingService := w._factory.Following()
	return followingService.QueryByFolder(w._session, userID, followingID)
}

func (w Inbox) FollowingByToken(followingToken string) (model.Following, error) {

	userID := w.AuthenticatedID()

	followingService := w._factory.Following()

	following := model.NewFollowing()

	if err := followingService.LoadByToken(w._session, userID, followingToken, &following); err != nil {
		return model.Following{}, derp.Wrap(err, "build.Inbox.FollowingByID", "Unable to load following")
	}

	return following, nil
}

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

func (w Inbox) RuleByToken(token string) model.Rule {
	ruleService := w._factory.Rule()
	rule := model.NewRule()

	if err := ruleService.LoadByToken(w._session, w.AuthenticatedID(), token, &rule); err != nil {
		derp.Report(derp.Wrap(err, "build.Inbox.RuleByToken", "Unable to load rule", token))
	}

	return rule
}

// Inbox returns a QueryBuilder for current User's inbox
func (w Inbox) Inbox() (QueryBuilder[model.Message], error) {

	userID := w.AuthenticatedID()

	if userID.IsZero() {
		return QueryBuilder[model.Message]{}, derp.UnauthorizedError("build.Inbox.Inbox", "Must be signed in to view inbox")
	}

	queryString := w._request.URL.Query()

	folderID, err := primitive.ObjectIDFromHex(queryString.Get("folderId"))

	if err != nil {
		return QueryBuilder[model.Message]{}, derp.Wrap(err, "build.Inbox.Inbox", "Invalid folderId", queryString.Get("folderId"))
	}

	expBuilder := builder.NewBuilder().
		ObjectID("origin.followingId").
		ObjectID("followingId", builder.WithAlias("origin.followingId")).
		Int("rank").
		Int("readDate").
		Int("createDate")

	criteria := exp.And(
		exp.Equal("userId", w.AuthenticatedID()),
		exp.Equal("folderId", folderID),
		exp.Equal("deleteDate", 0),
		expBuilder.Evaluate(queryString),
	)

	return NewQueryBuilder[model.Message](w._factory.Inbox(), w._session, criteria), nil
}

// IsInboxEmpty returns TRUE if the inbox has no results and there are no filters applied
// This corresponds to there being NOTHING in the inbox, instead of just being filtered out.
func (w Inbox) IsInboxEmpty(inbox []model.Message) bool {

	if len(inbox) > 0 {
		return false
	}

	if w._request.URL.Query().Get("rank") != "" {
		return false
	}

	return true
}

// Conversations returns a QueryBuilder for current User's conversations
func (w Inbox) Conversations() (QueryBuilder[model.Conversation], error) {

	// Collect the currently authenticated user
	userID := w.AuthenticatedID()

	if userID.IsZero() {
		return QueryBuilder[model.Conversation]{}, derp.UnauthorizedError("build.Inbox.Conversations", "Must be signed in to view conversations")
	}

	queryString := w._request.URL.Query()

	expBuilder := builder.NewBuilder().
		Int("updateDate")

	criteria := exp.And(
		exp.Equal("userId", w.AuthenticatedID()),
		exp.Equal("deleteDate", 0),
		expBuilder.Evaluate(queryString),
	)

	conversationService := w._factory.Conversation()

	return NewQueryBuilder[model.Conversation](&conversationService, w._session, criteria), nil
}

func (w Inbox) Privileges() QueryBuilder[model.Privilege] {

	expressionBuilder := builder.NewBuilder().
		String("search", builder.WithAlias("emailAddress"), builder.WithDefaultOpBeginsWith())

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w._user.UserID),
	)

	return NewQueryBuilder[model.Privilege](w._factory.Privilege(), w._session, criteria)

}

func (w Inbox) MerchantAccount(merchantAccountID string) (model.MerchantAccount, error) {
	result := model.NewMerchantAccount()
	err := w._factory.MerchantAccount().LoadByUserAndToken(w._session, w._user.UserID, merchantAccountID, &result)
	return result, err
}

func (w Inbox) MerchantAccounts() QueryBuilder[model.MerchantAccount] {

	expressionBuilder := builder.NewBuilder().
		String("search", builder.WithAlias("name"), builder.WithDefaultOpBeginsWith())

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w._user.UserID),
	)

	return NewQueryBuilder[model.MerchantAccount](w._factory.MerchantAccount(), w._session, criteria)
}

// FIlteredByFollowing returns the Following record that is being used to filter the Inbox
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
		return result, derp.ForbiddenError("build.Inbox.Folders", "Not authenticated")
	}

	folderService := w._factory.Folder()
	folders, err := folderService.QueryByUserID(w._session, w.AuthenticatedID())

	if err != nil {
		return result, derp.Wrap(err, "build.Inbox.Folders", "Unable to load folders")
	}

	result.Folders = folders
	return result, nil
}

func (w Inbox) FoldersWithSelection() (model.FolderList, error) {

	const location = "build.Inbox.FoldersWithSelection"

	// Get Folder List
	result, err := w.Folders()

	if err != nil {
		return result, derp.Wrap(err, location, "Unable to load folders")
	}

	// Guarantee that we have at least one folder
	if len(result.Folders) == 0 {
		return result, derp.InternalError(location, "No folders found", nil)
	}

	// Find/Mark the Selected FolderID
	token := w._request.URL.Query().Get("folderId")

	if folderID, err := primitive.ObjectIDFromHex(token); err == nil {
		result.SelectedID = folderID
	} else {
		result.SelectedID = result.Folders[0].FolderID
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
		result, err = nil, derp.InternalError("build.Common.SubBuilder", "Invalid object type", object)
	}

	if err != nil {
		err = derp.Wrap(err, "build.Common.SubBuilder", "Unable to create sub-builder for object", object)
		derp.Report(err)
	}

	return result, err
}

/******************************************
 * Message Methods
 ******************************************/

// Message uses the queryString ?messageId= parameter to load a Message from the database
// If the messageId parameter does not exist, is malformed, or if the message does not exist, then
// a new, empty Message is returned.
// In addition, if there is a "sibling" URL parameter (either "next" or "prev") then the next/previous
// message is loaded instead.
func (w Inbox) Message() model.Message {

	const location = "build.Inbox.Message"

	// Get the messageID from the query string
	messageID, err := primitive.ObjectIDFromHex(w._request.URL.Query().Get("messageId"))

	if err != nil {
		return model.NewMessage()
	}

	// Load the message from the database
	inboxService := w._factory.Inbox()
	message := model.NewMessage()

	if err := inboxService.LoadByID(w._session, w.AuthenticatedID(), messageID, &message); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to load message", messageID))
		return model.NewMessage()
	}

	// If sibling (prev/next) is specified, then try to look that up before returning.
	if sibling := w._request.URL.Query().Get("sibling"); sibling != "" {

		// Otherwise, look up the next/previous message
		criteria := exp.Equal("userId", w.AuthenticatedID()).AndEqual("folderId", message.FolderID)
		options := []option.Option{option.MaxRows(1)}

		if sibling == "next" {
			criteria = criteria.And(exp.GreaterThan("rank", message.Rank))
			options = append(options, option.SortAsc("rank"))
		} else {
			criteria = criteria.And(exp.LessThan("rank", message.Rank))
			options = append(options, option.SortDesc("rank"))
		}

		// Limit results to a particular origin, if specified
		if followingID := w._request.URL.Query().Get("origin.followingId"); followingID != "" {
			criteria = criteria.And(exp.Equal("origin.followingId", followingID))
		}

		// Get results from the database
		result, _ := inboxService.Query(w._session, criteria, options...)

		// If we have (a) result, then return it.
		if len(result) > 0 {
			message = result[0]
		}

		// Update the QueryString to reflect the "correct" message
		w.SetQueryParam("messageId", message.ID())
		w.SetQueryParam("sibling", "")
	}

	// Icky side effect to update the URI parameter to use the new Message
	w.SetQueryParam("messageId", message.MessageID.Hex())

	if url := w.QueryParam("url"); url == "" {
		w.SetQueryParam("url", message.URL)
	}

	if folderID := w.QueryParam("folderId"); folderID == "" {
		w.SetQueryParam("folderId", message.FolderID.Hex())
	}

	// Otherwise, there was some error (likely 404 Not Found) so return the original message instead.
	return message
}

func (w Inbox) QueryByContext(contextID string, afterDate int64, maxRows int) (sliceof.Object[streams.Document], error) {
	activityService := w._factory.ActivityStream(model.ActorTypeUser, w.AuthenticatedID())
	result, err := activityService.QueryByContext(w._request.Context(), contextID, afterDate, maxRows)
	return result, err
}

/*
func (w Inbox) QueryByContext_Tree(contextID string) (sliceof.Object[*treebuilder.Tree[model.DocumentLink]], error) {
	activityService := w._factory.ActivityStream(model.ActorTypeUser, w.AuthenticatedID())
	result, err := activityService.QueryByContext_Tree(w._request.Context(), contextID)

	return result, err
}
*/

func (w Inbox) RepliesBefore(url string, dateString string, maxRows int) sliceof.Object[streams.Document] {

	done := make(channel.Done)

	// Get all ActivityStreams that reply to the provided URL
	activityService := w._factory.ActivityStream(model.ActorTypeUser, w.AuthenticatedID())
	maxDate := convert.Int64Default(dateString, math.MaxInt)
	replies := activityService.QueryRepliesBeforeDate(w._request.Context(), url, maxDate, done)

	// Filter replies based on rules
	ruleService := w._factory.Rule()
	ruleFilter := ruleService.Filter(w.AuthenticatedID())
	filteredReplies := ruleFilter.Channel(replies)

	// Limit to maximum number of replies
	// limitedReplies := channel.Limit(maxRows, filteredReplies, done)
	// result := channel.Slice(limitedReplies)
	result := channel.Slice(filteredReplies)

	// For glory and honor!
	return slice.Reverse(result)
}

func (w Inbox) RepliesAfter(url string, dateString string, maxRows int) sliceof.Object[ascache.Value] {
	activityService := w._factory.ActivityStream(model.ActorTypeUser, w.AuthenticatedID())
	minDate := convert.Int64(dateString)
	return activityService.QueryRepliesAfterDate(w._request.Context(), url, minDate, int64(maxRows))

	/*
		done := make(channel.Done)

		// Get all ActivityStreams that reply to the provided URL
		activityService := w._factory.ActivityStream(model.ActorTypeUser, w.AuthenticatedID())
		minDate := convert.Int64(dateString)
		replies := activityService.QueryRepliesAfterDate(w._request.Context(), url, minDate, done)

		// Filter replies based on rules
		ruleService := w._factory.Rule()
		ruleFilter := ruleService.Filter(w.AuthenticatedID())
		filteredReplies := ruleFilter.Channel(replies)

		// Limit to maximum number of replies
		limitedReplies := channel.Limit(maxRows, filteredReplies, done)
		result := channel.Slice(limitedReplies)

		// Invictus
		return result
	*/
}

/*
func (w Inbox) AnnouncesBefore(url string, dateString string, maxRows int) sliceof.Object[streams.Document] {

	done := make(channel.Done)

	// Get all ActivityStreams that announce the provided URL
	activityService := w._factory.ActivityStream(model.ActorTypeUser, w.AuthenticatedID())
	maxDate := convert.Int64Default(dateString, math.MaxInt64)
	announces := activityService.QueryAnnouncesBeforeDate(w._request.Context(), url, maxDate, done)

	// Filter replies based on rules
	ruleService := w._factory.Rule()
	ruleFilter := ruleService.Filter(w.AuthenticatedID())
	filteredAnnounces := ruleFilter.Channel(announces)

	// Limit to maximum number of replies
	limitedAnnounces := channel.Limit(maxRows, filteredAnnounces, done)
	result := channel.Slice(limitedAnnounces)

	// Victory
	return slice.Reverse(result)
}

func (w Inbox) LikesBefore(url string, dateString string, maxRows int) sliceof.Object[streams.Document] {

	done := make(channel.Done)

	// Get all ActivityStreams that announce the provided URL
	activityService := w._factory.ActivityStream(model.ActorTypeUser, w.AuthenticatedID())
	maxDate := convert.Int64Default(dateString, math.MaxInt64)
	announces := activityService.QueryLikesBeforeDate(w._request.Context(), url, maxDate, done)

	// Filter replies based on rules
	ruleService := w._factory.Rule()
	ruleFilter := ruleService.Filter(w.AuthenticatedID())
	filteredLikes := ruleFilter.Channel(announces)

	// Limit to maximum number of replies
	limitedLikes := channel.Limit(maxRows, filteredLikes, done)
	result := channel.Slice(limitedLikes)

	// Success
	return slice.Reverse(result)
}
*/

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
	if err := ruleService.LoadByTrigger(w._session, w._user.UserID, ruleType, trigger, &rule); err != nil {
		if !derp.IsNotFound(err) {
			derp.Report(derp.Wrap(err, "build.Inbox.HasRule", "Unable to load rule", ruleType, trigger))
		}
	}

	// Return the (possibly empty) Rule record
	return rule
}

func (w Inbox) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Inbox")
}
