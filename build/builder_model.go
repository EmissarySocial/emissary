package build

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Model builds objects from any model service that implements the ModelService interface
type Model struct {
	_object  model.AccessLister
	_service service.ModelService
	CommonWithTemplate
}

// NewModel returns a fully initialized `Model` builder.
func NewModel(factory Factory, request *http.Request, response http.ResponseWriter, template model.Template, object model.AccessLister, actionID string) (Model, error) {

	const location = "build.NewModel"

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, request, response, template, object, actionID)

	if err != nil {
		return Model{}, derp.Wrap(err, location, "Error creating common builder")
	}

	// Enforce permissions on the requested action
	if !common.UserCan(actionID) {
		if common._authorization.IsAuthenticated() {
			return Model{}, derp.ForbiddenError(location, "Forbidden", "User is authenticated, but this action is not allowed", actionID)
		} else {
			return Model{}, derp.UnauthorizedError(location, "Anonymous user is not authorized to perform this action", actionID)
		}
	}

	// Retrieve the correct service to use for this Model object
	modelService := factory.ModelService(object)

	if modelService == nil {
		return Model{}, derp.InternalError(location, "Invalid model service", object)
	}

	// Return the Model builder
	return Model{
		_object:            object,
		_service:           modelService,
		CommonWithTemplate: common,
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

func (w Model) Object() any {
	return w._object
}

func (w Model) ObjectID() string {
	return w._object.ID()
}

func (w Model) Name() string {
	return w.Label()
}

func (w Model) Label() string {
	switch typed := w._object.(type) {

	case *model.Circle:
		return typed.Name

	case *model.Folder:
		return typed.Label

	case *model.Follower:
		return typed.Actor.Name

	case *model.Following:
		return typed.Label

	case *model.Identity:
		return typed.Name

	case *model.Rule:
		return typed.Label

	case *model.Stream:
		return typed.Label

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
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Model.Render", "Error generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Stream
func (w Model) View(actionID string) (template.HTML, error) {

	const location = "build.Stream.View"

	// Create a new builder (this will also validate the user's permissions)
	subStream, err := NewModel(w._factory, w._request, w._response, w._template, w._object, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating sub-builder")
	}

	// Generate HTML template
	return subStream.Render()
}

/******************************************
 * Custom Queries
 * (may only apply to certain model objects)
 ******************************************/

func (w Model) Identity(identityID primitive.ObjectID) (model.Identity, error) {

	const location = "build.Model.Identity"

	// User must be signed in to view Identities
	if !w._authorization.IsAuthenticated() {
		return model.Identity{}, derp.UnauthorizedError(location, "Anonymous user is not authorized to perform this action", identityID)
	}

	// Load the Identity from the database
	identity := model.NewIdentity()

	if err := w.factory().Identity().LoadByID(identityID, &identity); err != nil {
		return model.Identity{}, derp.Wrap(err, location, "Error loading identity by token")
	}

	// Everything is groovy!
	return identity, nil
}

// CircleMembers returns a QueryBuilder for Circle Members
// in the current Circle (only works on Circle objects)
func (w Model) CircleMembers() (QueryBuilder[model.Identity], error) {

	const location = "build.Model.CircleMembers"

	// Guarantee that we are working with a Circle model object
	circle, isCircle := w._object.(*model.Circle)

	if !isCircle {
		return QueryBuilder[model.Identity]{}, derp.InternalError(location, "Builder method `CircleMembers` can only be used within a `with-circle` action.")
	}

	// Define inbound parameters
	expressionBuilder := builder.NewBuilder().
		String("name")

	// Calculate criteria
	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("privileges", circle.CircleID.Hex()),
	)

	// Return the query builder
	return NewQueryBuilder[model.Identity](w._factory.Identity(), criteria), nil
}

/******************************************
 * Helper functions
 ******************************************/

func (w Model) setState(stateID string) error {

	if setter, ok := w._object.(model.StateSetter); ok {
		setter.SetState(stateID)
		return nil
	}

	return derp.InternalError("build.Model.SetState", "Object does not implement model.StateSetter interface", w._object)
}

func (w Model) clone(action string) (Builder, error) {
	return NewModel(w._factory, w._request, w._response, w._template, w._object, action)
}

func (w Model) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Model")
}
