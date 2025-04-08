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

type Syndication struct {
	_domain *model.Domain

	CommonWithTemplate
}

// NewSyndication returns a fully initialized `Syndication` builder.
func NewSyndication(factory Factory, request *http.Request, response http.ResponseWriter, template model.Template, actionID string) (Syndication, error) {

	const location = "build.NewSyndication"

	// Create the common Builder
	common, err := NewCommonWithTemplate(factory, request, response, template, actionID)

	if err != nil {
		return Syndication{}, derp.Wrap(err, location, "Error creating common builder")
	}

	// Verify that the user is a Syndication Owner
	if !common._authorization.DomainOwner {
		return Syndication{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	// Create and return the Syndication builder
	result := Syndication{
		CommonWithTemplate: common,
	}

	// Find/Create new database record for the domain.
	domainService := factory.Domain()
	if _, err := domainService.LoadDomain(); err != nil {
		return Syndication{}, derp.Wrap(err, location, "Error creating a new Syndication")
	}

	result._domain = domainService.GetPointer()
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
		err := derp.Wrap(status.Error, location, "Error generating HTML")
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

	builder, err := NewSyndication(w._factory, w._request, w._response, w._template, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating Group builder")
	}

	return builder.Render()
}

func (w Syndication) Token() string {
	return list.Second(w.PathList())
	// return w.context().Param("param1")
}

func (w Syndication) object() data.Object {
	return w._domain
}

func (w Syndication) objectID() primitive.ObjectID {
	return w._domain.DomainID
}

func (w Syndication) objectType() string {
	return "Domain"
}

func (w Syndication) schema() schema.Schema {
	return schema.New(model.DomainSchema())
}

func (w Syndication) service() service.ModelService {
	return w._factory.Domain()
}

func (w Syndication) NavigationID() string {
	return "admin"
}

func (w Syndication) Permalink() string {
	return w.Host() + "/admin/syndication"
}

func (w Syndication) BasePath() string {
	return "/admin/syndication"
}

func (w Syndication) PageTitle() string {
	return "Settings"
}

func (w Syndication) clone(action string) (Builder, error) {
	return NewSyndication(w._factory, w._request, w._response, w._template, action)
}

func (w Syndication) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_syndication")
}

/******************************************
 * Other Data Accessors
 ******************************************/

// IsAdminBuilder returns TRUE because Syndication is an admin route.
func (w Syndication) IsSyndication() bool {
	return false
}
