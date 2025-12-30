package service

import (
	"iter"

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
	importItemService *ImportItem
	privilegeService  *Privilege
}

// NewCircle returns a fully populated Circle service
func NewCircle() Circle {
	return Circle{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Circle) Refresh(importItemService *ImportItem, privilegeService *Privilege) {
	service.importItemService = importItemService
	service.privilegeService = privilegeService
}

// Close stops any background processes controlled by this service
func (service *Circle) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Circle) collection(session data.Session) data.Collection {
	return session.Collection("Circle")
}

// Count returns the number of records that match the provided criteria
func (service *Circle) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice of Circles that match the provided criteria
func (service *Circle) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Circle, error) {
	result := make([]model.Circle, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Circles who match the provided criteria
func (service *Circle) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Circles that match the provided criteria
func (service *Circle) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Circle], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Circle.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewCircle), nil
}

// Load retrieves an Circle from the database
func (service *Circle) Load(session data.Session, criteria exp.Expression, result *model.Circle) error {

	const location = "service.Circle.Load"

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, location, "Unable to load Circle", criteria)
	}

	return nil
}

// Save adds/updates an Circle in the database
func (service *Circle) Save(session data.Session, circle *model.Circle, note string) error {

	const location = "service.Circle.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(circle); err != nil {
		return derp.Wrap(err, location, "Unable to validate Circle", circle)
	}

	// Save the value to the database
	if err := service.collection(session).Save(circle, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Circle", circle, note)
	}

	// Recalculate privileges based on new Circle settings.
	if err := service.privilegeService.RefreshCircleInfo(session, circle); err != nil {
		return derp.Wrap(err, location, "Unable to refresh Privileges for Circle", circle.CircleID, note)
	}

	return nil
}

// Delete removes an Circle from the database (virtual delete)
func (service *Circle) Delete(session data.Session, circle *model.Circle, note string) error {

	const location = "service.Circle.Delete"

	if err := service.collection(session).Delete(circle, note); err != nil {
		return derp.Wrap(err, location, "Unable to delete Circle", circle, note)
	}

	if err := service.privilegeService.DeleteByCircle(session, circle.CircleID, note); err != nil {
		return derp.Wrap(err, location, "Unable to delete Privileges for Circle", circle.CircleID, note)
	}

	// TODO: HIGH: Also remove connections to Streams that still use this Circle

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *Circle) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Circle record, without applying any additional business rules
func (service *Circle) HardDeleteByID(session data.Session, userID primitive.ObjectID, circleID primitive.ObjectID) error {

	const location = "service.Circle.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", circleID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Circle", "userID: "+userID.Hex(), "circleID: "+circleID.Hex())
	}

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

func (service *Circle) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Circle) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewCircle()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Circle) ObjectSave(session data.Session, object data.Object, comment string) error {
	if circle, ok := object.(*model.Circle); ok {
		return service.Save(session, circle, comment)
	}
	return derp.Internal("service.Circle.ObjectSave", "Invalid Object Type", object)
}

func (service *Circle) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if circle, ok := object.(*model.Circle); ok {
		return service.Delete(session, circle, comment)
	}
	return derp.Internal("service.Circle.ObjectDelete", "Invalid Object Type", object)
}

func (service *Circle) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Circle", "Not Authorized")
}

func (service *Circle) Schema() schema.Schema {
	return schema.New(model.CircleSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Circle) QueryByIDs(session data.Session, userID primitive.ObjectID, circleIDs []primitive.ObjectID, options ...option.Option) ([]model.Circle, error) {

	const location = "service.Circle.QueryByIDs"

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, derp.Validation("UserID cannot be zero")
	}

	// RULE: Require at least one CircleID
	if len(circleIDs) == 0 {
		return nil, derp.Validation("CircleIDs cannot be empty")
	}

	criteria := exp.In("_id", circleIDs).AndEqual("userId", userID)

	// Load the Merchant Accounts for this User
	result, err := service.Query(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to load merchant accounts")
	}

	return result, nil
}

// QueryByUser returns all Circles that are owned by the provided userID
func (service *Circle) QueryByUser(session data.Session, userID primitive.ObjectID, options ...option.Option) (sliceof.Object[model.Circle], error) {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, derp.Validation("UserID cannot be zero")
	}

	criteria := exp.Equal("userId", userID)
	return service.Query(session, criteria, options...)
}

// QueryPrivilegedByUser returns all Circles that are marked as "featured" by the provided userID
func (service *Circle) QueryFeaturedByUser(session data.Session, userID primitive.ObjectID, options ...option.Option) (sliceof.Object[model.Circle], error) {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, derp.Validation("UserID cannot be zero")
	}

	criteria := exp.Equal("userId", userID).AndEqual("isFeatured", true)
	return service.Query(session, criteria, options...)
}

// LoadByID loads a single model.Circle object that matches the provided circleID
func (service *Circle) LoadByID(session data.Session, userID primitive.ObjectID, circleID primitive.ObjectID, result *model.Circle) error {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: Require a valid CircleID
	if circleID.IsZero() {
		return derp.Validation("CircleID cannot be zero")
	}

	criteria := exp.Equal("_id", circleID).AndEqual("userId", userID)
	return service.Load(session, criteria, result)
}

func (service *Circle) LoadByProductID(session data.Session, userID primitive.ObjectID, productID primitive.ObjectID, result *model.Circle) error {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: Require a valid RemoteToken
	if productID.IsZero() {
		return derp.Validation("ProductID cannot be zero")
	}

	criteria := exp.Equal("userId", userID).AndEqual("productIds", productID)
	return service.Load(session, criteria, result)
}

// RangeByUserID returns a RangeFunc that yields all Circles owned by the provided UserID
func (service *Circle) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.Circle], error) {
	criteria := exp.Equal("userId", userID)
	return service.Range(session, criteria)
}

// DeleteByUserID deletes all Circles owned by the provided UserID
func (service *Circle) DeleteByUserID(session data.Session, userID primitive.ObjectID, note string) error {

	const location = "service.Circle.DeleteByUserID"

	// Retrieve all Circles
	circles, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to query Circles by UserID", userID)
	}

	// Delete each circle
	for circle := range circles {
		if err := service.Delete(session, &circle, note); err != nil {
			return derp.Wrap(err, location, "Unable to delete Circle", circle)
		}
	}

	// Success
	return nil
}

func (service *Circle) HasProducts(session data.Session, userID primitive.ObjectID) (bool, error) {

	count, err := service.ProductCount(session, userID)

	if err != nil {
		return false, derp.Wrap(err, "service.Circle.HasProducts", "Error counting products for User", userID)
	}

	return count > 0, nil
}

func (service *Circle) ProductCount(session data.Session, userID primitive.ObjectID) (int, error) {

	const location = "service.Circle.ProductCount"

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return 0, derp.Validation("UserID cannot be zero")
	}

	// Count the number of remote products for this user
	circles, err := service.QueryFeaturedByUser(session, userID)

	if err != nil {
		return 0, derp.Wrap(err, location, "Error counting remote products for User", userID)
	}

	// Count all products across all "Featured" circles
	result := 0
	for _, circle := range circles {
		result += circle.ProductCount()
	}
	return result, nil
}

func (service *Circle) AssignedProductIDs(session data.Session, userID primitive.ObjectID) (id.Slice, error) {

	const location = "service.Circle.AssignedProductIDs"

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, derp.Validation("UserID cannot be zero")
	}

	// Load all Circles for this User
	circles, err := service.QueryFeaturedByUser(session, userID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to load remote products for User", userID)
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
func (service *Circle) QueryByUserAsLookupCode(session data.Session, userID primitive.ObjectID, options ...option.Option) ([]form.LookupCode, error) {

	const location = "service.Circle.QueryByUserAsLookupCode"

	// Query for all Circles owned by the user
	circles, err := service.QueryByUser(session, userID, options...)
	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to query Circles by User", userID)
	}

	// Convert the Circles to a slice of lookup codes
	result := slice.Map(circles, func(circle model.Circle) form.LookupCode {
		return circle.LookupCode()
	})

	return result, nil
}

// RefreshMemberCounts updates the member counts for the Circle with the provided circleID
func (service *Circle) RefreshMemberCounts(session data.Session, userID primitive.ObjectID, circleID primitive.ObjectID) error {

	const location = "service.Circle.RefreshMemberCounts"

	// Load the circle to ensure it exists
	circle := model.NewCircle()
	if err := service.LoadByID(session, userID, circleID, &circle); err != nil {
		if derp.IsNotFound(err) {
			return nil
		}
		return derp.Wrap(err, location, "Unable to load Circle", circleID)
	}

	// Count the number of privileges for this Circle
	count, err := service.privilegeService.CountByCircle(session, circleID)

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
	if err := service.collection(session).Save(&circle, "Refreshing Member Count"); err != nil {
		return derp.Wrap(err, location, "Unable to save Circle", circle)
	}

	// We have survived another day
	return nil
}
