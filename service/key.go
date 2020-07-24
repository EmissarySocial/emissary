package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionKey is the database collection where Keys are stored
const CollectionKey = "Key"

// Key manages all interactions with the Key collection
type Key struct {
	factory    Factory
	collection data.Collection
}

// New creates a newly initialized Key that is ready to use
func (service Key) New() *model.Key {
	return &model.Key{
		KeyID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Keys who match the provided criteria
func (service Key) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an Key from the database
func (service Key) Load(criteria expression.Expression) (*model.Key, *derp.Error) {

	key := service.New()

	if err := service.collection.Load(criteria, key); err != nil {
		return nil, derp.Wrap(err, "service.Key", "Error loading Key", criteria)
	}

	return key, nil
}

// Save adds/updates an Key in the database
func (service Key) Save(key *model.Key, note string) *derp.Error {

	if err := service.collection.Save(key, note); err != nil {
		return derp.Wrap(err, "service.Key", "Error saving Key", key, note)
	}

	return nil
}

// Delete removes an Key from the database (virtual delete)
func (service Key) Delete(key *model.Key, note string) *derp.Error {

	if err := service.collection.Delete(key, note); err != nil {
		return derp.Wrap(err, "service.Key", "Error deleting Key", key, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service Key) NewObject() data.Object {
	return service.New()
}

// ListObjects wraps the `List` method as a generic Object
func (service Key) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service Key) LoadObject(criteria expression.Expression) (data.Object, *derp.Error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service Key) SaveObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Key); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Key", "Object is not a model.Key", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service Key) DeleteObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Key); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Key", "Object is not a model.Key", object, note)
}

// Close cleans up the service and any outstanding connections.
func (service Key) Close() {
	service.factory.Close()
}
