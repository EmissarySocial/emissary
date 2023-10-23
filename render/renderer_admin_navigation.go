package render

import (
	"bytes"
	"html/template"
	"io"
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

// Navigation is a renderer for the admin/navigation page
// It can only be accessed by a Domain Owner
type Navigation struct {
	_stream *model.Stream
	Common
}

// NewNavigation returns a fully initialized `Navigation` renderer.
func NewNavigation(factory Factory, request *http.Request, response http.ResponseWriter, template model.Template, stream *model.Stream, actionID string) (Navigation, error) {

	const location = "render.NewGroup"

	// Create the underlying Common renderer
	common, err := NewCommon(factory, request, response, template, actionID)

	if err != nil {
		return Navigation{}, derp.Wrap(err, location, "Error creating common renderer")
	}

	// Verify that the user is a Domain Owner
	if !common._authorization.DomainOwner {
		return Navigation{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	// Return the Navigation renderer
	return Navigation{
		Common: common,
	}, nil
}

/******************************************
 * RENDERER INTERFACE
 ******************************************/

// Render generates the string value for this Stream
func (w Navigation) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "render.Navigation.Render", "Error generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Navigation) View(actionID string) (template.HTML, error) {

	const location = "render.Navigation.View"

	renderer, err := NewNavigation(w.factory(), w._request, w._response, w._template, w._stream, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating Group renderer")
	}

	return renderer.Render()
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

func (w Navigation) executeTemplate(wr io.Writer, name string, data any) error {
	return w._template.HTMLTemplate.ExecuteTemplate(wr, name, data)
}

func (w Navigation) clone(action string) (Renderer, error) {
	return NewNavigation(w._factory, w._request, w._response, w._template, w._stream, action)
}

func (w Navigation) debug() {
	log.Debug().Interface("object", w.object()).Msg("renderer_admin_avigation")
}
