package render

import (
	"bytes"
	"html/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Profile struct {
	user *model.User
	Common
}

func NewProfile(factory Factory, ctx *steranko.Context, user *model.User, actionID string) (Profile, error) {

	// Load the Template
	templateService := factory.Template()

	template, err := templateService.Load("user-profile")

	if err != nil {
		return Profile{}, derp.Wrap(err, "render.NewProfile", "Error loading template")
	}

	// Verify the requested action is valid for this template
	action := template.Action(actionID)

	if action == nil {
		return Profile{}, derp.NewBadRequestError("render.NewProfile", "Invalid action", actionID)
	}

	return Profile{
		user:   user,
		Common: NewCommon(factory, ctx, template, action, actionID),
	}, nil
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// Render generates the string value for this Profile
func (w Profile) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w._factory, &w, &buffer); err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Profile.Render", "Error generating HTML"))

	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Profile
func (w Profile) View(actionID string) (template.HTML, error) {

	renderer, err := NewProfile(w._factory, w._context, w.user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "render.Profile.View", "Error creating Profile renderer")
	}

	return renderer.Render()
}

// TopLevelID returns the ID to use for highlighing navigation menus
func (w Profile) TopLevelID() string {

	// TODO: This is returning incorrect values when we CREATE a new outbox item.
	// Is there a better way to handle this that doesn't just HARDCODE stuff in here?

	// If the user is viewing their own profile, then the top-level ID is the user's own ID
	if w.UserID() == w.Common.AuthenticatedID().Hex() {

		if w.ActionID() == "view" {
			return "profile"
		}
		return "inbox"
	}

	return ""
}

func (w Profile) PageTitle() string {

	if w.ActionID() == "view" {
		if w.template().TemplateID == "user-outbox" {
			return "Profile"
		}
		return "Inbox"
	}

	return ""
}

func (w Profile) Permalink() string {
	return w.Host() + "/@" + w.user.UserID.Hex()
}

func (w Profile) Token() string {
	return "users"
}

func (w Profile) object() data.Object {
	return w.user
}

func (w Profile) objectID() primitive.ObjectID {
	return w.user.UserID
}

func (w Profile) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

func (w Profile) service() service.ModelService {
	return w._factory.User()
}

func (w Profile) templateRole() string {
	return "outbox"
}

// UserCan returns TRUE if this Request is authorized to access the requested view
func (w Profile) UserCan(actionID string) bool {

	action := w.template().Action(actionID)

	if action == nil {
		return false
	}

	authorization := w.authorization()

	return action.UserCan(w.user, &authorization)
}

/*******************************************
 * DATA ACCESSORS
 *******************************************/

func (w Profile) UserID() string {
	return w.user.UserID.Hex()
}

func (w Profile) Username() string {
	return w.user.Username
}

func (w Profile) FolderID() string {
	return w.context().QueryParam("folderId")
}

func (w Profile) DisplayName() string {
	return w.user.DisplayName
}

func (w Profile) StatusMessage() string {
	return w.user.StatusMessage
}

func (w Profile) ImageURL() string {
	return w.user.ImageURL
}

func (w Profile) Location() string {
	return w.user.Location
}

func (w Profile) Links() []model.PersonLink {
	return w.user.Links
}

func (w Profile) ActivityPubProfileURL() string {
	return w.user.ActivityPubProfileURL(w.Host())
}

func (w Profile) ActivityPubAvatarURL() string {
	return w.user.ActivityPubAvatarURL(w.Host())
}

func (w Profile) ActivityPubInboxURL() string {
	return w.user.ActivityPubInboxURL(w.Host())
}

func (w Profile) ActivityPubOutboxURL() string {
	return w.user.ActivityPubOutboxURL(w.Host())
}

func (w Profile) ActivityPubFollowersURL() string {
	return w.user.ActivityPubFollowersURL(w.Host())
}

func (w Profile) ActivityPubFollowingURL() string {
	return w.user.ActivityPubFollowingURL(w.Host())
}

func (w Profile) ActivityPubLikedURL() string {
	return w.user.ActivityPubLikedURL(w.Host())
}

func (w Profile) ActivityPubPublicKeyURL() string {
	return w.user.ActivityPubPublicKeyURL(w.Host())
}

/*******************************************
 * QUERY BUILDERS
 *******************************************/

func (w Profile) Inbox() ([]model.Activity, error) {

	if !w.IsAuthenticated() {
		return []model.Activity{}, derp.NewForbiddenError("render.Profile.Inbox", "Not authenticated")
	}

	factory := w._factory

	expBuilder := builder.NewBuilder().
		ObjectID("folderId").
		Int("readDate").
		Int("publishDate")

	criteria := expBuilder.Evaluate(w._context.Request().URL.Query())
	criteria = criteria.And(
		exp.Equal("ownerId", w.AuthenticatedID()),
	)

	return factory.Inbox().Query(criteria, option.MaxRows(10), option.SortAsc("publishDate"))
}

// IsInboxEmpty returns TRUE if the inbox has no results and there are no filters applied
// This corresponds to there being NOTHING in the inbox, instead of just being filtered out.
func (w Profile) IsInboxEmpty(inbox []model.Activity) bool {
	if len(inbox) > 0 {
		return false
	}

	if w._context.Request().URL.Query().Get("publishDate") != "" {
		return false
	}

	return true
}

func (w Profile) Activity() (model.Activity, error) {

	// Guarantee that the user is signed in
	if !w.IsAuthenticated() {
		return model.Activity{}, derp.NewForbiddenError("render.Profile.Activity", "Not authenticated")
	}

	// Try to parse the activityID from the URL
	activityID, err := primitive.ObjectIDFromHex(w._context.QueryParam("activityId"))

	if err != nil {
		return model.Activity{}, derp.NewBadRequestError("render.Profile.Activity", "Invalid activityId", w._context.QueryParam("activityId"))
	}

	// Try to load the record from the database
	result := model.NewActivity()
	inboxService := w._factory.Inbox()

	if err := inboxService.LoadItemByID(w.AuthenticatedID(), activityID, &result); err != nil {
		return model.Activity{}, derp.Wrap(err, "render.Profile.Activity", "Error loading inbox item")
	}

	// Success!
	return result, nil
}

func (w Profile) Folders() ([]model.Folder, error) {

	if !w.IsAuthenticated() {
		return []model.Folder{}, derp.NewForbiddenError("render.Profile.Folders", "Not authenticated")
	}

	folderService := w._factory.Folder()
	return folderService.QueryByUserID(w.AuthenticatedID())
}

func (w Profile) Folder() (model.Folder, error) {

	// Guarantee that the user is signed in
	if !w.IsAuthenticated() {
		return model.Folder{}, derp.NewForbiddenError("render.Profile.Folders", "Not authenticated")
	}

	// Try to load the record from the database
	folder := model.NewFolder()
	folderID := w._context.QueryParam("folderId")
	folderService := w._factory.Folder()

	err := folderService.LoadByToken(w.AuthenticatedID(), folderID, &folder)
	return folder, err
}

func (w Profile) Outbox() *SliceBuilder[model.StreamSummary] {

	if !w.IsAuthenticated() {
		return nil
	}

	queryBuilder := builder.NewBuilder().
		Int("publishDate")

	criteria := exp.And(
		queryBuilder.Evaluate(w._context.Request().URL.Query()),
		exp.Equal("parentId", w.AuthenticatedID()),
	)

	result := NewSliceBuilder[model.StreamSummary](w._factory.Stream(), criteria)

	return &result
}

func (w Profile) Subscriptions() ([]model.SubscriptionSummary, error) {

	userID := w.AuthenticatedID()

	if userID.IsZero() {
		return nil, derp.NewUnauthorizedError("render.Profile.Subscriptions", "Must be signed in to view subscriptions")
	}

	subscriptionService := w._factory.Subscription()

	return subscriptionService.QueryByUserID(userID)
}
