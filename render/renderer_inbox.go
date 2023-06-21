package render

import (
	"bytes"
	"html/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Inbox struct {
	user *model.User
	Common
}

func NewInbox(factory Factory, ctx *steranko.Context, user *model.User, actionID string) (Inbox, error) {

	// Load the Template
	templateService := factory.Template()
	template, err := templateService.Load("user-inbox") // TODO: Users should get to select their inbox template

	if err != nil {
		return Inbox{}, derp.Wrap(err, "render.NewInbox", "Error loading template")
	}

	// Create the underlying Common renderer
	common, err := NewCommon(factory, ctx, template, actionID)

	if err != nil {
		return Inbox{}, derp.Wrap(err, "render.NewInbox", "Error creating common renderer")
	}

	return Inbox{
		user:   user,
		Common: common,
	}, nil
}

/******************************************
 * RENDERER INTERFACE
 ******************************************/

// Render generates the string value for this Inbox
func (w Inbox) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w.action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		return "", derp.Report(derp.Wrap(status.Error, "render.Inbox.Render", "Error generating HTML", w._context.Request().URL.String()))
	}

	// Success!
	status.Apply(w._context)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Inbox
func (w Inbox) View(actionID string) (template.HTML, error) {

	renderer, err := NewInbox(w._factory, w._context, w.user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "render.Inbox.View", "Error creating Inbox renderer")
	}

	return renderer.Render()
}

// NavigationID returns the ID to use for highlighing navigation menus
func (w Inbox) NavigationID() string {
	return "inbox"
}

func (w Inbox) PageTitle() string {
	return w.user.DisplayName
}

func (w Inbox) Permalink() string {
	return w.Host() + "/@me/inbox"
}

func (w Inbox) Token() string {
	return "users"
}

func (w Inbox) object() data.Object {
	return w.user
}

func (w Inbox) objectID() primitive.ObjectID {
	return w.user.UserID
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

func (w Inbox) clone(action string) (Renderer, error) {
	return NewInbox(w._factory, w._context, w.user, action)
}

// UserCan returns TRUE if this Request is authorized to access the requested view
func (w Inbox) UserCan(actionID string) bool {

	action, ok := w._template.Action(actionID)

	if !ok {
		return false
	}

	authorization := w.authorization()

	return action.UserCan(w.user, &authorization)
}

/******************************************
 * Data Accessors
 ******************************************/

func (w Inbox) UserID() string {
	return w.user.UserID.Hex()
}

// Myself returns TRUE if the current user is viewing their own profile
func (w Inbox) Myself() bool {
	authorization := getAuthorization(w._context)

	if err := authorization.Valid(); err == nil {
		return authorization.UserID == w.user.UserID
	}

	return false
}

func (w Inbox) Username() string {
	return w.user.Username
}

func (w Inbox) FollowerCount() int {
	return w.user.FollowerCount
}

func (w Inbox) FollowingCount() int {
	return w.user.FollowingCount
}

func (w Inbox) BlockCount() int {
	return w.user.BlockCount
}

func (w Inbox) DisplayName() string {
	return w.user.DisplayName
}

func (w Inbox) ProfileURL() string {
	return w.user.ProfileURL
}

func (w Inbox) ImageURL() string {
	return w.user.ActivityPubAvatarURL()
}

/******************************************
 * Inbox Methods
 ******************************************/

func (w Inbox) Followers() QueryBuilder[model.FollowerSummary] {

	expressionBuilder := builder.NewBuilder().
		String("displayName")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._context.Request().URL.Query()),
		exp.Equal("parentId", w.AuthenticatedID()),
	)

	result := NewQueryBuilder[model.FollowerSummary](w._factory.Follower(), criteria)

	return result
}

func (w Inbox) Following() ([]model.FollowingSummary, error) {

	userID := w.AuthenticatedID()

	if userID.IsZero() {
		return nil, derp.NewUnauthorizedError("render.Inbox.Following", "Must be signed in to view following")
	}

	followingService := w._factory.Following()

	return followingService.QueryByUser(userID)
}

func (w Inbox) FollowingByFolder(token string) ([]model.FollowingSummary, error) {

	// Get the UserID from the authentication scope
	userID := w.AuthenticatedID()

	if userID.IsZero() {
		return nil, derp.NewUnauthorizedError("render.Inbox.FollowingByFolder", "Must be signed in to view following")
	}

	// Get the followingID from the token
	followingID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return nil, derp.Wrap(err, "render.Inbox.FollowingByFolder", "Invalid following ID", token)
	}

	// Try to load the matching records
	followingService := w._factory.Following()
	return followingService.QueryByFolder(userID, followingID)

}

func (w Inbox) Blocks() QueryBuilder[model.Block] {

	expressionBuilder := builder.NewBuilder()

	criteria := exp.And(
		expressionBuilder.Evaluate(w._context.Request().URL.Query()),
		exp.Equal("userId", w.AuthenticatedID()),
	)

	result := NewQueryBuilder[model.Block](w._factory.Block(), criteria)

	return result
}

func (w Inbox) BlocksByType(blockType string) QueryBuilder[model.Block] {

	expressionBuilder := builder.NewBuilder()

	criteria := exp.And(
		expressionBuilder.Evaluate(w._context.Request().URL.Query()),
		exp.Equal("userId", w.AuthenticatedID()),
		exp.Equal("type", blockType),
	)

	result := NewQueryBuilder[model.Block](w._factory.Block(), criteria)

	return result
}

func (w Inbox) CountBlocks(blockType string) (int, error) {
	return w._factory.Block().CountByType(w.objectID(), blockType)
}

// Inbox returns a slice of messages in the current User's inbox
func (w Inbox) Inbox() (QueryBuilder[model.Message], error) {

	userID := w.AuthenticatedID()

	if userID.IsZero() {
		return QueryBuilder[model.Message]{}, derp.NewUnauthorizedError("render.Inbox.Inbox", "Must be signed in to view inbox")
	}

	folderID, err := primitive.ObjectIDFromHex(w.context().Request().URL.Query().Get("folderId"))

	if err != nil {
		return QueryBuilder[model.Message]{}, derp.Wrap(err, "render.Inbox.Inbox", "Invalid folderId", w.context().QueryParam("folderId"))
	}

	expBuilder := builder.NewBuilder().
		ObjectID("origin.followingId").
		ObjectID("folderId").
		Int("rank")

	criteria := exp.And(
		exp.Equal("userId", w.AuthenticatedID()),
		exp.Equal("folderId", folderID),
		expBuilder.Evaluate(w._context.Request().URL.Query()),
	)

	return NewQueryBuilder[model.Message](w._factory.Inbox(), criteria), nil
}

// IsInboxEmpty returns TRUE if the inbox has no results and there are no filters applied
// This corresponds to there being NOTHING in the inbox, instead of just being filtered out.
func (w Inbox) IsInboxEmpty(inbox []model.Message) bool {

	if len(inbox) > 0 {
		return false
	}

	if w._context.Request().URL.Query().Get("rank") != "" {
		return false
	}

	return true
}

// FIlteredByFollowing returns the Following record that is being used to filter the Inbox
func (w Inbox) FilteredByFollowing() model.Following {

	result := model.NewFollowing()

	if !w.IsAuthenticated() {
		return result
	}

	token := w._context.QueryParam("origin.followingId")

	if followingID, err := primitive.ObjectIDFromHex(token); err == nil {
		followingService := w._factory.Following()

		if err := followingService.LoadByID(w.AuthenticatedID(), followingID, &result); err == nil {
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
		return result, derp.NewForbiddenError("render.Inbox.Folders", "Not authenticated")
	}

	folderService := w._factory.Folder()
	folders, err := folderService.QueryByUserID(w.AuthenticatedID())

	if err != nil {
		return result, derp.Wrap(err, "render.Inbox.Folders", "Error loading folders")
	}

	result.Folders = folders
	return result, nil
}

func (w Inbox) FoldersWithSelection() (model.FolderList, error) {

	// Get Folder List
	result, err := w.Folders()

	if err != nil {
		return result, derp.Wrap(err, "render.Inbox.FoldersWithSelection", "Error loading folders")
	}

	// Guarantee that we have at least one folder
	if len(result.Folders) == 0 {
		return result, derp.NewInternalError("render.Inbox.FoldersWithSelection", "No folders found", nil)
	}

	// Find/Mark the Selected FolderID
	token := w._context.QueryParam("folderId")

	if folderID, err := primitive.ObjectIDFromHex(token); err == nil {
		result.SelectedID = folderID
	} else {
		result.SelectedID = result.Folders[0].FolderID
	}

	// Update the query string to reflect the selected folder
	w.setQuery("folderId", result.SelectedID.Hex())
	if w._context.QueryParam("rank") == "" {
		w.setQuery("rank", "GT:"+result.Selected().ReadDateString())
	}

	// Return the result
	return result, nil
}

// Responses generates a "Responses" renderer and passes it to the (hard-coded named) "responses" template.
// A default file is provided in the "base-social" template but can be overridden by other installed packages.
func (w Inbox) Responses(message model.Message) template.HTML {

	// Collect values for Responses renderer
	userID := w.authorization().UserID
	internalURL := "/@me/messages/" + message.MessageID.Hex()
	responseService := w.factory().Response()

	renderer := NewResponses(userID, internalURL, message.URL, responseService)

	// Render the responses widget
	var buffer bytes.Buffer

	// nolint:errcheck
	if err := w._template.HTMLTemplate.ExecuteTemplate(&buffer, "responses", renderer); err != nil {
		derp.Report(derp.Wrap(err, "render.Inbox.Responses", "Error rendering responses"))
	}

	// Celebrate with Triumph.
	return template.HTML(buffer.String())
}

func (w Inbox) debug() {
	spew.Dump("Inbox")
}
