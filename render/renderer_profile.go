package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/EmissarySocial/emissary/model"
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
	template *model.Template
	user     *model.User
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
		template: template,
		user:     user,
		Common:   NewCommon(factory, ctx, action, actionID),
	}, nil
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// Render generates the string value for this Profile
func (w Profile) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer); err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Profile.Render", "Error generating HTML"))

	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Profile
func (w Profile) View(actionID string) (template.HTML, error) {

	renderer, err := NewProfile(w.factory(), w.ctx, w.user, actionID)

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
		if w.template.TemplateID == "user-outbox" {
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
	return w.user.Schema()
}

func (w Profile) service() ModelService {
	return w.f.User()
}

func (w Profile) executeTemplate(writer io.Writer, name string, data any) error {
	return w.template.HTMLTemplate.ExecuteTemplate(writer, name, data)
}

// UserCan returns TRUE if this Request is authorized to access the requested view
func (w Profile) UserCan(actionID string) bool {

	action := w.template.Action(actionID)

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

	factory := w.factory()

	query := builder.NewBuilder().
		Int("publishDate").
		ObjectID("inboxFolderId")

	criteria := exp.And(
		query.Evaluate(w.ctx.Request().URL.Query()),
		exp.Equal("userId", w.AuthenticatedID()),
	)

	return factory.Inbox().Query(criteria, option.MaxRows(60), option.SortAsc("publishDate"))
}

func (w Profile) InboxItem() (model.InboxItem, error) {

	// Guarantee that the user is signed in
	if !w.IsAuthenticated() {
		return model.InboxItem{}, derp.NewForbiddenError("render.Profile.InboxItem", "Not authenticated")
	}

	// Convert the inboxItemID QueryParam to an ObjectID
	inboxItemID, err := primitive.ObjectIDFromHex(w.ctx.QueryParam("inboxItemId"))

	if err != nil {
		return model.InboxItem{}, derp.New(derp.CodeBadRequestError, "render.Profile.InboxItem", "Invalid inboxItemID", err)
	}

	// Try to load the record from the database
	result := model.NewInboxItem()
	inboxService := w.factory().Inbox()
	err = inboxService.LoadItemByID(w.AuthenticatedID(), inboxItemID, &result)

	// Success!
	return result, err
}

func (w Profile) InboxFolders() ([]model.InboxFolder, error) {

	if !w.IsAuthenticated() {
		return []model.InboxFolder{}, derp.NewForbiddenError("render.Profile.InboxFolders", "Not authenticated")
	}

	inboxFolderService := w.factory().InboxFolder()
	return inboxFolderService.QueryByUserID(w.AuthenticatedID())
}

func (w Profile) Outbox() *QueryBuilder {

	if !w.IsAuthenticated() {
		return nil
	}

	factory := w.factory()
	context := w.context()

	query := builder.NewBuilder().
		Int("publishDate")

	criteria := exp.And(
		query.Evaluate(w.ctx.Request().URL.Query()),
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

	subscriptionService := w.factory().Subscription()

	return subscriptionService.QueryByUserID(userID)
}
