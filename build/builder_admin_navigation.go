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
		return Navigation{}, derp.Wrap(err, location, "Error creating common builder")
	}

	// Verify that the user is a Domain Owner
	if !common._authorization.DomainOwner {
		return Navigation{}, derp.ForbiddenError(location, "Must be domain owner to continue")
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
		err := derp.Wrap(status.Error, "build.Navigation.Render", "Error generating HTML")
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
		return template.HTML(""), derp.Wrap(err, location, "Error creating Group builder")
	}

	return builder.Render()
}

func (w Navigation) Token() string {
	return list.Second(w.PathList())
}

func (w Navigation) NavigationID() string {
	return "admin"
}

func (w Navigation) PageTitle() string {
	return "Settings"
}

func (w Navigation) Permalink() string {
	return ""
}

func (w Navigation) BasePath() string {
	return ""
}

func (w Navigation) object() data.Object {
	return w._stream
}

func (w Navigation) objectID() primitive.ObjectID {
	return w._stream.StreamID
}

func (w Navigation) objectType() string {
	return "Stream"
}

func (w Navigation) schema() schema.Schema {
	return w._template.Schema
}

func (w Navigation) service() service.ModelService {
	return w._factory.Stream()
}

func (w Navigation) clone(action string) (Builder, error) {
	return NewNavigation(w._factory, w._session, w._request, w._response, w._template, w._stream, action)
}

func (w Navigation) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_avigation")
}

/******************************************
 * Other Data Accessors
 ******************************************/

// IsAdminBuilder returns TRUE because Navigation is an admin route.
func (w Navigation) IsAdminBuilder() bool {
	return false
}
