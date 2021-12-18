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

type TopLevel struct {
	layout   model.Layout
	actionID string
	Common
}

func NewTopLevel(factory Factory, ctx *steranko.Context, actionID string) TopLevel {

	layoutService := factory.Layout()
	layout := layoutService.TopLevel()

	return TopLevel{
		layout:   layout,
		actionID: actionID,
		Common:   NewCommon(factory, ctx),
	}
}

/*******************************************
 * PATH INTERFACE
 * (not available via templates)
 *******************************************/

func (domain *TopLevel) GetPath(p path.Path) (interface{}, error) {
	return domain.domain.GetPath(p)
}

func (domain *TopLevel) SetPath(p path.Path, value interface{}) error {
	return domain.domain.SetPath(p, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the name of the action being performed
func (domain TopLevel) ActionID() string {
	return domain.actionID
}

// Action returns the model.Action configured into this renderer
func (domain TopLevel) Action() (model.Action, bool) {
	return domain.layout.Action(domain.actionID)
}

// Render generates the string value for this Stream
func (domain TopLevel) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	if action, ok := domain.layout.Action(domain.actionID); ok {

		// Execute step (write HTML to buffer, update context)
		if err := DoPipeline(domain.factory, &domain, &buffer, action.Steps, ActionMethodGet); err != nil {
			return "", derp.Report(derp.Wrap(err, "ghost.render.Stream.Render", "Error generating HTML"))
		}
	}
	// Success!
	return template.HTML(buffer.String()), nil
}

func (domain TopLevel) Token() string {
	return domain.ctx.Param("param1")
}

func (domain TopLevel) TopLevelID() string {
	return "admin"
}

func (domain TopLevel) object() data.Object {
	return nil
}

func (domain TopLevel) schema() schema.Schema {
	return domain.domain.Schema()
}

func (domain TopLevel) common() Common {
	return domain.Common
}

func (domain TopLevel) executeTemplate(wr io.Writer, name string, data interface{}) error {
	return domain.layout.HTMLTemplate.ExecuteTemplate(wr, name, data)
}

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (domain TopLevel) AdminSections() []model.Option {
	return AdminSections()
}
