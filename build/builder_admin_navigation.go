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

// Navigation is a builder for the admin/navigation page
// It can only be accessed by a Domain Owner
type Navigation struct {
	_stream *model.Stream
	CommonWithTemplate
}

// NewNavigation returns a fully initialized `Navigation` builder.
func NewNavigation(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, template model.Template, stream *model.Stream, actionID string) (Navigation, error) {

	const location = "build.NewGroup"

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, stream, actionID)

	if err != nil {
		return Navigation{}, derp.Wrap(err, location, "Creating common builder")
	}

	// Verify that the user is a Domain Owner
	if !common._authorization.DomainOwner {
		return Navigation{}, derp.Forbidden(location, "Must be domain owner to continue")
	}

	// Return the Navigation builder
	return Navigation{
		_stream:            stream,
		CommonWithTemplate: common,
	}, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Stream
func (w Navigation) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Navigation.Render", "Generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Navigation) View(actionID string) (template.HTML, error) {

	const location = "build.Navigation.View"

	builder, err := NewNavigation(w._factory, w._session, w._request, w._response, w._template, w._stream, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Creating Group builder")
	}

	return builder.Render()
}

// Token returns the URL token of the record being built. Implements the Builder interface.
func (w Navigation) Token() string {
	return list.Second(w.PathList())
}

// NavigationID returns the top-level navigation item to highlight. Implements the Builder interface.
func (w Navigation) NavigationID() string {
	return "admin"
}

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w Navigation) PageTitle() string {
	return "Settings"
}

// Permalink returns the permanent URL of the record being built. Implements the Builder interface.
func (w Navigation) Permalink() string {
	return ""
}

// BasePath returns the URL path of this object, without any action. Implements the Builder interface.
func (w Navigation) BasePath() string {
	return ""
}

// object returns the model object being built. Implements the Builder interface.
func (w Navigation) object() data.Object {
	return w._stream
}

// objectID returns the unique ID of the object being built. Implements the Builder interface.
func (w Navigation) objectID() primitive.ObjectID {
	return w._stream.StreamID
}

// objectType returns the name of the model type being built. Implements the Builder interface.
func (w Navigation) objectType() string {
	return "Stream"
}

// schema returns the rosetta schema that validates this object. Implements the Builder interface.
func (w Navigation) schema() schema.Schema {
	return w._template.Schema
}

// service returns the ModelService that backs this Builder. Implements the Builder interface.
func (w Navigation) service() service.ModelService {
	return w._factory.Stream()
}

// clone returns a copy of this Builder, pointed at a different action. Implements the Builder interface.
func (w Navigation) clone(action string) (Builder, error) {
	return NewNavigation(w._factory, w._session, w._request, w._response, w._template, w._stream, action)
}

// debug writes this Builder's state to the console. Implements the Builder interface.
func (w Navigation) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_avigation")
}

/******************************************
 * Other Data Accessors
 ******************************************/

// IsAdminBuilder returns TRUE because Navigation is an admin route.
func (w Navigation) IsAdminBuilder() bool {
	return true
}

// DefaultPage returns the landing page for a visitor, based on how they are signed in
func (w Navigation) DefaultPage() string {
	domain := w.factory().Domain().Get()
	return domain.DefaultPage(w._authorization)
}

// DefaultPage_Anonymous returns the landing page for a visitor who is not signed in
func (w Navigation) DefaultPage_Anonymous() string {
	domain := w.factory().Domain().Get()
	return domain.DefaultPage_Anonymous()
}

// DefaultPage_Authenticated returns the landing page for a signed-in User
func (w Navigation) DefaultPage_Authenticated() string {
	domain := w.factory().Domain().Get()
	return domain.DefaultPage_Authenticated()
}

// DefaultPage_Owner returns the landing page for a domain owner
func (w Navigation) DefaultPage_Owner() string {
	domain := w.factory().Domain().Get()
	return domain.DefaultPage_Owner()
}
