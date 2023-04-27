package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Domain struct {
	externalService *service.Provider
	domain          *model.Domain
	Common
}

func NewDomain(factory Factory, ctx *steranko.Context, externalService *service.Provider, template model.Template, domain *model.Domain, actionID string) (Domain, error) {

	const location = "render.NewDomain"

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return Domain{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	// Create the underlying common renderer
	common, err := NewCommon(factory, ctx, template, actionID)

	if err != nil {
		return Domain{}, derp.Wrap(err, location, "Error creating common renderer")
	}

	// Create and return the Domain renderer
	result := Domain{
		externalService: externalService,
		Common:          common,
	}

	result.domain = domain
	return result, nil
}

/******************************************
 * RENDERER INTERFACE
 ******************************************/

// Render generates the string value for this Stream
func (w Domain) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w._factory, &w, &buffer); err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Stream.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Domain) View(actionID string) (template.HTML, error) {

	const location = "render.Domain.View"

	renderer, err := NewDomain(w._factory, w._context, w.externalService, w._template, w.domain, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating Group renderer")
	}

	return renderer.Render()
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

func (w Domain) objectType() string {
	return "Domain"
}

func (w Domain) schema() schema.Schema {
	return w._template.Schema
}

func (w Domain) service() service.ModelService {
	return w._factory.Domain()
}

func (w Domain) domainService() *service.Domain {
	return w._factory.Domain()
}

func (w Domain) executeTemplate(wr io.Writer, name string, data any) error {
	return w._template.HTMLTemplate.ExecuteTemplate(wr, name, data)
}

func (w Domain) NavigationID() string {
	return "admin"
}

func (w Domain) Permalink() string {
	return ""
}

func (w Domain) PageTitle() string {
	return "Settings"
}

/******************************************
 * Other Data Accessors
 ******************************************/

func (w Domain) ThemeID() string {
	return w.domain.ThemeID
}

// SignupForm returns the SignupForm associated with this Domain.
func (w Domain) SignupForm() model.SignupForm {
	return w.domain.SignupForm
}

/******************************************
 * OTHER METHODS
 ******************************************/

func (w Domain) Providers() []form.LookupCode {

	providers := w._factory.Providers()

	return slice.Filter(dataset.Providers(), func(lookupCode form.LookupCode) bool {
		if lookupCode.Group == "MANUAL" {
			return true
		}

		provider, _ := providers.Get(lookupCode.Value)
		return !provider.IsEmpty()
	})
}

func (w Domain) Client(providerID string) model.Client {

	if connection, ok := w.domain.Clients.Get(providerID); ok {
		return connection
	}

	return model.NewClient(providerID)
}

func (w Domain) Provider(providerID string) providers.Provider {
	result, _ := w.externalService.GetProvider(providerID)
	return result
}
