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
func NewUser(collection data.Collection) User {
	return User{
		collection: collection,
	}
}

/*******************************************
 * COMMON DATA FUNCTIONS
 *******************************************/

// List returns an iterator containing all of the Users who match the provided criteria
func (service *User) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an User from the database
func (service *User) Load(criteria exp.Expression, result *model.User) error {
	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.User", "Error loading User", criteria)
	}

	return nil
}

// Save adds/updates an User in the database
func (service *User) Save(user *model.User, note string) error {

	if err := service.collection.Save(user, note); err != nil {
		return derp.Wrap(err, "service.User", "Error saving User", user, note)
	}

	return nil
}

// Delete removes an User from the database (virtual delete)
func (service *User) Delete(user *model.User, note string) error {

	if err := service.collection.Delete(user, note); err != nil {
		return derp.Wrap(err, "service.User", "Error deleting User", user, note)
	}

	return nil
}

/*******************************************
 * GENERIC DATA FUNCTIONS
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *User) ObjectNew() data.Object {
	result := model.NewUser()
	return &result
}

func (service *User) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *User) ObjectLoad(criteria exp.Expression, object data.Object) error {
	return service.Load(criteria, object.(*model.User))
}

func (service *User) ObjectSave(object data.Object, comment string) error {
	return service.Save(object.(*model.User), comment)
}

func (service *User) ObjectDelete(object data.Object, comment string) error {
	return service.Delete(object.(*model.User), comment)
}

/*******************************************
 * CUSTOM QUERIES
 *******************************************/

// LoadByID loads a single model.User object that matches the provided userID
func (service *User) LoadByID(userID primitive.ObjectID, result *model.User) error {
	criteria := exp.Equal("_id", userID)
	return service.Load(criteria, result)
}

// LoadByUsername loads a single model.User object that matches the provided username
func (service *User) LoadByUsername(username string, result *model.User) error {
	criteria := exp.Equal("username", username)
	return service.Load(criteria, result)
}

// LoadByToken loads a single model.User object that matches the provided username
func (service *User) LoadByToken(username string, result *model.User) error {
	criteria := exp.Equal("username", username)
	return service.Load(criteria, result)
}

// ListByGroup returns all users that match a provided group name
func (service *User) ListByGroup(group string) (data.Iterator, error) {
	return service.List(exp.Equal("groupId", group))
}
