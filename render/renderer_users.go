package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/exp/builder"
	"github.com/benpate/ghost/model"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
)

type User struct {
	user     *model.User
	layout   *model.Layout
	actionID string
	Common
}

func NewUser(factory Factory, ctx *steranko.Context, user *model.User, actionID string) User {

	layoutService := factory.Layout()
	layout := layoutService.User()

	return User{
		user:     user,
		layout:   layout,
		actionID: actionID,
		Common:   NewCommon(factory, ctx),
	}
}

/*******************************************
 * PATH INTERFACE
 * (not available via templates)
 *******************************************/

func (w *User) GetPath(p path.Path) (interface{}, error) {
	return w.object().GetPath(p)
}

func (w *User) SetPath(p path.Path, value interface{}) error {
	return w.object().SetPath(p, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the unique ID of the Action configured into this renderer
func (w User) ActionID() string {
	return w.actionID
}

// Action returns the model.Action configured into this renderer
func (w User) Action() (model.Action, bool) {
	return w.layout.Action(w.ActionID())
}

// Render generates the string value for this User
func (w User) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	if action, ok := w.layout.Action(w.actionID); ok {

		// Execute step (write HTML to buffer, update context)
		if err := DoPipeline(&w, &buffer, action.Steps, ActionMethodGet); err != nil {
			return "", derp.Report(derp.Wrap(err, "ghost.render.User.Render", "Error generating HTML"))
		}
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this User
func (w User) View(actionID string) (template.HTML, error) {
	return NewUser(w.factory(), w.ctx, w.user, actionID).Render()
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

	return result, derp.Wrap(err, "ghost.render.User.AssignedGroups", "Error listing groups", w.user.GroupIDs)
}
