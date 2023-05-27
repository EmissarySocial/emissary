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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Outbox struct {
	user *model.User
	Common
}

func NewOutbox(factory Factory, ctx *steranko.Context, user *model.User, actionID string) (Outbox, error) {

	// Load the Template
	templateService := factory.Template()
	template, err := templateService.Load("user-outbox") // Users should get to choose their own outbox template

	if err != nil {
		return Outbox{}, derp.Wrap(err, "render.NewOutbox", "Error loading template")
	}

	// Create the underlying Common renderer
	common, err := NewCommon(factory, ctx, template, actionID)

	if err != nil {
		return Outbox{}, derp.Wrap(err, "render.NewOutbox", "Error creating common renderer")
	}

	return Outbox{
		user:   user,
		Common: common,
	}, nil
}

/******************************************
 * RENDERER INTERFACE
 ******************************************/

// Render generates the string value for this Outbox
func (w Outbox) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w._factory, &w, &buffer); err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Outbox.Render", "Error generating HTML", w._context.Request().URL.String()))

	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Outbox
func (w Outbox) View(actionID string) (template.HTML, error) {

	renderer, err := NewOutbox(w._factory, w._context, w.user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "render.Outbox.View", "Error creating Outbox renderer")
	}

	return renderer.Render()
}

// NavigationID returns the ID to use for highlighing navigation menus
func (w Outbox) NavigationID() string {
	return "outbox"
}

func (w Outbox) PageTitle() string {
	return w.user.DisplayName
}

func (w Outbox) Permalink() string {
	return w.Host() + "/@" + w.user.UserID.Hex()
}

func (w Outbox) Token() string {
	return "users"
}

func (w Outbox) object() data.Object {
	return w.user
}

func (w Outbox) objectID() primitive.ObjectID {
	return w.user.UserID
}

func (w Outbox) objectType() string {
	return "User"
}

func (w Outbox) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

func (w Outbox) service() service.ModelService {
	return w._factory.User()
}

func (w Outbox) templateRole() string {
	return "outbox"
}

func (w Outbox) clone(action string) (Renderer, error) {
	return NewOutbox(w._factory, w._context, w.user, action)
}

// UserCan returns TRUE if this Request is authorized to access the requested view
func (w Outbox) UserCan(actionID string) bool {

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

func (w Outbox) UserID() string {
	return w.user.UserID.Hex()
}

// Myself returns TRUE if the current user is viewing their own profile
func (w Outbox) Myself() bool {
	authorization := getAuthorization(w._context)

	if err := authorization.Valid(); err == nil {
		return authorization.UserID == w.user.UserID
	}

	return false
}

func (w Outbox) Username() string {
	return w.user.Username
}

func (w Outbox) BlockCount() int {
	return w.user.BlockCount
}

func (w Outbox) DisplayName() string {
	return w.user.DisplayName
}

func (w Outbox) StatusMessage() string {
	return w.user.StatusMessage
}

func (w Outbox) ProfileURL() string {
	return w.user.ProfileURL
}

func (w Outbox) ImageURL() string {
	return w.user.ActivityPubAvatarURL()
}

func (w Outbox) Location() string {
	return w.user.Location
}

func (w Outbox) Links() []model.PersonLink {
	return w.user.Links
}

func (w Outbox) ActivityPubURL() string {
	return w.user.ActivityPubURL()
}

func (w Outbox) ActivityPubAvatarURL() string {
	return w.user.ActivityPubAvatarURL()
}

func (w Outbox) ActivityPubInboxURL() string {
	return w.user.ActivityPubInboxURL()
}

func (w Outbox) ActivityPubOutboxURL() string {
	return w.user.ActivityPubOutboxURL()
}

func (w Outbox) ActivityPubFollowersURL() string {
	return w.user.ActivityPubFollowersURL()
}

func (w Outbox) ActivityPubFollowingURL() string {
	return w.user.ActivityPubFollowingURL()
}

func (w Outbox) ActivityPubLikedURL() string {
	return w.user.ActivityPubLikedURL()
}

func (w Outbox) ActivityPubPublicKeyURL() string {
	return w.user.ActivityPubPublicKeyURL()
}

/******************************************
 * Outbox Methods
 ******************************************/

func (w Outbox) Outbox() QueryBuilder[model.StreamSummary] {

	expressionBuilder := builder.NewBuilder().
		Int("publishDate")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._context.Request().URL.Query()),
		exp.Equal("parentId", w.AuthenticatedID()),
	)

	result := NewQueryBuilder[model.StreamSummary](w._factory.Stream(), criteria)

	return result
}
