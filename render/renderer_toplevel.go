package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TopLevel struct {
	layout   *model.Layout
	stream   *model.Stream
	actionID string
	Common
}

func NewTopLevel(factory Factory, ctx *steranko.Context, layout *model.Layout, objectID string, actionID string) (TopLevel, error) {

	result := TopLevel{
		layout:   layout,
		actionID: actionID,
		Common:   NewCommon(factory, ctx),
	}

	if streamID, err := primitive.ObjectIDFromHex(objectID); err == nil {

		streamService := factory.Stream()
		result.stream = new(model.Stream)
		if err := streamService.LoadTopLevelByID(streamID, result.stream); err != nil {
			return TopLevel{}, derp.Wrap(err, "ghost.render.NewTopLevel", "Error loading Top-Level record", streamID)
		}

		result.actionID = actionID

	} else {
		result.actionID = objectID
	}

	if result.actionID == "" {
		result.actionID = "index"
	}

	return result, nil
}

/*******************************************
 * PATH INTERFACE
 *******************************************/

func (w *TopLevel) GetPath(p path.Path) (interface{}, error) {
	return w.stream.GetPath(p)
}

func (w *TopLevel) SetPath(p path.Path, value interface{}) error {
	return w.stream.SetPath(p, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the name of the action being performed
func (w TopLevel) ActionID() string {
	return w.actionID
}

// Action returns the model.Action configured into this renderer
func (w TopLevel) Action() (model.Action, bool) {
	return w.layout.Action(w.actionID)
}

// Render generates the string value for this Stream
func (w TopLevel) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	if action, ok := w.layout.Action(w.actionID); ok {

		// Execute step (write HTML to buffer, update context)
		if err := DoPipeline(w.factory, &w, &buffer, action.Steps, ActionMethodGet); err != nil {
			return "", derp.Report(derp.Wrap(err, "ghost.render.Stream.Render", "Error generating HTML"))
		}
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

func (w TopLevel) schema() schema.Schema {
	return w.domain.Schema()
}

func (w TopLevel) common() Common {
	return w.Common
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
