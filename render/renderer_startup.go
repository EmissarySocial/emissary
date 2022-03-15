package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Startup struct {
	layout *model.Layout
	action *model.Action
	domain *model.Domain
	Common
}

func NewStartup(factory Factory, ctx *steranko.Context, layout *model.Layout, action *model.Action) Startup {

	return Startup{
		layout: layout,
		action: action,
		Common: NewCommon(factory, ctx),
	}
}

/*******************************************
 * PATH INTERFACE
 * (not available via templates)
 *******************************************/

func (w Startup) GetPath(name string) (interface{}, bool) {
	return path.GetOK(w.domain, name)
}

func (w Startup) SetPath(name string, value interface{}) error {
	return path.Set(w.domain, name, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the name of the action being performed
func (w Startup) ActionID() string {
	return w.action.ActionID
}

// Action returns the model.Action configured into this renderer
func (w Startup) Action() *model.Action {
	return w.action
}

// Render generates the string value for this Stream
func (w Startup) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := DoPipeline(&w, &buffer, w.action.Steps, ActionMethodGet); err != nil {
		return "", derp.Report(derp.Wrap(err, "whisper.render.Stream.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

func (w Startup) Token() string {
	return w.context().Param("param1")
}

func (w Startup) object() data.Object {
	return w.domain
}

func (w Startup) objectID() primitive.ObjectID {
	return w.domain.DomainID
}

func (w Startup) schema() schema.Schema {
	return w.domain.Schema()
}

func (w Startup) service() ModelService {
	return w.f.Domain()
}

func (w Startup) executeTemplate(wr io.Writer, name string, data interface{}) error {
	return w.layout.HTMLTemplate.ExecuteTemplate(wr, name, data)
}

func (w Startup) TopLevelID() string {
	return "admin"
}

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (w Startup) AdminSections() []model.Option {
	return AdminSections()
}
