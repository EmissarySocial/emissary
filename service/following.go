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

// Following defines a service that tracks the (possibly external) accounts an internal User is following.
type Following struct {
	collection data.Collection
}

// NewFollowing returns a fully initialized Following service
func NewFollowing(collection data.Collection) Following {
	service := Following{}
	service.Refresh(collection)
	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Following) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Following) Close() {
	// Nothin to do here.
}

/*******************************************
 * Common Data Methods
 *******************************************/

// List returns an iterator containing all of the Followings who match the provided criteria
func (service *Following) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Following from the database
func (service *Following) Load(criteria exp.Expression, following *model.Following) error {

	if err := service.collection.Load(notDeleted(criteria), following); err != nil {
		return derp.Wrap(err, "service.Following.Load", "Error loading Following", criteria)
	}

	return nil
}

// Save adds/updates an Following in the database
func (service *Following) Save(following *model.Following, note string) error {

	if err := service.collection.Save(following, note); err != nil {
		return derp.Wrap(err, "service.Following.Save", "Error saving Following", following, note)
	}

	return nil
}

// Delete removes an Following from the database (virtual delete)
func (service *Following) Delete(following *model.Following, note string) error {

	// Delete this Following
	if err := service.collection.Delete(following, note); err != nil {
		return derp.Wrap(err, "service.Following.Delete", "Error deleting Following", following, note)
	}

	return nil
}

/*******************************************
 * Model Service Methods
 *******************************************/

// New returns a fully initialized model.Group as a data.Object.
func (service *Following) ObjectNew() data.Object {
	result := model.NewFollowing()
	return &result
}

func (service *Following) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Following); ok {
		return mention.FollowingID
	}

	return primitive.NilObjectID
}

func (service *Following) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Following) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewFollowing()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Following) ObjectSave(object data.Object, comment string) error {
	if following, ok := object.(*model.Following); ok {
		return service.Save(following, comment)
	}
	return derp.NewInternalError("service.Following.ObjectSave", "Invalid Object Type", object)
}

func (service *Following) ObjectDelete(object data.Object, comment string) error {
	if following, ok := object.(*model.Following); ok {
		return service.Delete(following, comment)
	}
	return derp.NewInternalError("service.Following.ObjectDelete", "Invalid Object Type", object)
}

func (service *Following) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Following", "Not Authorized")
}

func (service *Following) Schema() schema.Schema {
	return schema.New(model.FollowingSchema())
}

/*******************************************
 * Custom Queries
 *******************************************/

func (service *Following) QueryAllURLs(criteria exp.Expression) ([]string, error) {

	result := make([]string, 0)
	fields := option.Fields("object.profileUrl")

	if err := service.collection.Query(&result, notDeleted(criteria), fields); err != nil {
		return result, derp.Wrap(err, "service.Following.QueryFollowingURLs", "Error querying following URLs", criteria)
	}

	return result, nil
}
