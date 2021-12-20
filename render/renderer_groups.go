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

type Group struct {
	layout   *model.Layout
	group    *model.Group
	actionID string
	Common
}

func NewGroup(factory Factory, ctx *steranko.Context, group *model.Group, actionID string) Group {

	layoutService := factory.Layout()
	layout := layoutService.Group()

	return Group{
		group:    group,
		layout:   layout,
		actionID: actionID,
		Common:   NewCommon(factory, ctx),
	}
}

/*******************************************
 * PATH INTERFACE
 * (not available via templates)
 *******************************************/

func (u *Group) GetPath(p path.Path) (interface{}, error) {
	return u.group.GetPath(p)
}

func (u *Group) SetPath(p path.Path, value interface{}) error {
	return u.group.SetPath(p, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the unique ID of the Action configured into this renderer
func (group Group) ActionID() string {
	return group.actionID
}

// Action returns the model.Action configured into this renderer
func (group Group) Action() (model.Action, bool) {
	return group.layout.Action(group.ActionID())
}

// Render generates the string value for this Stream
func (group Group) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	if action, ok := group.layout.Action(group.actionID); ok {

		// Execute step (write HTML to buffer, update context)
		if err := DoPipeline(group.factory, &group, &buffer, action.Steps, ActionMethodGet); err != nil {
			return "", derp.Report(derp.Wrap(err, "ghost.render.Stream.Render", "Error generating HTML"))
		}
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (group Group) View(actionID string) (template.HTML, error) {
	return NewGroup(group.factory, group.ctx, group.group, actionID).Render()
}

func (group Group) TopLevelID() string {
	return "admin"
}

func (group Group) Token() string {
	return "groups"
}

func (group Group) object() data.Object {
	return group.group
}

func (group Group) schema() schema.Schema {
	return group.group.Schema()
}

func (group Group) common() Common {
	return group.Common
}

func (group Group) executeTemplate(writer io.Writer, name string, data interface{}) error {
	return group.layout.HTMLTemplate.ExecuteTemplate(writer, name, data)
}

/*******************************************
 * DATA ACCESSORS
 *******************************************/

func (group Group) GroupID() string {
	return group.group.GroupID.Hex()
}

func (group Group) Label() string {
	return group.group.Label
}

/*******************************************
 * QUERY BUILDERS
 *******************************************/

func (group Group) Groups() *QueryBuilder {

	query := builder.NewBuilder().
		String("displayName").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(group.ctx.Request().URL.Query()),
		exp.Equal("journal.deleteDate", 0),
	)

	result := NewQueryBuilder(group.factory, group.ctx, group.factory.Group(), criteria)

	return &result
}

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (group Group) AdminSections() []model.Option {
	return AdminSections()
}
