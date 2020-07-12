package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionActor is the database collection where Actors are stored
const CollectionActor = "Actor"

// Actor manages all interactions with the Actor collection
type Actor struct {
	factory Factory
	session data.Session
}

// New creates a newly initialized Actor that is ready to use
func (service Actor) New() *model.Actor {
	return &model.Actor{
		ActorID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Actors who match the provided criteria
func (service Actor) List(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return service.session.List(CollectionActor, criteria, options...)
}

// Load retrieves an Actor from the database
func (service Actor) Load(criteria expression.Expression) (*model.Actor, error) {

	actor := service.New()

	if err := service.session.Load(CollectionActor, criteria, actor); err != nil {
		return nil, derp.Wrap(err, "service.Actor", "Error loading Actor", criteria)
	}

	return actor, nil
}

// Save adds/updates an Actor in the database
func (service Actor) Save(actor *model.Actor, note string) error {

	if err := service.session.Save(CollectionActor, actor, note); err != nil {
		return derp.Wrap(err, "service.Actor", "Error saving Actor", actor, note)
	}

	return nil
}

// Delete removes an Actor from the database (virtual delete)
func (service Actor) Delete(actor *model.Actor, note string) error {

	if err := service.session.Delete(CollectionActor, actor, note); err != nil {
		return derp.Wrap(err, "service.Actor", "Error deleting Actor", actor, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service Actor) NewObject() data.Object {
	return service.New()
}

// ListObjects wraps the `List` method as a generic Object
func (service Actor) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service Actor) LoadObject(criteria expression.Expression) (data.Object, error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service Actor) SaveObject(object data.Object, note string) error {

	if object, ok := object.(*model.Actor); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Actor", "Object is not a model.Actor", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service Actor) DeleteObject(object data.Object, note string) error {

	if object, ok := object.(*model.Actor); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Actor", "Object is not a model.Actor", object, note)
}

// Close cleans up the service and any outstanding connections.
func (service Actor) Close() {
	service.session.Close()
}
