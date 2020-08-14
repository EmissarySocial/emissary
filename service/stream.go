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
	factory    Factory
	collection data.Collection
}

// New creates a newly initialized Stream that is ready to use
func (service Stream) New() *model.Stream {
	return &model.Stream{
		StreamID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Streams who match the provided criteria
func (service Stream) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an Stream from the database
func (service Stream) Load(criteria expression.Expression) (*model.Stream, *derp.Error) {

	stream := service.New()

	if err := service.collection.Load(criteria, stream); err != nil {
		return nil, derp.Wrap(err, "service.Stream", "Error loading Stream", criteria)
	}

	return stream, nil
}

// Save adds/updates an Stream in the database
func (service Stream) Save(stream *model.Stream, note string) *derp.Error {

	if err := service.collection.Save(stream, note); err != nil {
		return derp.Wrap(err, "service.Stream", "Error saving Stream", stream, note)
	}

	return nil
}

// Delete removes an Stream from the database (virtual delete)
func (service Stream) Delete(stream *model.Stream, note string) *derp.Error {

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
func (service Stream) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service Stream) LoadObject(criteria expression.Expression) (data.Object, *derp.Error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service Stream) SaveObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Stream); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Stream", "Object is not a model.Stream", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service Stream) DeleteObject(object data.Object, note string) *derp.Error {

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

func (service Stream) ListByTemplate(template string) (data.Iterator, *derp.Error) {
	return service.List(
		expression.New("template", expression.OperatorEqual, template))
}

func (service Stream) LoadByToken(token string) (*model.Stream, *derp.Error) {
	return service.Load(
		expression.
			New("token", expression.OperatorEqual, token).
			And("journal.deleteDate", expression.OperatorEqual, 0))
}

// LoadBySourceURL locates a single stream that matches the provided SourceURL
func (service Stream) LoadBySourceURL(url string) (*model.Stream, *derp.Error) {
	return service.Load(
		expression.New("sourceUrl", expression.OperatorEqual, url))
}

///////////////////

// Render generates HTML output for the provided stream.  It looks up the appropriate
// template/view for this stream, and executes the template.
func (service Stream) Render(stream *model.Stream, viewName string) (string, *derp.Error) {

	templateService := service.factory.Template()

	// Try to load the template from the database
	template, err := templateService.Load(stream.Template)

	if err != nil {
		return "", derp.Wrap(err, "service.Template.Render", "Unable to load Template", stream)
	}

	// Locate / Authenticate the view to use

	view, err := template.View(stream.State, viewName)

	if err != nil {
		return "", derp.Wrap(err, "service.Template.Render", "Unrecognized view", view)
	}

	// TODO: need to enforce permissions somewhere...

	// Try to generate the HTML response using the provided data
	result, err := view.Execute(stream)

	if err != nil {
		return "", derp.Wrap(err, "service.Template.Render", "Error rendering view")
	}

	result = html.CollapseWhitespace(result)

	// TODO: Add caching here...

	// Success!
	return result, nil
}

// Transition handles a transition request to move the stream from one state into another state.
func (service Stream) Transition(stream *model.Stream, transition *model.Transition, data map[string]interface{}) *derp.Error {

	// TODO: where are permissions processed?

	paths := transition.Form.AllPaths()

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
	stream.State = transition.StateID

	// TODO:  Actions will be processes here.

	return service.Save(stream, "stream transition: "+transition.ID)
}
