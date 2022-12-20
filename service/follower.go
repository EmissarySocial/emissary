package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Follower defines a service that tracks the (possibly external) accounts that are followers of an internal User

type Follower struct {
	collection  data.Collection
	userService *User
	host        string
}

// NewFollower returns a fully initialized Follower service
func NewFollower(collection data.Collection, userService *User, host string) Follower {
	service := Follower{
		userService: userService,
		host:        host,
	}

	service.Refresh(collection)
	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Follower) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Follower) Close() {
	// Nothin to do here.
}

/*******************************************
 * Common Data Methods
 *******************************************/

func (service *Follower) Query(criteria exp.Expression, options ...option.Option) ([]model.Follower, error) {
	result := make([]model.Follower, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Followers who match the provided criteria
func (service *Follower) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Follower from the database
func (service *Follower) Load(criteria exp.Expression, follower *model.Follower) error {

	if err := service.collection.Load(notDeleted(criteria), follower); err != nil {
		return derp.Wrap(err, "service.Follower.Load", "Error loading Follower", criteria)
	}

	return nil
}

// Save adds/updates an Follower in the database
func (service *Follower) Save(follower *model.Follower, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(follower); err != nil {
		return derp.Wrap(err, "service.Follower.Save", "Error cleaning Follower", follower)
	}

	// Save the follower to the database
	if err := service.collection.Save(follower, note); err != nil {
		return derp.Wrap(err, "service.Follower.Save", "Error saving Follower", follower, note)
	}

	// Recalculate the follower count for this user
	go service.userService.CalcFollowerCount(follower.ParentID)

	// TODO: Notify followers (if necessary)

	return nil
}

// Delete removes an Follower from the database (virtual delete)
func (service *Follower) Delete(follower *model.Follower, note string) error {

	// Delete this Follower
	if err := service.collection.Delete(follower, note); err != nil {
		return derp.Wrap(err, "service.Follower.Delete", "Error deleting Follower", follower, note)
	}

	return nil
}

/*******************************************
 * Model Service Methods
 *******************************************/

// ObjectType returns the type of object that this service manages
func (service *Follower) ObjectType() string {
	return "Follow"
}

// New returns a fully initialized model.Group as a data.Object.
func (service *Follower) ObjectNew() data.Object {
	result := model.NewFollower()
	return &result
}

func (service *Follower) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Follower); ok {
		return mention.FollowerID
	}

	return primitive.NilObjectID
}

func (service *Follower) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Follower) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Follower) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewFollower()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Follower) ObjectSave(object data.Object, comment string) error {
	if follower, ok := object.(*model.Follower); ok {
		return service.Save(follower, comment)
	}
	return derp.NewInternalError("service.Follower.ObjectSave", "Invalid Object Type", object)
}

func (service *Follower) ObjectDelete(object data.Object, comment string) error {
	if follower, ok := object.(*model.Follower); ok {
		return service.Delete(follower, comment)
	}
	return derp.NewInternalError("service.Follower.ObjectDelete", "Invalid Object Type", object)
}

func (service *Follower) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Follower", "Not Authorized")
}

func (service *Follower) Schema() schema.Schema {
	return schema.New(model.FollowerSchema())
}

/*******************************************
 * Custom Queries
 *******************************************/

func (service *Follower) QueryAllURLs(criteria exp.Expression) ([]string, error) {

	result := make([]string, 0)
	fields := option.Fields("actor.profileUrl")

	if err := service.collection.Query(&result, notDeleted(criteria), fields); err != nil {
		return result, derp.Wrap(err, "service.Follower.QueryFollowerURLs", "Error querying follower URLs", criteria)
	}

	return result, nil
}

/*******************************************
 * WebSub Queries
 *******************************************/

func (service *Follower) ListWebSub(parentID primitive.ObjectID) (data.Iterator, error) {

	criteria := exp.
		Equal("parentId", parentID).
		AndEqual("method", model.FollowMethodWebSub)

	return service.List(criteria)
}

func (service *Follower) ListWebSubByCallback(parentID primitive.ObjectID, callback string) (data.Iterator, error) {
	criteria := exp.
		Equal("parentId", parentID).
		AndEqual("data.callback", callback).
		AndEqual("method", model.FollowMethodWebSub)

	return service.List(criteria)
}

func (service *Follower) LoadByWebSub(parentID primitive.ObjectID, callback string, result *model.Follower) error {

	criteria := exp.
		Equal("userId", parentID).
		AndEqual("data.callback", callback).
		AndEqual("method", model.FollowMethodWebSub)

	return service.Load(criteria, result)
}
