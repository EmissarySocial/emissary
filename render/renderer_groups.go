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

type Group struct {
	layout *model.Layout
	action *model.Action
	group  *model.Group
	Common
}

func NewGroup(factory Factory, ctx *steranko.Context, layout *model.Layout, action *model.Action, group *model.Group) Group {

	return Group{
		group:  group,
		layout: layout,
		action: action,
		Common: NewCommon(factory, ctx),
	}
}

/*******************************************
 * PATH INTERFACE
 *******************************************/

func (w Group) GetPath(path string) (interface{}, bool) {
	return w.group.GetPath(path)
}

func (w Group) SetPath(path string, value interface{}) error {
	return w.group.SetPath(path, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the unique ID of the Action configured into this renderer
func (w Group) ActionID() string {
	return w.action.ActionID
}

// Action returns the model.Action configured into this renderer
func (w Group) Action() *model.Action {
	return w.action
}

// Render generates the string value for this Stream
func (w Group) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := DoPipeline(&w, &buffer, w.action.Steps, ActionMethodGet); err != nil {
		return "", derp.Report(derp.Wrap(err, "whisper.render.Group.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Group) View(actionID string) (template.HTML, error) {

	action := w.layout.Action(actionID)

	if action == nil {
		return template.HTML(""), derp.NewNotFoundError("whisper.render.Group.View", "Unrecognized Action", actionID)
	}

	return NewGroup(w.factory(), w.context(), w.layout, action, w.group).Render()
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

func (w Group) objectID() primitive.ObjectID {
	return w.group.GroupID
}

func (w Group) schema() schema.Schema {
	return w.group.Schema()
}

func (w Group) service() ModelService {
	return w.f.Group()
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
		String("label").
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
