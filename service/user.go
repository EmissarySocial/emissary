package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User manages all interactions with the User collection
type User struct {
	collection data.Collection
}

// NewUser returns a fully populated User service
func NewUser(collection data.Collection) *User {
	return &User{
		collection: collection,
	}
}

// New creates a newly initialized User that is ready to use
func (service User) New() *model.User {
	return &model.User{
		UserID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Users who match the provided criteria
func (service User) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an User from the database
func (service User) Load(criteria exp.Expression) (*model.User, error) {

	contact := service.New()

	if err := service.collection.Load(criteria, contact); err != nil {
		return nil, derp.Wrap(err, "service.User", "Error loading User", criteria)
	}

	return contact, nil
}

// Save adds/updates an User in the database
func (service User) Save(user *model.User, note string) error {

	if err := service.collection.Save(user, note); err != nil {
		return derp.Wrap(err, "service.User", "Error saving User", user, note)
	}

	return nil
}

// Delete removes an User from the database (virtual delete)
func (service User) Delete(user *model.User, note string) error {

	if err := service.collection.Delete(user, note); err != nil {
		return derp.Wrap(err, "service.User", "Error deleting User", user, note)
	}

	return nil
}

///////////////////////////
// Queries

// LoadByUsername loads a single model.User object that matches the provided username
func (service User) LoadByUsername(username string) (*model.User, error) {
	return service.Load(exp.Equal("username", username))
}

// ListByGroup returns all users that match a provided group name
func (service User) ListByGroup(group string) (data.Iterator, error) {
	return service.List(exp.Equal("groupId", group))
}
