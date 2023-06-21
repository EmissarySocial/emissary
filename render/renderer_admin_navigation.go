package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Navigation struct {
	stream *model.Stream
	Common
}

func NewNavigation(factory Factory, ctx *steranko.Context, template model.Template, stream *model.Stream, actionID string) (Navigation, error) {

	const location = "render.NewGroup"

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return Navigation{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	// Create the underlying Common renderer
	common, err := NewCommon(factory, ctx, template, actionID)

	if err != nil {
		return Navigation{}, derp.Wrap(err, location, "Error creating common renderer")
	}

	// Return the Navigation renderer
	return Navigation{
		stream: stream,
		Common: common,
	}, nil
}

/******************************************
 * RENDERER INTERFACE
 ******************************************/

// Render generates the string value for this Stream
func (w Navigation) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer)

	if status.Error != nil {
		return "", derp.Report(derp.Wrap(status.Error, "render.Navigation.Render", "Error generating HTML"))
	}

	// Success!
	status.Apply(w._context)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Navigation) View(actionID string) (template.HTML, error) {

	const location = "render.Navigation.View"

	renderer, err := NewNavigation(w.factory(), w.context(), w._template, w.stream, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating Group renderer")
	}

	return renderer.Render()
}

func (w Navigation) Token() string {
	return w._context.Param("param1")
}

func (w Navigation) NavigationID() string {
	return "admin"
}

func (w Navigation) PageTitle() string {
	return "Settings"
}

func (w Navigation) Permalink() string {
	return ""
}

func (w Navigation) object() data.Object {
	return w.stream
}

func (w Navigation) objectID() primitive.ObjectID {
	return w.stream.StreamID
}

func (w Navigation) objectType() string {
	return "Stream"
}

func (w Navigation) schema() schema.Schema {
	return w._template.Schema
}

func (w Navigation) service() service.ModelService {
	return w._factory.Stream()
}

func (w Navigation) executeTemplate(wr io.Writer, name string, data any) error {
	return w._template.HTMLTemplate.ExecuteTemplate(wr, name, data)
}

func (w Navigation) clone(action string) (Renderer, error) {
	return NewNavigation(w._factory, w._context, w._template, w.stream, action)
}

func (service Navigation) debug() {
	spew.Dump("Navigation", service.object())
}
