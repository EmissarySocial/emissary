package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionItem defines a service that can send and receive collectionItem data
type CollectionItem struct {
	importItemService *ImportItem
	host              string
}

// NewCollectionItem returns a fully initialized CollectionItem service
func NewCollectionItem() CollectionItem {
	return CollectionItem{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *CollectionItem) Refresh(factory *Factory) {
	service.host = factory.Host()
	service.importItemService = factory.ImportItem()
}

// Close stops any background processes controlled by this service
func (service *CollectionItem) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *CollectionItem) collection(session data.Session) data.Collection {
	return session.Collection("CollectionItem")
}

// Count returns the number of CollectionItems that match the provided criteria
func (service *CollectionItem) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the CollectionItems that match the provided criteria
func (service *CollectionItem) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.CollectionItem, error) {
	result := make([]model.CollectionItem, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the CollectionItems that match the provided criteria
func (service *CollectionItem) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns an iterator containing all of the CollectionItems who match the provided criteria
func (service *CollectionItem) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.CollectionItem], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.CollectionItem.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewCollectionItem), nil
}

// Load retrieves an CollectionItem from the database
func (service *CollectionItem) Load(session data.Session, criteria exp.Expression, collectionItem *model.CollectionItem) error {

	if err := service.collection(session).Load(notDeleted(criteria), collectionItem); err != nil {
		return derp.Wrap(err, "service.CollectionItem.Load", "Unable to load CollectionItem", criteria)
	}

	return nil
}

// Save adds/updates an CollectionItem in the database
func (service *CollectionItem) Save(session data.Session, collectionItem *model.CollectionItem, note string) error {

	const location = "service.CollectionItem.Save"

	// Validate the value before saving
	if _, err := service.Schema().Validate(collectionItem); err != nil {
		return derp.Wrap(err, location, "Unable to validate CollectionItem", collectionItem)
	}

	// Save the value to the database
	if err := service.collection(session).Save(collectionItem, note); err != nil {
		return derp.Wrap(err, location, "Unable to save CollectionItem", collectionItem, note)
	}

	return nil
}

// Delete removes an CollectionItem from the database (hard delete)
func (service *CollectionItem) Delete(session data.Session, collectionItem *model.CollectionItem, note string) error {

	const location = "service.CollectionItem.Delete"

	// Delete this CollectionItem from the database
	if err := service.collection(session).HardDelete(exp.Equal("_id", collectionItem.CollectionItemID)); err != nil {
		return derp.Wrap(err, location, "Unable to delete CollectionItem", collectionItem)
	}

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly documents that match the provided criteria
func (service *CollectionItem) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific CollectionItem record, without applying any additional business rules
func (service *CollectionItem) HardDeleteByID(session data.Session, userID primitive.ObjectID, collectionItemID primitive.ObjectID) error {

	const location = "service.CollectionItem.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", collectionItemID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete CollectionItem", "userID: "+userID.Hex(), "collectionItemID: "+collectionItemID.Hex())
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *CollectionItem) ObjectType() string {
	return "CollectionItem"
}

// New returns a fully initialized model.CollectionItem as a data.Object.
func (service *CollectionItem) ObjectNew() data.Object {
	result := model.NewCollectionItem()
	return &result
}

func (service *CollectionItem) ObjectID(object data.Object) primitive.ObjectID {

	if collectionItem, ok := object.(*model.CollectionItem); ok {
		return collectionItem.CollectionItemID
	}

	return primitive.NilObjectID
}

func (service *CollectionItem) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *CollectionItem) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewCollectionItem()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *CollectionItem) ObjectSave(session data.Session, object data.Object, note string) error {

	if collectionItem, ok := object.(*model.CollectionItem); ok {
		return service.Save(session, collectionItem, note)
	}
	return derp.Internal("service.CollectionItem.ObjectSave", "Invalid object type", object)
}

func (service *CollectionItem) ObjectDelete(session data.Session, object data.Object, note string) error {
	if collectionItem, ok := object.(*model.CollectionItem); ok {
		return service.Delete(session, collectionItem, note)
	}
	return derp.Internal("service.CollectionItem.ObjectDelete", "Invalid object type", object)
}

func (service *CollectionItem) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.CollectionItem", "Not Authorized")
}

func (service *CollectionItem) Schema() schema.Schema {
	return schema.New(model.CollectionItemSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *CollectionItem) LoadByID(session data.Session, userID primitive.ObjectID, collectionItemID primitive.ObjectID, collectionItem *model.CollectionItem) error {
	criteria := exp.Equal("userId", userID).AndEqual("_id", collectionItemID)
	return service.Load(session, criteria, collectionItem)
}

func (service *CollectionItem) CountByCollection(session data.Session, userID primitive.ObjectID, collectionID primitive.ObjectID, criteria exp.Expression) (int64, error) {
	criteria = criteria.AndEqual("userId", userID).AndEqual("collectionId", collectionID)
	return service.Count(session, criteria)
}

func (service *CollectionItem) RangeByCollection(session data.Session, userID primitive.ObjectID, collectionID primitive.ObjectID, criteria exp.Expression, options ...option.Option) (iter.Seq[model.CollectionItem], error) {
	criteria = criteria.AndEqual("userId", userID).AndEqual("collectionId", collectionID)
	return service.Range(session, criteria, options...)
}

func (service *CollectionItem) QueryByInReplyTo(session data.Session, inReplyTo string, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.CollectionItem], error) {
	criteria = criteria.AndEqual("inReplyTo", inReplyTo)
	return service.Query(session, criteria, options...)
}

func (service *CollectionItem) LoadByURI(session data.Session, collectionID primitive.ObjectID, URI string, collectionItem *model.CollectionItem) error {
	criteria := exp.Equal("collectionId", collectionID).AndEqual("uri", URI)
	return service.Load(session, criteria, collectionItem)
}

func (service *CollectionItem) HardDeleteByURI(session data.Session, URI string) error {
	criteria := exp.Equal("uri", URI)

	// Delete this CollectionItem from the database
	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.CollectionItem.DeleteByURI", "Unable to delete CollectionItem", "uri: "+URI)
	}
	return nil
}

// DeleteByCollectionID deletes all CollectionItems owned by the provided CollectionID
func (service *CollectionItem) DeleteByCollection(session data.Session, userID primitive.ObjectID, collectionID primitive.ObjectID, note string) error {

	const location = "service.CollectionItem.DeleteByCollection"

	// Retrieve all CollectionItems
	collectionItems, err := service.RangeByCollection(session, userID, collectionID, exp.All())

	if err != nil {
		return derp.Wrap(err, location, "Unable to query collection items by CollectionID", "userID: "+userID.Hex(), "collectionID: "+collectionID.Hex())
	}

	// Delete each collection item
	for collectionItem := range collectionItems {
		if err := service.Delete(session, &collectionItem, note); err != nil {
			return derp.Wrap(err, location, "Unable to delete CollectionItem", collectionItem)
		}
	}

	// Success
	return nil
}

/******************************************
 * Custom Behaviors
 ******************************************/

// SaveUnique guarantees that there is only one CollectionItem for a given URI.  It does this
// by removing any existing CollectionItem that matches this URI before saving the new one.
func (service *CollectionItem) SaveUnique(session data.Session, collectionItem *model.CollectionItem, note string) error {

	const location = "service.CollectionItem.SaveUnique"

	// Remove previous URI (if possible)
	if err := service.HardDeleteByURI(session, collectionItem.URI); err != nil {
		return derp.Wrap(err, location, "Unable to delete existing CollectionItem", "uri: "+collectionItem.URI)
	}

	// Insert new CollectionItem
	if err := service.Save(session, collectionItem, note); err != nil {
		return derp.Wrap(err, location, "Unable to save CollectionItem", collectionItem, note)
	}

	// Woot.
	return nil
}

/******************************************
 * Collection Interface
 ******************************************/

func (service *CollectionItem) CollectionCount(session data.Session, userID primitive.ObjectID, collectionID primitive.ObjectID, criteria exp.Expression) collection.CounterFunc {
	return func() (int64, error) {
		return service.CountByCollection(session, userID, collectionID, criteria)
	}
}

// CollectionIterator returns the iterator function for this collection
func (service *CollectionItem) CollectionIterator(session data.Session, userID primitive.ObjectID, collectionID primitive.ObjectID, criteria exp.Expression) collection.IteratorFunc {

	const location = "service.CollectionItem.CollectionIterator"

	return func(startAfter string) (iter.Seq[mapof.Any], error) {

		// Add the "startAfter" criteria (if applicable)
		if startAfter != "" {
			marker := model.NewCollectionItem()
			if err := service.LoadByURI(session, collectionID, startAfter, &marker); err == nil {
				criteria = criteria.AndLessThan("_id", marker.CollectionItemID)
			}
		}

		// Get Replies for this CollectionItem (sorted by insertion date)
		result, err := service.RangeByCollection(
			session,
			userID,
			collectionID,
			criteria,
			option.Fields("_id"),
			option.SortDesc("_id"),
		)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unable to create iterator", "collectionID: "+collectionID.Hex())
		}

		// Map into a range of JSON-LD objects
		return ranges.Map(result, func(item model.CollectionItem) mapof.Any {
			return mapof.Any{
				vocab.PropertyID: item.CollectionItemID.Hex(),
			}
		}), nil
	}
}
