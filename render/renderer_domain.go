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
)

type Domain struct {
	domain   model.Domain
	layout   *model.Layout
	actionID string
	Common
}

func NewDomain(factory Factory, ctx *steranko.Context, layout *model.Layout, actionID string) Domain {

	return Domain{
		layout:   layout,
		actionID: actionID,
		Common:   NewCommon(factory, ctx),
	}
}

/*******************************************
 * PATH INTERFACE
 * (not available via templates)
 *******************************************/

func (w *Domain) GetPath(p path.Path) (interface{}, error) {
	return w.domain.GetPath(p)
}

func (w *Domain) SetPath(p path.Path, value interface{}) error {
	return w.domain.SetPath(p, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the name of the action being performed
func (w Domain) ActionID() string {
	return w.actionID
}

// Action returns the model.Action configured into this renderer
func (w Domain) Action() (model.Action, bool) {
	return w.layout.Action(w.actionID)
}

// Render generates the string value for this Stream
func (w Domain) Render() (template.HTML, error) {

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

func (w Domain) Token() string {
	return w.context().Param("param1")
}

func (w Domain) object() data.Object {
	return &w.domain
}

func (w Domain) schema() schema.Schema {
	return w.domain.Schema()
}

func (w Domain) service() ModelService {
	return w.f.Domain()
}

func (w Domain) executeTemplate(wr io.Writer, name string, data interface{}) error {
	return w.layout.HTMLTemplate.ExecuteTemplate(wr, name, data)
}

func (w Domain) TopLevelID() string {
	return "admin"
}

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (w Domain) AdminSections() []model.Option {
	return AdminSections()
}
