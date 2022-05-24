package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
	"github.com/whisperverse/whisperverse/model"
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
		Common: NewCommon(factory, ctx, action, actionID),
	}, nil
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// Action returns the model.Action configured into this renderer
func (w TopLevel) Action() *model.Action {
	return w.action
}

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

func (w TopLevel) Token() string {
	return w.ctx.Param("param1")
}

func (w TopLevel) TopLevelID() string {
	return "admin"
}

func (w TopLevel) object() data.Object {
	return w.stream
}

func (w TopLevel) objectID() primitive.ObjectID {
	return w.stream.StreamID
}

func (w TopLevel) schema() schema.Schema {
	return w.layout.Schema
}

func (w TopLevel) service() ModelService {
	return w.f.Stream()
}

func (w TopLevel) executeTemplate(wr io.Writer, name string, data any) error {
	return w.layout.HTMLTemplate.ExecuteTemplate(wr, name, data)
}
