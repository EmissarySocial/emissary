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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TopLevel struct {
	layout *model.Layout
	stream *model.Stream
	Common
}

func NewTopLevel(factory Factory, ctx *steranko.Context, stream *model.Stream, actionID string) (TopLevel, error) {

	const location = "render.NewGroup"

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return TopLevel{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	layout := factory.Layout().TopLevel()

	// Verify the requested action
	action := layout.Action(actionID)

	if action == nil {
		return TopLevel{}, derp.NewBadRequestError(location, "Invalid action", actionID)
	}

	return TopLevel{
		layout: layout,
		stream: stream,
		Common: NewCommon(factory, ctx, nil, action, actionID),
	}, nil
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// Render generates the string value for this Stream
func (w TopLevel) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer); err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Stream.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w TopLevel) View(actionID string) (template.HTML, error) {

	const location = "render.TopLevel.View"

	renderer, err := NewTopLevel(w.factory(), w.context(), w.stream, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating Group renderer")
	}

	return renderer.Render()
}

func (w TopLevel) Token() string {
	return w._context.Param("param1")
}

func (w TopLevel) TopLevelID() string {
	return "admin"
}

func (w TopLevel) PageTitle() string {
	return "Settings"
}

func (w TopLevel) Permalink() string {
	return ""
}

func (w TopLevel) object() data.Object {
	return w.stream
}

func (w TopLevel) objectID() primitive.ObjectID {
	return w.stream.StreamID
}

func (w TopLevel) objectType() string {
	return "Stream"
}

func (w TopLevel) schema() schema.Schema {
	return w.layout.Schema
}

func (w TopLevel) service() service.ModelService {
	return w._factory.Stream()
}

func (w TopLevel) executeTemplate(wr io.Writer, name string, data any) error {
	return w.layout.HTMLTemplate.ExecuteTemplate(wr, name, data)
}
