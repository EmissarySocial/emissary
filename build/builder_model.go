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
func NewModel(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, template model.Template, object model.AccessLister, actionID string) (Model, error) {

	const location = "build.NewModel"

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, object, actionID)

	if err != nil {
		return Model{}, derp.Wrap(err, location, "Creating common builder")
	}

	// Enforce permissions on the requested action
	if !common.UserCan(actionID) {

		var traceInfo = "Uncomment the TraceUserCan function for debugging info."
		// traceInfo := common.TraceUserCan(actionID)

		if common._authorization.IsAuthenticated() {
			return Model{}, derp.Forbidden(location, "Forbidden", "User is authenticated, but this action is not allowed", object, actionID, traceInfo)
		} else {
			return Model{}, derp.Unauthorized(location, "Anonymous user is not authorized to perform this action", actionID, traceInfo)
		}
	}

	// Retrieve the correct service to use for this Model object
	modelService := factory.ModelService(object)

	if modelService == nil {
		return Model{}, derp.Internal(location, "Invalid model service", object)
	}

	// Return the Model builder
	return Model{
		_object:            object,
		_service:           modelService,
		CommonWithTemplate: common,
	}, nil
}

// object returns the model object being built. Implements the Builder interface.
func (w Model) object() data.Object {
	return w._object
}

// objectType returns the name of the model type being built. Implements the Builder interface.
func (w Model) objectType() string {
	return w._service.ObjectType()
}

// objectID returns the unique ID of the object being built. Implements the Builder interface.
func (w Model) objectID() primitive.ObjectID {
	return w._service.ObjectID(w._object)
}

// schema returns the rosetta schema that validates this object. Implements the Builder interface.
func (w Model) schema() schema.Schema {
	return w._service.Schema()
}

// service returns the ModelService that backs this Builder. Implements the Builder interface.
func (w Model) service() service.ModelService {
	return w._service
}

// IsNew returns the is new of the Model being built
func (w Model) IsNew() bool {
	return w._object.IsNew()
}

// Object returns the raw model object being built
func (w Model) Object() any {
	return w._object
}

// ObjectID returns the object ID of the Model being built
func (w Model) ObjectID() string {
	return w._object.ID()
}

// Name returns the display label of the object being built
func (w Model) Name() string {
	return w.Label()
}

// Label returns the display label of the object being built, chosen per model type
func (w Model) Label() string {
	switch typed := w._object.(type) {

	case *model.Annotation:
		return typed.Name

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

// Token returns the URL token of the record being built. Implements the Builder interface.
func (w Model) Token() string {
	return ""
}

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w Model) PageTitle() string {
	return ""
}

// Permalink returns the permanent URL of the record being built. Implements the Builder interface.
func (w Model) Permalink() string {
	return ""
}

// BasePath returns the URL path of this object, without any action. Implements the Builder interface.
func (w Model) BasePath() string {
	return ""
}

// UserCan returns TRUE if the signed-in User may run the named action. Implements the Builder interface.
func (w Model) UserCan(string) bool {
	return false
}

// Render builds this object into HTML. Implements the Builder interface.
func (w Model) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Model.Render", "Generating HTML")
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
	subStream, err := NewModel(w._factory, w._session, w._request, w._response, w._template, w._object, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Creating sub-builder")
	}

	// Generate HTML template
	return subStream.Render()
}

/******************************************
 * Custom Queries
 * (may only apply to certain model objects)
 ******************************************/

// Identity returns a QueryBuilder that lists the guest Identities visible to the signed-in User
func (w Model) Identity(identityID primitive.ObjectID) (model.Identity, error) {

	const location = "build.Model.Identity"

	// User must be signed in to view Identities
	if !w._authorization.IsAuthenticated() {
		return model.Identity{}, derp.Unauthorized(location, "Anonymous user is not authorized to perform this action", identityID)
	}

	// Load the Identity from the database
	identity := model.NewIdentity()

	if err := w._factory.Identity().LoadByID(w._session, identityID, &identity); err != nil {
		return model.Identity{}, derp.Wrap(err, location, "Loading identity by token")
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
		return QueryBuilder[model.Identity]{}, derp.Internal(location, "Builder method `CircleMembers` can only be used within a `with-circle` action.")
	}

	// Define inbound parameters
	expressionBuilder := builder.NewBuilder().
		String("name")

	// Calculate criteria
	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("privileges", circle.CircleID),
	)

	// Return the query builder
	return NewQueryBuilder[model.Identity](w._factory.Identity(), w._session, criteria), nil
}

// OAuthCodeURL returns the OAuth authorization URL for an Import, or an empty string for any other model
func (w Model) OAuthCodeURL() string {

	if record, isImport := w.object().(*model.Import); isImport {
		return record.OAuthCodeURL()
	}

	return ""
}

/******************************************
 * Helper functions
 ******************************************/

// setState moves the underlying object into the provided state
func (w Model) setState(stateID string) error {

	if setter, ok := w._object.(model.StateSetter); ok {
		setter.SetState(stateID)
		return nil
	}

	return derp.Internal("build.Model.SetState", "Object does not implement model.StateSetter interface", w._object)
}

// clone returns a copy of this Builder, pointed at a different action. Implements the Builder interface.
func (w Model) clone(action string) (Builder, error) {
	return NewModel(w._factory, w._session, w._request, w._response, w._template, w._object, action)
}

// debug writes this Builder's state to the console. Implements the Builder interface.
func (w Model) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Model")
}
