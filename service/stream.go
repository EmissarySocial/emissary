package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionStream is the database collection where Streams are stored
const CollectionStream = "Stream"

// Stream manages all interactions with the Stream collection
type Stream struct {
	factory Factory
	session data.Session
}

// New creates a newly initialized Stream that is ready to use
func (service Stream) New() *model.Stream {
	return &model.Stream{
		StreamID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Streams who match the provided criteria
func (service Stream) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.session.List(CollectionStream, criteria, options...)
}

// Load retrieves an Stream from the database
func (service Stream) Load(criteria expression.Expression) (*model.Stream, *derp.Error) {

	account := service.New()

	if err := service.session.Load(CollectionStream, criteria, account); err != nil {
		return nil, derp.Wrap(err, "service.Stream", "Error loading Stream", criteria)
	}

	return account, nil
}

// Save adds/updates an Stream in the database
func (service Stream) Save(account *model.Stream, note string) *derp.Error {

	if err := service.session.Save(CollectionStream, account, note); err != nil {
		return derp.Wrap(err, "service.Stream", "Error saving Stream", account, note)
	}

	return nil
}

// Delete removes an Stream from the database (virtual delete)
func (service Stream) Delete(account *model.Stream, note string) *derp.Error {

	if err := service.session.Delete(CollectionStream, account, note); err != nil {
		return derp.Wrap(err, "service.Stream", "Error deleting Stream", account, note)
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
	service.session.Close()
}

// QUERIES /////////////////////////

func (service Stream) LoadByToken(token string) (*model.Stream, *derp.Error) {

	return service.Load(
		expression.New("token", expression.OperatorEqual, token))
}

// LoadBySourceURL locates a single stream that matches the provided SourceURL
func (service Stream) LoadBySourceURL(url string) (*model.Stream, *derp.Error) {
	return service.Load(
		expression.New("sourceUrl", expression.OperatorEqual, url))
}

///////////////////

// SaveUniqueStreamBySourceURL saves a stream, and avoids duplicates using the SourceURL property.
func (service Stream) SaveUniqueStreamBySourceURL(stream *model.Stream, note string) *derp.Error {

	object, err := service.LoadBySourceURL(stream.SourceURL)

	// We already have this object.
	// TODO: Add compare/copy function to model.Stream object and use it here.
	if err == nil {

		if changed := object.UpdateWith(stream); changed == false {
			return nil
		}

		// Fall through to here means that we have made changes. Update the "stream" variable with the now-correct value from "object" and continue as normal
		stream = object

	} else {

		// This means that there was an *derp.Error connecting to the database.
		if err.NotFound() == false {
			return derp.Wrap(err, "service.Stream.SaveUniqueStreamBySourceURL", "Error querying Stream from database", stream)
		}
	}

	// Fall through to here means that it was a "Not Found" *derp.Error, which means we can safely add the new stream.
	if err := service.Save(stream, note); err != nil {
		return derp.Wrap(err, "service.Stream.SaveUniqueStreamBySourceURL", "Error saving new Stream", stream, note)
	}

	return nil
}

// Render generates HTML output for the provided stream.  It looks up the appropriate
// template/view for this stream, and executes the template.
func (service Stream) Render(stream *model.Stream, viewID string) (string, *derp.Error) {

	templateService := service.factory.Template()

	// Try to load the template from the database
	template, err := templateService.LoadByName(stream.Template)

	if err != nil {
		return "", derp.Wrap(err, "service.Template.Render", "Unable to load Template", stream)
	}

	// Try to find the view in the list of views
	view, ok := template.Views[viewID]

	if !ok {
		return "", derp.New(404, "service.Template.Render", "Unrecognized view", viewID)
	}

	// TODO: need to enforce permissions somewhere...

	// Try to generate the HTML response using the provided data
	html, err := view.Execute(stream)

	if err != nil {
		return "", derp.Wrap(err, "service.Template.Render", "Error rendering view")
	}

	// Success!
	return html, nil
}
