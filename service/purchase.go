package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Purchase defines a service that manages all content purchases created and imported by Users.
type Purchase struct {
	collection data.Collection
}

// NewPurchase returns a fully initialized Purchase service
func NewPurchase() Purchase {
	return Purchase{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Purchase) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Purchase) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Purchase) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe Purchases that match the provided criteria
func (service *Purchase) Query(criteria exp.Expression, options ...option.Option) ([]model.Purchase, error) {
	result := make([]model.Purchase, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Purchases that match the provided criteria
func (service *Purchase) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Purchase records that match the provided criteria
func (service *Purchase) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Purchase], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Purchase.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewPurchase), nil
}

// Load retrieves an Purchase from the database
func (service *Purchase) Load(criteria exp.Expression, purchase *model.Purchase) error {

	if err := service.collection.Load(notDeleted(criteria), purchase); err != nil {
		return derp.Wrap(err, "service.Purchase.Load", "Error loading Purchase", criteria)
	}

	return nil
}

// Save adds/updates an Purchase in the database
func (service *Purchase) Save(purchase *model.Purchase, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(purchase); err != nil {
		return derp.Wrap(err, "service.Purchase.Save", "Error validating Purchase", purchase)
	}

	// Save the purchase to the database
	if err := service.collection.Save(purchase, note); err != nil {
		return derp.Wrap(err, "service.Purchase.Save", "Error saving Purchase", purchase, note)
	}

	return nil
}

// Delete removes an Purchase from the database (virtual delete)
func (service *Purchase) Delete(purchase *model.Purchase, note string) error {

	// Delete this Purchase
	if err := service.collection.Delete(purchase, note); err != nil {
		return derp.Wrap(err, "service.Purchase.Delete", "Error deleting Purchase", purchase, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Purchase) ObjectType() string {
	return "Purchase"
}

// New returns a fully initialized model.Purchase as a data.Object.
func (service *Purchase) ObjectNew() data.Object {
	result := model.NewPurchase()
	return &result
}

func (service *Purchase) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Purchase); ok {
		return mention.PurchaseID
	}

	return primitive.NilObjectID
}

func (service *Purchase) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Purchase) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewPurchase()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Purchase) ObjectSave(object data.Object, comment string) error {
	if purchase, ok := object.(*model.Purchase); ok {
		return service.Save(purchase, comment)
	}
	return derp.InternalError("service.Purchase.ObjectSave", "Invalid Object Type", object)
}

func (service *Purchase) ObjectDelete(object data.Object, comment string) error {
	if purchase, ok := object.(*model.Purchase); ok {
		return service.Delete(purchase, comment)
	}
	return derp.InternalError("service.Purchase.ObjectDelete", "Invalid Object Type", object)
}

func (service *Purchase) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Purchase.ObjectUserCan", "Not Authorized")
}

func (service *Purchase) Schema() schema.Schema {
	return schema.New(model.PurchaseSchema())
}

/******************************************
 * Custom Actions
 ******************************************/

// Sync creates/updates each purchase in the provided list.  Purchases are matched
// on `RemoteGuestID` and `RemoteProductID`
func (service *Purchase) Sync(purchases ...model.Purchase) error {

	const location = "service.Purchase.Sync"

	for _, purchase := range purchases {

		if err := service.CreateOrUpdate(&purchase); err != nil {
			return derp.Wrap(err, location, "Error loading or creating purchase")
		}
	}

	return nil
}

// CreateOrUpdate creates a new purchase or updates an existing purchase,
// matching on `RemoteGuestID` and `RemoteProductID`.
func (service *Purchase) CreateOrUpdate(purchase *model.Purchase) error {

	// Try to load the purchase by email address
	currentPurchase := model.NewPurchase()

	// Try to find a current, matching purchase record
	if err := service.LoadByRemoteIDs(purchase.RemoteGuestID, purchase.RemoteProductID, &currentPurchase); !derp.IsNilOrNotFound(err) {
		return derp.Wrap(err, "service.Purchase.CreateOrUpdate", "Error loading purchase")
	}

	// Try to update the current purchase.  Complete if no changes were made.
	if changed := currentPurchase.UpdateWith(purchase); !changed {
		return nil
	}

	// Save the changes to the current purchase
	if err := service.Save(&currentPurchase, "Updated"); err != nil {
		return derp.Wrap(err, "service.Purchase.CreateOrUpdate", "Error saving purchase", currentPurchase)
	}

	// Success.
	return nil
}

/******************************************
 * Custom Queries
 ******************************************/

// CountByGuestAndProduct returns the number of purchases made by a guest for a list of products
func (service *Purchase) CountByGuestAndProduct(guestID primitive.ObjectID, productIDs ...primitive.ObjectID) (int64, error) {

	const location = "service.Purchase.CountByGuestAndProduct"

	// RULE: GuestID must be provided
	if guestID.IsZero() {
		return 0, derp.InternalError(location, "No guestID provided")
	}

	// RULE: At least one productID must be provided
	if len(productIDs) == 0 {
		return 0, derp.InternalError(location, "No productIDs provided")
	}

	criteria := exp.Equal("guestId", guestID).AndIn("productId", productIDs)

	return service.Count(criteria)
}

// LoadByRemoteIDs retrieves a purchase using the remote IDs for the user, product, and purchase
func (service *Purchase) LoadByRemoteIDs(remoteUserID string, remoteProductID string, purchase *model.Purchase) error {
	criteria := exp.Equal("remoteUserId", remoteUserID).
		AndEqual("remoteProductId", remoteProductID)

	return service.Load(criteria, purchase)
}
