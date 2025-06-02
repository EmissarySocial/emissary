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

// Circle manages all interactions with the Circle collection
type Circle struct {
	collection       data.Collection
	privilegeService *Privilege
}

// NewCircle returns a fully populated Circle service
func NewCircle() Circle {
	return Circle{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Circle) Refresh(collection data.Collection, privilegeService *Privilege) {
	service.collection = collection
	service.privilegeService = privilegeService
}

// Close stops any background processes controlled by this service
func (service *Circle) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// Count returns the number of records that match the provided criteria
func (service *Circle) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

func (service *Circle) Query(criteria exp.Expression, options ...option.Option) ([]model.Circle, error) {
	result := make([]model.Circle, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Circles who match the provided criteria
func (service *Circle) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Circle from the database
func (service *Circle) Load(criteria exp.Expression, result *model.Circle) error {
	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Circle.Load", "Error loading Circle", criteria)
	}

	return nil
}

// Save adds/updates an Circle in the database
func (service *Circle) Save(circle *model.Circle, note string) error {

	const location = "service.Circle.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(circle); err != nil {
		return derp.Wrap(err, location, "Error validating Circle", circle)
	}

	// Save the value to the database
	if err := service.collection.Save(circle, note); err != nil {
		return derp.Wrap(err, location, "Error saving Circle", circle, note)
	}

	return nil
}

// Delete removes an Circle from the database (virtual delete)
func (service *Circle) Delete(circle *model.Circle, note string) error {

	const location = "service.Circle.Delete"

	if err := service.collection.Delete(circle, note); err != nil {
		return derp.Wrap(err, location, "Error deleting Circle", circle, note)
	}

	if err := service.privilegeService.DeleteByCircle(circle.CircleID, note); err != nil {
		return derp.Wrap(err, location, "Error deleting Privileges for Circle", circle.CircleID, note)
	}

	// TODO: HIGH: Also remove connections to Streams that still use this Circle

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Circle) ObjectType() string {
	return "Circle"
}

// New returns a fully initialized model.Circle as a data.Object.
func (service *Circle) ObjectNew() data.Object {
	result := model.NewCircle()
	return &result
}

func (service *Circle) ObjectID(object data.Object) primitive.ObjectID {

	if circle, ok := object.(*model.Circle); ok {
		return circle.CircleID
	}

	return primitive.NilObjectID
}

func (service *Circle) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Circle) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewCircle()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Circle) ObjectSave(object data.Object, comment string) error {
	if circle, ok := object.(*model.Circle); ok {
		return service.Save(circle, comment)
	}
	return derp.InternalError("service.Circle.ObjectSave", "Invalid Object Type", object)
}

func (service *Circle) ObjectDelete(object data.Object, comment string) error {
	if circle, ok := object.(*model.Circle); ok {
		return service.Delete(circle, comment)
	}
	return derp.InternalError("service.Circle.ObjectDelete", "Invalid Object Type", object)
}

func (service *Circle) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Circle", "Not Authorized")
}

func (service *Circle) Schema() schema.Schema {
	return schema.New(model.CircleSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// QueryByUser returns all Circles that are owned by the provided userID
func (service *Circle) QueryByUser(userID primitive.ObjectID, options ...option.Option) ([]model.Circle, error) {
	criteria := exp.Equal("userId", userID)
	return service.Query(criteria, options...)
}

// LoadByID loads a single model.Circle object that matches the provided circleID
func (service *Circle) LoadByID(userID primitive.ObjectID, circleID primitive.ObjectID, result *model.Circle) error {
	criteria := exp.Equal("_id", circleID).AndEqual("userId", userID)
	return service.Load(criteria, result)
}

/******************************************
 * Custom Behaviors
 ******************************************/

// RefreshMemberCounts updates the member counts for the Circle with the provided circleID
func (service *Circle) RefreshMemberCounts(userID primitive.ObjectID, circleID primitive.ObjectID) error {

	const location = "service.Circle.RefreshMemberCounts"

	// Load the circle to ensure it exists
	circle := model.NewCircle()
	if err := service.LoadByID(userID, circleID, &circle); err != nil {
		return derp.Wrap(err, location, "Error loading Circle", circleID)
	}

	// Count the number of privileges for this Circle
	count, err := service.privilegeService.CountByCircle(circleID)

	if err != nil {
		return derp.Wrap(err, location, "Error counting privileges for Circle", circleID)
	}

	// If the count is correct, then we have triumphed
	if count == circle.MemberCount {
		return nil
	}

	// Otherwise, true grit would update the Circle with the new member count
	circle.MemberCount = count

	if err := service.Save(&circle, "Refresh Member Count"); err != nil {
		return derp.Wrap(err, location, "Error saving Circle with updated member count", circleID)
	}

	// We have survived another day
	return nil
}
