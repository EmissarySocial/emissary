package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/html"
	"github.com/benpate/path"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionStream is the database collection where Streams are stored
const CollectionStream = "Stream"

// Stream manages all interactions with the Stream collection
type Stream struct {
	factory    *Factory
	collection data.Collection
}

// New creates a newly initialized Stream that is ready to use
func (service Stream) New() *model.Stream {
	return &model.Stream{
		StreamID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Streams who match the provided criteria
func (service Stream) List(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an Stream from the database
func (service Stream) Load(criteria expression.Expression) (*model.Stream, error) {

	stream := service.New()

	if err := service.collection.Load(criteria, stream); err != nil {
		return nil, derp.Wrap(err, "service.Stream", "Error loading Stream", criteria)
	}

	return stream, nil
}

// Save adds/updates an Stream in the database
func (service Stream) Save(stream *model.Stream, note string) error {

	if err := service.collection.Save(stream, note); err != nil {
		return derp.Wrap(err, "service.Stream", "Error saving Stream", stream, note)
	}

	return nil
}

// Delete removes an Stream from the database (virtual delete)
func (service Stream) Delete(stream *model.Stream, note string) error {

	if err := service.collection.Delete(stream, note); err != nil {
		return derp.Wrap(err, "service.Stream", "Error deleting Stream", stream, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service Stream) NewObject() data.Object {
	return service.New()
}

// ListObjects wraps the `List` method as a generic Object
func (service Stream) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service Stream) LoadObject(criteria expression.Expression) (data.Object, error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service Stream) SaveObject(object data.Object, note string) error {

	if object, ok := object.(*model.Stream); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Stream", "Object is not a model.Stream", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service Stream) DeleteObject(object data.Object, note string) error {

	if object, ok := object.(*model.Stream); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Stream", "Object is not a model.Stream", object, note)
}

// Close cleans up the service and any outstanding connections.
func (service Stream) Close() {
	service.factory.Close()
}

// QUERIES /////////////////////////

func (service Stream) ListByParent(parentID primitive.ObjectID) (data.Iterator, error) {
	return service.List(
		expression.
			New("parentId", expression.OperatorEqual, parentID).
			And("journal.deleteDate", expression.OperatorEqual, 0))
}

func (service Stream) ListByTemplate(template string) (data.Iterator, error) {
	return service.List(
		expression.
			New("template", expression.OperatorEqual, template).
			And("journal.deleteDate", expression.OperatorEqual, 0))
}

func (service Stream) LoadByToken(token string) (*model.Stream, error) {
	return service.Load(
		expression.
			New("token", expression.OperatorEqual, token).
			And("journal.deleteDate", expression.OperatorEqual, 0))
}

func (service Stream) LoadByID(streamID primitive.ObjectID) (*model.Stream, error) {
	return service.Load(
		expression.
			New("_id", expression.OperatorEqual, streamID).
			And("journal.deleteDate", expression.OperatorEqual, 0))
}

func (service Stream) LoadParent(stream *model.Stream) (*model.Stream, error) {

	if stream.HasParent() == false {
		return nil, derp.New(404, "ghost.service.Stream.LoadParent", "Stream does not have a parent")
	}

	stream, err := service.LoadByID(stream.ParentID)

	return stream, derp.Wrap(err, "ghost.service.stream.LoadParent", "Error loading parent", stream)
}

// LoadBySourceURL locates a single stream that matches the provided SourceURL
func (service Stream) LoadBySourceURL(url string) (*model.Stream, error) {
	return service.Load(
		expression.
			New("sourceUrl", expression.OperatorEqual, url).
			And("journal.deleteDate", expression.OperatorEqual, 0))
}

///////////////////

// Render generates HTML output for the provided stream.  It looks up the appropriate
// template/view for this stream, and executes the template.
func (service Stream) Render(stream *model.Stream, viewName string) (string, error) {

	templateService := service.factory.Template()

	// Try to load the template from the database
	template, err := templateService.Load(stream.Template)

	if err != nil {
		return "", derp.Wrap(err, "service.Stream.Render", "Unable to load Template", stream)
	}

	// Locate / Authenticate the view to use

	view, err := template.View(stream.State, viewName)

	if err != nil {
		return "", derp.Wrap(err, "service.Stream.Render", "Unrecognized view", viewName)
	}

	// TODO: need to enforce permissions somewhere...

	// Try to generate the HTML response using the provided data
	result, err := view.Execute(stream)

	if err != nil {
		return "", derp.Wrap(err, "service.Stream.Render", "Error rendering view")
	}

	result = html.CollapseWhitespace(result)

	// TODO: Add caching here...

	// Success!
	return result, nil
}

// Transition handles a transition request to move the stream from one state into another state.
func (service Stream) Transition(stream *model.Stream, template *model.Template, transitionID string, data map[string]interface{}) error {

	transition, err := template.Transition(stream.State, transitionID)

	if err != nil {
		return derp.Wrap(err, "ghost.service.Stream.Transition", "Unrecognized State/Tranition")
	}

	form, ok := template.Forms[transition.Form]

	if !ok {
		return derp.New(404, "ghost.service.Stream.Transition", "Unrecognized Form for Transition", transition)
	}

	// TODO: where are permissions processed?

	paths := form.AllPaths()

	// Only look for data in the registered paths for this form.  If other data is present, it will be ignored.
	for _, element := range paths {

		// TODO: What about form validation?  Can this happen HERE as well as in the template schema?

		// If we have a value, then set it.
		if value, ok := data[element.Path]; ok {
			if err := path.Set(stream, element.Path, value); err != nil {
				return derp.Wrap(err, "ghost.service.Stream.Transition", "Error updating stream", element, value)
			}
		}
		// TODO: Otherwise?  Should this form throw an error?
	}

	// Update the stream to the new state
	stream.State = transition.NextState

	// TODO:  Actions will be processes here.

	return service.Save(stream, "stream transition: "+transitionID)
}
