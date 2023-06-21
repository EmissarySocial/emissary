package render

import (
	"bytes"
	"html/template"
	"io"

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

type User struct {
	user *model.User
	Common
}

func NewUser(factory Factory, ctx *steranko.Context, template model.Template, user *model.User, actionID string) (User, error) {

	const location = "render.NewGroup"

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return User{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	// Create the underlying Common renderer
	common, err := NewCommon(factory, ctx, template, actionID)

	if err != nil {
		return User{}, derp.Wrap(err, location, "Error creating common renderer")
	}

	// Return the User renderer
	return User{
		user:   user,
		Common: common,
	}, nil
}

/******************************************
 * RENDERER INTERFACE
 ******************************************/

// Render generates the string value for this User
func (w User) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer)

	if status.Error != nil {
		return "", derp.Report(derp.Wrap(status.Error, "render.User.Render", "Error generating HTML"))
	}

	// Success!
	status.Apply(w._context)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this User
func (w User) View(actionID string) (template.HTML, error) {

	renderer, err := NewUser(w._factory, w._context, w._template, w.user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "render.User.View", "Error creating renderer")
	}

	return renderer.Render()
}

func (w User) NavigationID() string {
	return "admin"
}

func (w User) Token() string {
	return "users"
}

func (w User) PageTitle() string {
	return "Settings"
}

func (w User) Permalink() string {
	return ""
}

func (w User) object() data.Object {
	return w.user
}

func (w User) objectID() primitive.ObjectID {
	return w.user.UserID
}

func (w User) objectType() string {
	return "User"
}

func (w User) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

func (w User) service() service.ModelService {
	return w._factory.User()
}

func (w User) executeTemplate(writer io.Writer, name string, data any) error {
	return w._template.HTMLTemplate.ExecuteTemplate(writer, name, data)
}

func (w User) clone(action string) (Renderer, error) {
	return NewUser(w._factory, w._context, w._template, w.user, action)
}

/******************************************
 * Domain Data
 ******************************************/

func (w User) SignupForm() model.SignupForm {
	return w._factory.Domain().Get().SignupForm
}

/******************************************
 * User Data
 ******************************************/

func (w User) UserID() string {
	return w.user.UserID.Hex()
}

func (w User) Label() string {
	return w.user.DisplayName
}

func (w User) DisplayName() string {
	return w.user.DisplayName
}

func (w User) ImageURL() string {
	return w.user.ActivityPubAvatarURL()
}

/******************************************
 * Query Builders
 ******************************************/

func (w User) Users() *QueryBuilder[model.UserSummary] {

	query := builder.NewBuilder().
		String("displayName").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(w._context.Request().URL.Query()),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.UserSummary](w._factory.User(), criteria)

	return &result
}

/******************************************
 * ADDITIONAL DATA
 ******************************************/

// AssignedGroups lists all groups to which the current user is assigned.
func (w User) AssignedGroups() ([]model.Group, error) {
	groupService := w.factory().Group()
	result, err := groupService.ListByIDs(w.user.GroupIDs...)

	return result, derp.Wrap(err, "render.User.AssignedGroups", "Error listing groups", w.user.GroupIDs)
}

func (w User) debug() {
	spew.Dump("User", w.object())
}
