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
	layout   model.Layout
	actionID string
	Common
}

func NewDomain(factory Factory, ctx *steranko.Context, layout model.Layout, actionID string) Domain {

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

func (domain *Domain) GetPath(p path.Path) (interface{}, error) {
	return domain.domain.GetPath(p)
}

func (domain *Domain) SetPath(p path.Path, value interface{}) error {
	return domain.domain.SetPath(p, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the name of the action being performed
func (domain Domain) ActionID() string {
	return domain.actionID
}

// Action returns the model.Action configured into this renderer
func (domain Domain) Action() (model.Action, bool) {
	return domain.layout.Action(domain.actionID)
}

// Render generates the string value for this Stream
func (domain Domain) Render() (template.HTML, error) {

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

func (domain Domain) Token() string {
	return domain.ctx.Param("param1")
}

func (domain Domain) object() data.Object {
	return &domain.domain
}

func (domain Domain) schema() schema.Schema {
	return domain.domain.Schema()
}

func (domain Domain) common() Common {
	return domain.Common
}

func (domain Domain) executeTemplate(wr io.Writer, name string, data interface{}) error {
	return domain.layout.HTMLTemplate.ExecuteTemplate(wr, name, data)
}

func (domain Domain) TopLevelID() string {
	return "admin"
}

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (domain Domain) AdminSections() []model.Option {
	return AdminSections()
}
