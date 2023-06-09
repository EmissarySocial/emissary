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
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/exp/slices"
)

type Domain struct {
	externalService *service.Provider
	domain          *model.Domain
	Common
}

func NewDomain(factory Factory, ctx *steranko.Context, template model.Template, actionID string) (Domain, error) {

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
		externalService: factory.Provider(),
		Common:          common,
	}

	// Get a pointer to the domain for this renderer
	domainService := factory.Domain()

	// Find/Create new database record for the domain.
	if _, err := domainService.LoadOrCreateDomain(); err != nil {
		return Domain{}, derp.Wrap(err, location, "Error creating a new Domain")
	}

	result.domain = domainService.GetPointer()
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
		return "", derp.Report(derp.Wrap(err, "render.Domain.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Domain) View(actionID string) (template.HTML, error) {

	const location = "render.Domain.View"

	renderer, err := NewDomain(w._factory, w._context, w._template, actionID)

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
	return schema.New(model.DomainSchema())
	// return w._template.Schema
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

func (w Domain) clone(action string) (Renderer, error) {
	return NewDomain(w._factory, w._context, w._template, action)
}

/******************************************
 * Other Data Accessors
 ******************************************/

func (w Domain) ThemeID() string {
	return w.domain.ThemeID
}

func (w Domain) Theme(themeID string) model.Theme {
	themeService := w._factory.Theme()
	return themeService.GetTheme(themeID)
}

// SignupForm returns the SignupForm associated with this Domain.
func (w Domain) SignupForm() model.SignupForm {
	return w.domain.SignupForm
}

/******************************************
 * OTHER METHODS
 ******************************************/

func (w Domain) Themes() []model.Theme {
	themeService := w._factory.Theme()
	result := themeService.List()
	slices.SortFunc(result, func(a, b model.Theme) bool {
		return a.Label < b.Label
	})

	return result
}

func (w Domain) ActiveThemes() []model.Theme {
	return slice.Filter(w.Themes(), func(theme model.Theme) bool {
		return !theme.IsPlaceholder()
	})
}

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
func (service Domain) debug() {
	spew.Dump("Domain", service.object())
}
