package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
)

type Domain struct {
	domain   model.Domain
	layout   model.Layout
	actionID string
	Common
}

func NewDomain(factory Factory, ctx *steranko.Context, domain model.Domain, actionID string) Domain {

	layoutService := factory.Layout()
	layout := layoutService.Domain()

	return Domain{
		domain:   domain,
		layout:   layout,
		actionID: actionID,
		Common:   NewCommon(factory, ctx),
	}
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
	return domain.domain.ID()
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
