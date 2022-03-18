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
	action *model.Action
	stream *model.Stream
	Common
}

func NewTopLevel(factory Factory, ctx *steranko.Context, layout *model.Layout, action *model.Action, stream *model.Stream) TopLevel {

	return TopLevel{
		layout: layout,
		action: action,
		stream: stream,
		Common: NewCommon(factory, ctx),
	}
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the name of the action being performed
func (w TopLevel) ActionID() string {
	return w.action.ActionID
}

// Action returns the model.Action configured into this renderer
func (w TopLevel) Action() *model.Action {
	return w.action
}

// Render generates the string value for this Stream
func (w TopLevel) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := DoPipeline(&w, &buffer, w.action.Steps, ActionMethodGet); err != nil {
		return "", derp.Report(derp.Wrap(err, "whisper.render.Stream.Render", "Error generating HTML"))
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
	return w.domain.Schema()
}

func (w TopLevel) service() ModelService {
	return w.f.Stream()
}

func (w TopLevel) executeTemplate(wr io.Writer, name string, data interface{}) error {
	return w.layout.HTMLTemplate.ExecuteTemplate(wr, name, data)
}

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (w TopLevel) AdminSections() []model.Option {
	return AdminSections()
}
