package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionUser is the database collection where Users are stored
const CollectionUser = "User"

// User manages all interactions with the User collection
type User struct {
	factory *Factory
	session data.Session
}

// New creates a newly initialized User that is ready to use
func (service User) New() *model.User {
	return &model.User{
		UserID: primitive.NewObjectID(),
	}
}

// Load retrieves an User from the database
func (service User) Load(criteria expression.Expression) (*model.User, *derp.Error) {

	contact := service.New()

	if err := service.session.Load(CollectionUser, criteria, contact); err != nil {
		return nil, derp.Wrap(err, "service.User", "Error loading User", criteria)
	}

	return contact, nil
}

// Save adds/updates an User in the database
func (service User) Save(stage *model.User, note string) *derp.Error {

	if err := service.session.Save(CollectionUser, stage, note); err != nil {
		return derp.Wrap(err, "service.Stage", "Error saving Stage", stage, note)
	}

	return nil
}

// Delete removes an User from the database (virtual delete)
func (service User) Delete(stage *model.User, note string) *derp.Error {

	if err := service.session.Delete(CollectionUser, stage, note); err != nil {
		return derp.Wrap(err, "service.Stage", "Error deleting Stage", stage, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service User) NewObject() data.Object {
	return service.New()
}

// LoadObject wraps the `Load` method as a generic Object
func (service User) LoadObject(criteria expression.Expression) (data.Object, *derp.Error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service User) SaveObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.User); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.User", "Object is not a model.User", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service User) DeleteObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.User); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.User", "Object is not a model.User", object, note)
}

// Close cleans up the service and any outstanding connections.
func (service User) Close() {
	service.session.Close()
}
