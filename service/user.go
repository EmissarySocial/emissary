package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionUser is the database collection where Users are stored
const CollectionUser = "User"

// User manages all interactions with the User collection
type User struct {
	factory    *Factory
	collection data.Collection
}

// New creates a newly initialized User that is ready to use
func (service User) New() *model.User {
	return &model.User{
		UserID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Users who match the provided criteria
func (service User) List(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an User from the database
func (service User) Load(criteria expression.Expression) (*model.User, error) {

	contact := service.New()

	if err := service.collection.Load(criteria, contact); err != nil {
		return nil, derp.Wrap(err, "service.User", "Error loading User", criteria)
	}

	return contact, nil
}

// Save adds/updates an User in the database
func (service User) Save(stage *model.User, note string) error {

	if err := service.collection.Save(stage, note); err != nil {
		return derp.Wrap(err, "service.Stage", "Error saving Stage", stage, note)
	}

	return nil
}

// Delete removes an User from the database (virtual delete)
func (service User) Delete(stage *model.User, note string) error {

	if err := service.collection.Delete(stage, note); err != nil {
		return derp.Wrap(err, "service.Stage", "Error deleting Stage", stage, note)
	}

	return nil
}

func (service User) Close() {
	service.factory.Close()
}

func (service User) LoadByUsername(username string) (*model.User, error) {
	return service.Load(expression.Equal("username", username))
}
