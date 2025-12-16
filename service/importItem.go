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

// ImportItem manages all interactions with the ImportItem collection
type ImportItem struct {
}

// NewImportItem returns a fully populated ImportItem service
func NewImportItem() ImportItem {
	return ImportItem{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *ImportItem) Refresh() {
}

// Close stops any background processes controlled by this service
func (service *ImportItem) Close() {
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *ImportItem) collection(session data.Session) data.Collection {
	return session.Collection("ImportItem")
}

// Count returns the number of records that match the provided criteria
func (service *ImportItem) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Range returns a Go 1.23 RangeFunc that iterates over the Streams that match the provided criteria
func (service *ImportItem) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.ImportItem], error) {

	const location = "service.ImportItem.Range"

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to list records", criteria)
	}

	return RangeFunc(iter, model.NewImportItem), nil
}

func (service *ImportItem) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.ImportItem, error) {
	result := make([]model.ImportItem, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the ImportItems who match the provided criteria
func (service *ImportItem) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Load retrieves an ImportItem from the database
func (service *ImportItem) Load(session data.Session, criteria exp.Expression, result *model.ImportItem) error {
	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.ImportItem.Load", "Unable to load ImportItem", criteria)
	}

	return nil
}

// Save adds/updates an ImportItem in the database
func (service *ImportItem) Save(session data.Session, item *model.ImportItem, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(item); err != nil {
		return derp.Wrap(err, "service.ImportItem.Save", "Error validating ImportItem", item)
	}

	// Save the value to the database
	if err := service.collection(session).Save(item, note); err != nil {
		return derp.Wrap(err, "service.ImportItem.Save", "Unable to save ImportItem", item, note)
	}

	return nil
}

// Delete removes an ImportItem from the database (virtual delete)
func (service *ImportItem) Delete(session data.Session, item *model.ImportItem, note string) error {

	if err := service.collection(session).Delete(item, note); err != nil {
		return derp.Wrap(err, "service.ImportItem.Delete", "Error deleting ImportItem", item, note)
	}

	// TODO: HIGH: Also remove connections to Users that still use this ImportItem
	// TODO: HIGH: Also remove connections to Streams that still use this ImportItem

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *ImportItem) ObjectType() string {
	return "ImportItem"
}

// New returns a fully initialized model.ImportItem as a data.Object.
func (service *ImportItem) ObjectNew() data.Object {
	result := model.NewImportItem()
	return &result
}

func (service *ImportItem) ObjectID(object data.Object) primitive.ObjectID {

	if item, ok := object.(*model.ImportItem); ok {
		return item.ImportItemID
	}

	return primitive.NilObjectID
}

func (service *ImportItem) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *ImportItem) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewImportItem()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *ImportItem) ObjectSave(session data.Session, object data.Object, comment string) error {
	if item, ok := object.(*model.ImportItem); ok {
		return service.Save(session, item, comment)
	}
	return derp.InternalError("service.ImportItem.ObjectSave", "Invalid Object Type", object)
}

func (service *ImportItem) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if item, ok := object.(*model.ImportItem); ok {
		return service.Delete(session, item, comment)
	}
	return derp.InternalError("service.ImportItem.ObjectDelete", "Invalid Object Type", object)
}

func (service *ImportItem) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.ImportItem", "Not Authorized")
}

// Schema returns a validating schema for ImportItems
func (service *ImportItem) Schema() schema.Schema {
	return schema.New(model.ImportItemSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID loads a single model.ImportItem object that matches the provided itemID
func (service *ImportItem) LoadByID(session data.Session, itemID primitive.ObjectID, result *model.ImportItem) error {
	criteria := exp.Equal("_id", itemID)
	return service.Load(session, criteria, result)
}

// LoadByRemoteID loads a single Import record based on the provided UserID and RemoteID
func (service *ImportItem) LoadByRemoteID(session data.Session, userID primitive.ObjectID, remoteID primitive.ObjectID, result *model.ImportItem) error {
	criteria := exp.Equal("remoteId", remoteID).AndEqual("userId", userID)
	return service.Load(session, criteria, result)
}

// LoadByURL loads a single Import record based on the original URL from the remote server
func (service *ImportItem) LoadByRemoteURL(session data.Session, remoteURL string, result *model.ImportItem) error {
	criteria := exp.Equal("remoteUrl", remoteURL)
	return service.Load(session, criteria, result)
}

// LoadNext retrieves the next "NEW" ImportItem from the database
func (service *ImportItem) LoadNext(session data.Session, userID primitive.ObjectID, importID primitive.ObjectID, result *model.ImportItem) error {

	criteria := exp.
		Equal("userId", userID).
		AndEqual("stateId", model.ImportItemStateNew).
		AndEqual("importId", importID)

	return service.Load(session, criteria, result)
}

// RangeByImportID returns an iterator for all ImportItems that match the provided UserID and ImportID
func (service *ImportItem) RangeByImportID(session data.Session, userID primitive.ObjectID, importID primitive.ObjectID) (iter.Seq[model.ImportItem], error) {
	criteria := exp.Equal("userId", userID).AndEqual("importId", importID)
	return service.Range(session, criteria)
}

// DeleteByImportID deletes all ImportItems that match the provided UserID and ImportID
func (service *ImportItem) DeleteByImportID(session data.Session, userID primitive.ObjectID, importID primitive.ObjectID) error {

	const location = "service.ImportItem.DeleteByImportID"

	// Range over all ImportItems in the provided Import
	iterator, err := service.RangeByImportID(session, userID, importID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to retrieve ImportItems")
	}

	// Delete each ImportItem
	for importItem := range iterator {

		criteria := exp.Equal("_id", importItem.ImportItemID).
			AndEqual("importId", importItem.ImportID).
			AndEqual("userId", importItem.UserID)

		if err := service.collection(session).HardDelete(criteria); err != nil {
			return derp.Wrap(err, location, "Unable to delete ImportItem", importItem)
		}
	}

	// Success.
	return nil
}

// MapSourceID looks up the ID of an imported record and returns the local ID
func (service *ImportItem) mapRemoteID(session data.Session, userID primitive.ObjectID, value *primitive.ObjectID) error {

	const location = "service.ImportItem.mapRemoteID"

	importItem := model.NewImportItem()

	// Load the ImportItem using the value as the sourceID
	if err := service.LoadByRemoteID(session, userID, *value, &importItem); err != nil {
		return derp.Wrap(err, location, "Unable to load Import Item", "userID: "+userID.Hex(), "remoteID: "+value.Hex())
	}

	// Set the value to the localID
	*value = importItem.LocalID

	return nil
}
