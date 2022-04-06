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
	Common
}

func NewDomain(factory Factory, ctx *steranko.Context, layout *model.Layout, actionID string) (Domain, error) {

	const location = "render.NewDomain"

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return Domain{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	// Verify the requested action
	action := layout.Action(actionID)

	if action == nil {
		return Domain{}, derp.NewBadRequestError(location, "Invalid action", actionID)
	}

	result := Domain{
		layout: layout,
		Common: NewCommon(factory, ctx, action, actionID),
	}

	// Pre-load the domain record
	// TODO: This is clunky.
	if _, err := result.getDomain(); err != nil {
		return result, derp.Wrap(err, location, "Error loading Domain")
	}

	return result, nil
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// Action returns the model.Action configured into this renderer
func (w Domain) Action() *model.Action {
	return w.action
}

// Render generates the string value for this Stream
func (w Domain) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer); err != nil {
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

// SignupForm returns the SignupForm associated with this Domain.
func (w Domain) SignupForm() model.SignupForm {

	return w.domain.SignupForm
}

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (w Domain) AdminSections() []model.Option {
	return AdminSections()
}
