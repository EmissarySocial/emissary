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

type Group struct {
	group *model.Group
	Common
}

func NewGroup(factory Factory, ctx *steranko.Context, template model.Template, group *model.Group, actionID string) (Group, error) {

	const location = "render.NewGroup"

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return Group{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	// Create the underlying Common renderer
	common, err := NewCommon(factory, ctx, template, actionID)

	if err != nil {
		return Group{}, derp.Wrap(err, location, "Error creating common renderer")
	}

	// Return the Group renderer
	return Group{
		group:  group,
		Common: common,
	}, nil
}

/******************************************
 * RENDERER INTERFACE
 ******************************************/

// Render generates the string value for this Stream
func (w Group) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w.action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		return "", derp.Report(derp.Wrap(status.Error, "render.Group.Render", "Error generating HTML"))
	}

	// Success!
	status.Apply(w._context)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Group) View(actionID string) (template.HTML, error) {

	const location = "render.Group.View"

	renderer, err := NewGroup(w._factory, w.context(), w._template, w.group, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating Group renderer")
	}

	return renderer.Render()
}

func (w Group) NavigationID() string {
	return "admin"
}

func (w Group) Permalink() string {
	return w.Hostname() + "/groups/" + w.GroupID()
}

func (w Group) Token() string {
	return "groups"
}

func (w Group) PageTitle() string {
	return "Settings"
}

func (w Group) object() data.Object {
	return w.group
}

func (w Group) objectID() primitive.ObjectID {
	return w.group.GroupID
}

func (w Group) objectType() string {
	return "Group"
}

func (w Group) schema() schema.Schema {
	return schema.New(model.GroupSchema())
}

func (w Group) service() service.ModelService {
	return w._factory.Group()
}

func (w Group) executeTemplate(writer io.Writer, name string, data any) error {
	return w._template.HTMLTemplate.ExecuteTemplate(writer, name, data)
}

func (w Group) clone(action string) (Renderer, error) {
	return NewGroup(w._factory, w.context(), w._template, w.group, action)
}

/******************************************
 * DATA ACCESSORS
 ******************************************/

func (w Group) GroupID() string {
	return w.group.GroupID.Hex()
}

func (w Group) Label() string {
	return w.group.Label
}

/******************************************
 * QUERY BUILDERS
 ******************************************/

func (w Group) Groups() *QueryBuilder[model.Group] {

	query := builder.NewBuilder().
		String("label").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(w.context().Request().URL.Query()),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.Group](w._factory.Group(), criteria)

	return &result
}

func (service Group) debug() {
	spew.Dump("Group", service.object())
}
