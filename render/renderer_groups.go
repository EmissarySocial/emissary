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
	group  *model.Group
	Common
}

func NewGroup(factory Factory, ctx *steranko.Context, group *model.Group, actionID string) (Group, error) {

	const location = "render.NewGroup"

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return Group{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	layout := factory.Layout().Group()

	// Verify the requested action
	action := layout.Action(actionID)

	if action == nil {
		return Group{}, derp.NewBadRequestError(location, "Invalid action", actionID)
	}

	return Group{
		group:  group,
		layout: layout,
		Common: NewCommon(factory, ctx, action, actionID),
	}, nil
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// Action returns the model.Action configured into this renderer
func (w Group) Action() *model.Action {
	return w.action
}

// Render generates the string value for this Stream
func (w Group) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer); err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Group.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Group) View(actionID string) (template.HTML, error) {

	const location = "render.Group.View"

	renderer, err := NewGroup(w.factory(), w.context(), w.group, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating Group renderer")
	}

	return renderer.Render()
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

func (w Group) executeTemplate(writer io.Writer, name string, data any) error {
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
