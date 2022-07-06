package service

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/maps"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User manages all interactions with the User collection
type User struct {
	collection    data.Collection
	streamService *Stream
}

// NewUser returns a fully populated User service
func NewUser(collection data.Collection, streamService *Stream) User {
	return User{
		collection:    collection,
		streamService: streamService,
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

	// Guarantee Inbox location
	if user.InboxID == primitive.NilObjectID {
		if err := service.CreateInbox(user); err != nil {
			return derp.Wrap(err, "service.User", "Error creating inbox")
		}
	}

	// Guarantee Outbox location
	if user.OutboxID == primitive.NilObjectID {
		if err := service.CreateOutbox(user); err != nil {
			return derp.Wrap(err, "service.User", "Error creating inbox")
		}
	}

	// Save User
	if err := service.collection.Save(user, note); err != nil {
		return derp.Wrap(err, "service.User", "Error saving User", user, note)
	}

	// Success!
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

func (service *User) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewUser()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *User) ObjectSave(object data.Object, comment string) error {
	return service.Save(object.(*model.User), comment)
}

func (service *User) ObjectDelete(object data.Object, comment string) error {
	return service.Delete(object.(*model.User), comment)
}

func (service *User) Debug() maps.Map {
	return maps.Map{
		"service": "User",
	}
}

/*******************************************
 * CUSTOM QUERIES
 *******************************************/

// ListByIdentities returns all users that appear in the list of identities
func (service *User) ListByIdentities(identities []string) (data.Iterator, error) {
	return service.List(exp.In("identities", identities))
}

// ListByGroup returns all users that match a provided group name
func (service *User) ListByGroup(group string) (data.Iterator, error) {
	return service.List(exp.Equal("groupId", group))
}

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

// LoadByUsername loads a single model.User object that matches the provided token
func (service *User) LoadByToken(token string, result *model.User) error {

	// If the token *looks* like an ObjectID then try that first.  If it works, then return in triumph
	if userID, err := primitive.ObjectIDFromHex(token); err == nil {
		if err := service.LoadByID(userID, result); err == nil {
			return nil
		}
	}

	// Otherwise, use the token as a username
	criteria := exp.Equal("username", token)
	return service.Load(criteria, result)
}

// Count returns the number of (non-deleted) records in the User collection
func (service *User) Count(ctx context.Context, criteria exp.Expression) (int, error) {
	return queries.CountRecords(ctx, service.collection, notDeleted(criteria))
}

/*******************************************
 * CUSTOM ACTIONS
 *******************************************/

// CreateInbox creates a personal "inbox" stream for a user
func (service *User) CreateInbox(user *model.User) error {

	streamID, err := service.streamService.CreatePersonalStream(user, "social-inbox")

	if err == nil {
		user.InboxID = streamID
	}

	return err
}

// CreateOutbox creates a personal "outbox" stream for a user
func (service *User) CreateOutbox(user *model.User) error {

	streamID, err := service.streamService.CreatePersonalStream(user, "social-outbox")

	if err == nil {
		user.OutboxID = streamID
	}

	return err
}
