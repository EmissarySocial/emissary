package build

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Syndication wraps this Domain's syndication settings for display in an admin template
type Syndication struct {
	_domain *model.Domain

	CommonWithTemplate
}

// NewSyndication returns a fully initialized `Syndication` builder.
func NewSyndication(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, template model.Template, actionID string) (Syndication, error) {

	const location = "build.NewSyndication"

	// Find/Create new database record for the domain.
	domain := factory.Domain().Get()

	// Create the common Builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, domain, actionID)

	if err != nil {
		return Syndication{}, derp.Wrap(err, location, "Creating common builder")
	}

	// Verify that the user is a Syndication Owner
	if !common._authorization.DomainOwner {
		return Syndication{}, derp.Forbidden(location, "Must be domain owner to continue")
	}

	// Create and return the Syndication builder
	result := Syndication{
		CommonWithTemplate: common,
		_domain:            domain,
	}

	return result, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Stream
func (w Syndication) Render() (template.HTML, error) {

	const location = "build.Syndication.Render"

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, location, "Generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Syndication) View(actionID string) (template.HTML, error) {

	const location = "build.Syndication.View"

	builder, err := NewSyndication(w._factory, w._session, w._request, w._response, w._template, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Creating Group builder")
	}

	return builder.Render()
}

// Token returns the URL token of the record being built. Implements the Builder interface.
func (w Syndication) Token() string {
	return list.Second(w.PathList())
	// return w.context().Param("param1")
}

// object returns the model object being built. Implements the Builder interface.
func (w Syndication) object() data.Object {
	return w._domain
}

// objectID returns the unique ID of the object being built. Implements the Builder interface.
func (w Syndication) objectID() primitive.ObjectID {
	return w._domain.DomainID
}

// objectType returns the name of the model type being built. Implements the Builder interface.
func (w Syndication) objectType() string {
	return "Domain"
}

// schema returns the rosetta schema that validates this object. Implements the Builder interface.
func (w Syndication) schema() schema.Schema {
	return schema.New(model.DomainSchema())
}

// service returns the ModelService that backs this Builder. Implements the Builder interface.
func (w Syndication) service() service.ModelService {
	return w._factory.Domain()
}

// NavigationID returns the top-level navigation item to highlight. Implements the Builder interface.
func (w Syndication) NavigationID() string {
	return "admin"
}

// Permalink returns the permanent URL of the record being built. Implements the Builder interface.
func (w Syndication) Permalink() string {
	return w.Host() + "/admin/syndication"
}

// BasePath returns the URL path of this object, without any action. Implements the Builder interface.
func (w Syndication) BasePath() string {
	return "/admin/syndication"
}

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w Syndication) PageTitle() string {
	return "Settings"
}

// clone returns a copy of this Builder, pointed at a different action. Implements the Builder interface.
func (w Syndication) clone(action string) (Builder, error) {
	return NewSyndication(w._factory, w._session, w._request, w._response, w._template, action)
}

// debug writes this Builder's state to the console. Implements the Builder interface.
func (w Syndication) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_syndication")
}

/******************************************
 * Other Data Accessors
 ******************************************/

// IsAdminBuilder returns TRUE because Syndication is an admin route.
func (w Syndication) IsAdminBuilder() bool {
	return true
}
