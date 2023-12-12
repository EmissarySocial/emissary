package render

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Model renders objects from any model service that implements the ModelService interface
type Model struct {
	_service service.ModelService
	_object  data.Object
	Common
}

// NewModel returns a fully initialized `Model` renderer.
func NewModel(factory Factory, request *http.Request, response http.ResponseWriter, modelService service.ModelService, object data.Object, template model.Template, actionID string) (Model, error) {

	const location = "render.NewModel"

	action, ok := template.Action(actionID)

	if !ok {
		return Model{}, derp.NewBadRequestError(location, "Invalid action", actionID)
	}

	// Create the underlying Common renderer
	common, err := NewCommon(factory, request, response, template, actionID)

	if err != nil {
		return Model{}, derp.Wrap(err, location, "Error creating common renderer")
	}

	// Check permissions on this model object
	if roleStateEnumerator, ok := object.(model.RoleStateEnumerator); !ok {
		return Model{}, derp.NewBadRequestError(location, "Object does not implement model.RoleStateEnumerator", object)

	} else if !action.UserCan(roleStateEnumerator, &common._authorization) {

		if common._authorization.IsAuthenticated() {
			return Model{}, derp.NewForbiddenError(location, "Forbidden")
		} else {
			return Model{}, derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action", actionID)
		}
	}

	// Return the Model renderer
	return Model{
		_service: modelService,
		_object:  object,
		Common:   common,
	}, nil
}

func (w Model) object() data.Object {
	return w._object
}

func (w Model) objectType() string {
	return w._service.ObjectType()
}

func (w Model) objectID() primitive.ObjectID {
	return w._service.ObjectID(w._object)
}

func (w Model) schema() schema.Schema {
	return w._service.Schema()
}

func (w Model) service() service.ModelService {
	return w._service
}

func (w Model) ObjectID() string {
	return w._object.ID()
}

func (w Model) Label() string {
	switch object := w._object.(type) {

	case *model.Folder:
		return object.Label

	case *model.Following:
		return object.Label

	case *model.Follower:
		return object.Actor.Name

	case *model.Stream:
		return object.Label

	default:
		return ""
	}
}

func (w Model) Token() string {
	return ""
}

func (w Model) PageTitle() string {
	return ""
}

func (w Model) Permalink() string {
	return ""
}

func (w Model) BasePath() string {
	return ""
}

func (w Model) UserCan(string) bool {
	return false
}

func (w Model) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w.action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "render.Stream.Render", "Error generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Stream
func (w Model) View(actionID string) (template.HTML, error) {

	const location = "render.Stream.View"

	// Create a new renderer (this will also validate the user's permissions)
	subStream, err := NewModel(w._factory, w._request, w._response, w._service, w._object, w._template, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating sub-renderer")
	}

	// Generate HTML template
	return subStream.Render()
}

func (w Model) clone(action string) (Renderer, error) {
	return NewModel(w._factory, w._request, w._response, w._service, w._object, w._template, action)
}

func (w Model) debug() {
	log.Debug().Interface("object", w.object()).Msg("renderer_Model")
}
