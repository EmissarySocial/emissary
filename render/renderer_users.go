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
	action *model.Action
	user   *model.User
	Common
}

func NewUser(factory Factory, ctx *steranko.Context, layout *model.Layout, action *model.Action, user *model.User) User {

	return User{
		layout: layout,
		action: action,
		user:   user,
		Common: NewCommon(factory, ctx),
	}
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the unique ID of the Action configured into this renderer
func (w User) ActionID() string {
	return w.action.ActionID
}

// Action returns the model.Action configured into this renderer
func (w User) Action() *model.Action {
	return w.action
}

// Render generates the string value for this User
func (w User) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := DoPipeline(&w, &buffer, w.action.Steps, ActionMethodGet); err != nil {
		return "", derp.Report(derp.Wrap(err, "whisper.render.User.Render", "Error generating HTML"))

	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this User
func (w User) View(actionID string) (template.HTML, error) {

	action := w.layout.Action(actionID)

	return NewUser(w.factory(), w.ctx, w.layout, action, w.user).Render()
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

func (w User) executeTemplate(writer io.Writer, name string, data interface{}) error {
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

func (w User) AvatarURL() string {
	return w.user.AvatarURL
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

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (w User) AdminSections() []model.Option {
	return AdminSections()
}

// AssignedGroups lists all groups to which the current user is assigned.
func (w User) AssignedGroups() ([]model.Group, error) {
	groupService := w.factory().Group()
	result, err := groupService.ListByIDs(w.user.GroupIDs...)

	return result, derp.Wrap(err, "whisper.render.User.AssignedGroups", "Error listing groups", w.user.GroupIDs)
}
