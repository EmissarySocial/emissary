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
	return ""
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

func (w Profile) InboxFolderID() string {
	return w.context().QueryParam("inboxFolderId")
}

func (w Profile) DisplayName() string {
	return w.user.DisplayName
}

func (w Profile) Description() string {
	return w.user.Description
}

func (w Profile) ImageURL() string {
	return w.user.ImageURL
}

/*******************************************
 * QUERY BUILDERS
 *******************************************/

func (w Profile) Inbox() ([]model.InboxItem, error) {

	if !w.IsAuthenticated() {
		return []model.InboxItem{}, derp.NewForbiddenError("render.Profile.Inbox", "Not authenticated")
	}

	factory := w._factory

	expBuilder := builder.NewBuilder().
		ObjectID("inboxFolderId").
		Int("readDate").
		Int("publishDate")

	criteria := expBuilder.Evaluate(w._context.Request().URL.Query())
	criteria = criteria.And(
		exp.Equal("userId", w.AuthenticatedID()),
	)

	return factory.Inbox().Query(criteria, option.MaxRows(10), option.SortAsc("publishDate"))

}

// IsInboxEmpty returns TRUE if the inbox has no results and there are no filters applied
// This corresponds to there being NOTHING in the inbox, instead of just being filtered out.
func (w Profile) IsInboxEmpty(inbox []model.InboxItem) bool {
	if len(inbox) > 0 {
		return false
	}

	if w._context.Request().URL.Query().Get("publishDate") != "" {
		return false
	}

	return true
}

func (w Profile) InboxItem() (model.InboxItem, error) {

	// Guarantee that the user is signed in
	if !w.IsAuthenticated() {
		return model.InboxItem{}, derp.NewForbiddenError("render.Profile.InboxItem", "Not authenticated")
	}

	// Try to parse the inboxItemID from the URL
	inboxItemID, err := primitive.ObjectIDFromHex(w._context.QueryParam("inboxItemId"))

	if err != nil {
		return model.InboxItem{}, derp.NewBadRequestError("render.Profile.InboxItem", "Invalid inboxItemId", w._context.QueryParam("inboxItemId"))
	}

	// Try to load the record from the database
	result := model.NewInboxItem()
	inboxService := w._factory.Inbox()

	if err := inboxService.LoadItemByID(w.AuthenticatedID(), inboxItemID, &result); err != nil {
		return model.InboxItem{}, derp.Wrap(err, "render.Profile.InboxItem", "Error loading inbox item")
	}

	// Success!
	return result, nil
}

func (w Profile) InboxFolders() ([]model.InboxFolder, error) {

	if !w.IsAuthenticated() {
		return []model.InboxFolder{}, derp.NewForbiddenError("render.Profile.InboxFolders", "Not authenticated")
	}

	inboxFolderService := w._factory.InboxFolder()
	return inboxFolderService.QueryByUserID(w.AuthenticatedID())
}

func (w Profile) InboxFolder() (model.InboxFolder, error) {

	// Guarantee that the user is signed in
	if !w.IsAuthenticated() {
		return model.InboxFolder{}, derp.NewForbiddenError("render.Profile.InboxFolders", "Not authenticated")
	}

	// Try to load the record from the database
	inboxFolder := model.NewInboxFolder()
	inboxFolderID := w._context.QueryParam("inboxFolderId")
	inboxFolderService := w._factory.InboxFolder()

	err := inboxFolderService.LoadByToken(w.AuthenticatedID(), inboxFolderID, &inboxFolder)
	return inboxFolder, err
}

func (w Profile) Outbox() *QueryBuilder {

	if !w.IsAuthenticated() {
		return nil
	}

	factory := w._factory
	context := w.context()

	query := builder.NewBuilder().
		Int("publishDate")

	criteria := exp.And(
		query.Evaluate(w._context.Request().URL.Query()),
		exp.Equal("userId", w.AuthenticatedID()),
	)

	result := NewQueryBuilder(factory, context, factory.Stream(), criteria)
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
