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
