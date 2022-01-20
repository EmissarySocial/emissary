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

type Domain struct {
	layout *model.Layout
	action *model.Action
	domain *model.Domain
	Common
}

func NewDomain(factory Factory, ctx *steranko.Context, layout *model.Layout, action *model.Action) Domain {

	return Domain{
		layout: layout,
		action: action,
		Common: NewCommon(factory, ctx),
	}
}

/*******************************************
 * PATH INTERFACE
 * (not available via templates)
 *******************************************/

func (w Domain) GetPath(path string) (interface{}, bool) {
	return w.domain.GetPath(path)
}

func (w Domain) SetPath(path string, value interface{}) error {
	return w.domain.SetPath(path, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the name of the action being performed
func (w Domain) ActionID() string {
	return w.action.ActionID
}

// Action returns the model.Action configured into this renderer
func (w Domain) Action() *model.Action {
	return w.action
}

// Render generates the string value for this Stream
func (w Domain) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := DoPipeline(&w, &buffer, w.action.Steps, ActionMethodGet); err != nil {
		return "", derp.Report(derp.Wrap(err, "whisper.render.Stream.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

func (w Domain) Token() string {
	return w.context().Param("param1")
}

func (w Domain) object() data.Object {
	return w.domain
}

func (w Domain) objectID() primitive.ObjectID {
	return w.domain.DomainID
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
