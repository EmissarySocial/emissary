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

func (w *Group) GetPath(p path.Path) (interface{}, error) {
	return w.group.GetPath(p)
}

func (w *Group) SetPath(p path.Path, value interface{}) error {
	return w.group.SetPath(p, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the unique ID of the Action configured into this renderer
func (w Group) ActionID() string {
	return w.actionID
}

// Action returns the model.Action configured into this renderer
func (w Group) Action() (model.Action, bool) {
	return w.layout.Action(w.ActionID())
}

// Render generates the string value for this Stream
func (w Group) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	if action, ok := w.layout.Action(w.actionID); ok {

		// Execute step (write HTML to buffer, update context)
		if err := DoPipeline(&w, &buffer, action.Steps, ActionMethodGet); err != nil {
			return "", derp.Report(derp.Wrap(err, "ghost.render.Stream.Render", "Error generating HTML"))
		}
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Group) View(actionID string) (template.HTML, error) {
	return NewGroup(w.factory(), w.context(), w.group, actionID).Render()
}

func (w Group) TopLevelID() string {
	return "admin"
}

func (w Group) Token() string {
	return "groups"
}

func (w Group) object() data.Object {
	return w.group
}

func (w Group) schema() schema.Schema {
	return w.group.Schema()
}

func (w Group) common() Common {
	return w.Common
}

func (w Group) executeTemplate(writer io.Writer, name string, data interface{}) error {
	return w.layout.HTMLTemplate.ExecuteTemplate(writer, name, data)
}

/*******************************************
 * DATA ACCESSORS
 *******************************************/

func (w Group) GroupID() string {
	return w.group.GroupID.Hex()
}

func (w Group) Label() string {
	return w.group.Label
}

/*******************************************
 * QUERY BUILDERS
 *******************************************/

func (w Group) Groups() *QueryBuilder {

	query := builder.NewBuilder().
		String("displayName").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(w.context().Request().URL.Query()),
		exp.Equal("journal.deleteDate", 0),
	)

	result := NewQueryBuilder(w.factory(), w.context(), w.factory().Group(), criteria)

	return &result
}

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (w Group) AdminSections() []model.Option {
	return AdminSections()
}
