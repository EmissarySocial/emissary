package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
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

	const location = "service.Circle.Load"

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, location, "Error loading Circle", criteria)
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

	// Recalculate privileges based on new Circle settings.
	if err := service.privilegeService.RefreshCircleInfo(circle); err != nil {
		return derp.Wrap(err, location, "Error refreshing Privileges for Circle", circle.CircleID, note)
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

func (service *Circle) QueryByIDs(userID primitive.ObjectID, circleIDs []primitive.ObjectID, options ...option.Option) ([]model.Circle, error) {

	const location = "service.Circle.QueryByIDs"

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, derp.ValidationError("UserID cannot be zero")
	}

	// RULE: Require at least one CircleID
	if len(circleIDs) == 0 {
		return nil, derp.ValidationError("CircleIDs cannot be empty")
	}

	criteria := exp.In("_id", circleIDs).AndEqual("userId", userID)

	// Load the Merchant Accounts for this User
	result, err := service.Query(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading merchant accounts")
	}

	return result, nil
}

// QueryByUser returns all Circles that are owned by the provided userID
func (service *Circle) QueryByUser(userID primitive.ObjectID, options ...option.Option) (sliceof.Object[model.Circle], error) {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, derp.ValidationError("UserID cannot be zero")
	}

	criteria := exp.Equal("userId", userID)
	return service.Query(criteria, options...)
}

// QueryPrivilegedByUser returns all Circles that are marked as "featured" by the provided userID
func (service *Circle) QueryFeaturedByUser(userID primitive.ObjectID, options ...option.Option) (sliceof.Object[model.Circle], error) {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, derp.ValidationError("UserID cannot be zero")
	}

	criteria := exp.Equal("userId", userID).AndEqual("isFeatured", true)
	return service.Query(criteria, options...)
}

// LoadByID loads a single model.Circle object that matches the provided circleID
func (service *Circle) LoadByID(userID primitive.ObjectID, circleID primitive.ObjectID, result *model.Circle) error {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.ValidationError("UserID cannot be zero")
	}

	// RULE: Require a valid CircleID
	if circleID.IsZero() {
		return derp.ValidationError("CircleID cannot be zero")
	}

	criteria := exp.Equal("_id", circleID).AndEqual("userId", userID)
	return service.Load(criteria, result)
}

func (service *Circle) LoadByProductID(userID primitive.ObjectID, productID primitive.ObjectID, result *model.Circle) error {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.ValidationError("UserID cannot be zero")
	}

	// RULE: Require a valid RemoteToken
	if productID.IsZero() {
		return derp.ValidationError("ProductID cannot be zero")
	}

	criteria := exp.Equal("userId", userID).AndEqual("productIds", productID)
	return service.Load(criteria, result)
}

func (service *Circle) HasAssignedProducts(userID primitive.ObjectID) (bool, error) {

	const location = "service.Circle.CountRemoteProducts"

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return false, derp.ValidationError("UserID cannot be zero")
	}

	// Count the number of remote products for this user
	circles, err := service.QueryFeaturedByUser(userID)

	if err != nil {
		return false, derp.Wrap(err, location, "Error counting remote products for User", userID)
	}

	// Count all products across all "Featured" circles
	for _, circle := range circles {
		if circle.HasProducts() {
			return true, nil
		}
	}

	return false, nil
}

func (service *Circle) AssignedProductIDs(userID primitive.ObjectID) (id.Slice, error) {

	const location = "service.Circle.ProductIDs"

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, derp.ValidationError("UserID cannot be zero")
	}

	// Load all Circles for this User
	circles, err := service.QueryFeaturedByUser(userID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading remote products for User", userID)
	}

	// Collect all of the Remote Product IDs from the Circles
	result := id.NewSlice()

	for _, circle := range circles {
		result = append(result, circle.ProductIDs...)
	}

	result = slice.Unique(result)

	return result, nil
}

/******************************************
 * Custom Behaviors
 ******************************************/

// QueryByUserAsLookupCode returns all Circles owned by the provided userID as a slice of form.LookupCode
func (service *Circle) QueryByUserAsLookupCode(userID primitive.ObjectID, options ...option.Option) ([]form.LookupCode, error) {

	const location = "service.Circle.QueryByUserAsLookupCode"

	// Query for all Circles owned by the user
	circles, err := service.QueryByUser(userID, options...)
	if err != nil {
		return nil, derp.Wrap(err, location, "Error querying Circles by User", userID)
	}

	// Convert the Circles to a slice of lookup codes
	result := slice.Map(circles, func(circle model.Circle) form.LookupCode {
		return circle.LookupCode()
	})

	return result, nil
}

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

	// Save the value to the database
	// Saving directly to the Collection to bypass other validation and business logic.
	if err := service.collection.Save(&circle, "Refreshing Member Count"); err != nil {
		return derp.Wrap(err, location, "Error saving Circle", circle)
	}

	// We have survived another day
	return nil
}
