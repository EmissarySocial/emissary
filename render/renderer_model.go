package render

import (
	"bytes"
	"html/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Model struct {
	_service service.ModelService
	_object  data.Object
	Common
}

func NewModel(factory Factory, ctx *steranko.Context, modelService service.ModelService, object data.Object, template *model.Template, actionID string) (Model, error) {

	const location = "render.NewModel"

	action := template.Action(actionID)

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	// Check permissions on the InboxFolder
	if roleStateEnumerator, ok := object.(model.RoleStateEnumerator); !ok {
		return Model{}, derp.NewBadRequestError(location, "Object does not implement model.RoleStateEnumerator", object)
	} else {
		if !action.UserCan(roleStateEnumerator, &authorization) {
			if authorization.IsAuthenticated() {
				return Model{}, derp.NewForbiddenError(location, "Forbidden")
			} else {
				return Model{}, derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action", actionID)
			}
		}
	}

	return Model{
		_service: modelService,
		_object:  object,
		Common:   NewCommon(factory, ctx, template, action, actionID),
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

	case *model.Stream:
		return object.Document.Label

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

func (w Model) UserCan(string) bool {
	return false
}

func (w Model) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer); err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Stream.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Stream
func (w Model) View(actionID string) (template.HTML, error) {

	const location = "render.Stream.View"

	// Create a new renderer (this will also validate the user's permissions)
	subStream, err := NewModel(w._factory, w._context, w._service, w._object, w._template, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating sub-renderer")
	}

	// Generate HTML template
	return subStream.Render()
}
