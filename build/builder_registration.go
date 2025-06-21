package build

import (
	"bytes"
	"html/template"
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Registration is the builder for the admin/domain page
// It can only be accessed by a Registration Owner
type Registration struct {
	_actionID     string
	_action       model.Action
	_domain       *model.Domain
	_provider     *service.Provider
	_registration *model.Registration
	_user         model.User
	Common
}

// NewRegistration returns a fully initialized `Registration` builder.
func NewRegistration(factory Factory, request *http.Request, response http.ResponseWriter, registration *model.Registration, actionID string) (Registration, error) {

	const location = "build.NewRegistration"

	// Find the Action
	action, ok := registration.Action(actionID)

	if !ok {
		return Registration{}, derp.BadRequestError(location, "Invalid actionID", actionID)
	}

	// Create and return the Registration builder
	result := Registration{
		_actionID:     actionID,
		_action:       action,
		_domain:       factory.Domain().Get(),
		_provider:     factory.Provider(),
		_registration: registration,
		_user:         model.NewUser(),

		Common: NewCommon(factory, request, response),
	}

	return result, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Stream
func (w Registration) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Registration.Render", "Error generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Registration) View(actionID string) (template.HTML, error) {

	const location = "build.Registration.View"

	builder, err := NewRegistration(w._factory, w._request, w._response, w._registration, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating Group builder")
	}

	return builder.Render()
}

func (w Registration) Token() string {
	return list.Second(w.PathList())
	// return w.context().Param("param1")
}

func (w Registration) Label() string {
	return w._domain.Label
}

func (w Registration) IconURL() string {
	return w._domain.IconURL()
}

func (w Registration) DomainData() mapof.String {
	return w._domain.RegistrationData
}

func (w Registration) object() data.Object {
	return &w._user
}

func (w Registration) objectID() primitive.ObjectID {
	return w._user.UserID
}

func (w Registration) objectType() string {
	return "User"
}

func (w Registration) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

func (w Registration) service() service.ModelService {
	return w._factory.User()
}

func (w Registration) actions() map[string]model.Action {
	return w._registration.Actions
}

func (w Registration) action() model.Action {
	return w._action
}

func (w Registration) actionID() string {
	return w._actionID
}

func (w Registration) execute(wr io.Writer, name string, data any) error {
	return w._registration.HTMLTemplate.ExecuteTemplate(wr, name, data)
}

func (w Registration) NavigationID() string {
	return "register"
}

func (w Registration) Permalink() string {
	return w.Host() + "/register"
}

func (w Registration) BasePath() string {
	return "/register"
}

func (w Registration) PageTitle() string {
	return "Register"
}

func (w Registration) Data(key string) string {
	return w._domain.Data[key]
}

func (w Registration) ThemeData(key string) string {
	return w._domain.Data[key]
}

func (w Registration) RegistrationData(key string) string {
	return w._domain.RegistrationData[key]
}

func (w Registration) clone(action string) (Builder, error) {
	return NewRegistration(w._factory, w._request, w._response, w._registration, action)
}

/******************************************
 * Registration Methods
 ******************************************/

// Template returns the registration template selected for this domain
func (w Registration) Template() model.Registration {
	domain := w._factory.Domain().Get()
	registration, _ := w._factory.Registration().Load(domain.RegistrationID)
	return registration
}

/******************************************
 * Other Methods
 ******************************************/

// Providers lists all available external services that can be connected to this domain
func (w Registration) Providers() []form.LookupCode {
	return dataset.Providers()
}

// Connection loads an external service connection from the database
func (w Registration) AllConnections() mapof.Object[model.Connection] {
	return w.factory().Connection().AllAsMap()
}

func (w Registration) Provider(providerID string) providers.Provider {
	result, _ := w._provider.GetProvider(providerID)
	return result
}

func (w Registration) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_registration")
}
