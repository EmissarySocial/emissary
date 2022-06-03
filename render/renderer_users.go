package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/exp/builder"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	layout *model.Layout
	user   *model.User
	Common
}

func NewUser(factory Factory, ctx *steranko.Context, user *model.User, actionID string) (User, error) {

	const location = "render.NewGroup"

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return User{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	layout := factory.Layout().User()

	// Verify the requested action
	action := layout.Action(actionID)

	if action == nil {
		return User{}, derp.NewBadRequestError(location, "Invalid action", actionID)
	}

	return User{
		layout: layout,
		user:   user,
		Common: NewCommon(factory, ctx, action, actionID),
	}, nil
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// Render generates the string value for this User
func (w User) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer); err != nil {
		return "", derp.Report(derp.Wrap(err, "render.User.Render", "Error generating HTML"))

	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this User
func (w User) View(actionID string) (template.HTML, error) {

	renderer, err := NewUser(w.factory(), w.ctx, w.user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "render.User.View", "Error creating renderer")
	}

	return renderer.Render()
}

func (w User) TopLevelID() string {
	return "admin"
}

func (w User) Token() string {
	return "users"
}

func (w User) object() data.Object {
	return w.user
}

func (w User) objectID() primitive.ObjectID {
	return w.user.UserID
}

func (w User) schema() schema.Schema {
	return w.user.Schema()
}

func (w User) service() ModelService {
	return w.f.User()
}

func (w User) executeTemplate(writer io.Writer, name string, data any) error {
	return w.layout.HTMLTemplate.ExecuteTemplate(writer, name, data)
}

/*******************************************
 * DATA ACCESSORS
 *******************************************/

func (w User) UserID() string {
	return w.user.UserID.Hex()
}

func (w User) DisplayName() string {
	return w.user.DisplayName
}

func (w User) ImageURL() string {
	return w.user.ImageURL
}

/*******************************************
 * QUERY BUILDERS
 *******************************************/

func (w User) Users() *QueryBuilder {

	query := builder.NewBuilder().
		String("displayName").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(w.ctx.Request().URL.Query()),
		exp.Equal("journal.deleteDate", 0),
	)

	result := NewQueryBuilder(w.factory(), w.ctx, w.factory().User(), criteria)

	return &result
}

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AssignedGroups lists all groups to which the current user is assigned.
func (w User) AssignedGroups() ([]model.Group, error) {
	groupService := w.factory().Group()
	result, err := groupService.ListByIDs(w.user.GroupIDs...)

	return result, derp.Wrap(err, "render.User.AssignedGroups", "Error listing groups", w.user.GroupIDs)
}
