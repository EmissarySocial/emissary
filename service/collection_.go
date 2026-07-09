package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Collection defines a service that can send and receive collection data
type Collection struct {
	collectionItemService *CollectionItem
	importItemService     *ImportItem
	locatorService        *Locator
	host                  string
}

// NewCollection returns a fully initialized Collection service
func NewCollection() Collection {
	return Collection{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Collection) Refresh(factory *Factory) {
	service.collectionItemService = factory.CollectionItem()
	service.importItemService = factory.ImportItem()
	service.locatorService = factory.Locator()
	service.host = factory.Host()
}

// Close stops any background processes controlled by this service
func (service *Collection) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Collection) collection(session data.Session) data.Collection {
	return session.Collection("Collection")
}

// Count returns the number of Collections that match the provided criteria
func (service *Collection) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Collections that match the provided criteria
func (service *Collection) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Collection, error) {
	result := make([]model.Collection, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Collections that match the provided criteria
func (service *Collection) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns an iterator containing all of the Collections that match the provided criteria
func (service *Collection) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Collection], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Collection.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewCollection), nil
}

// Load retrieves an Collection from the database
func (service *Collection) Load(session data.Session, criteria exp.Expression, collection *model.Collection) error {

	if err := service.collection(session).Load(notDeleted(criteria), collection); err != nil {
		return derp.Wrap(err, "service.Collection.Load", "Loading Collection", criteria)
	}

	return nil
}

// Save adds/updates an Collection in the database
func (service *Collection) Save(session data.Session, collection *model.Collection, note string) error {

	const location = "service.Collection.Save"

	// Validate the value before saving
	if _, err := service.Schema().Validate(collection); err != nil {
		return derp.Wrap(err, location, "Validating Collection", collection)
	}

	// Save the value to the database
	if err := service.collection(session).Save(collection, note); err != nil {
		return derp.Wrap(err, location, "Saving Collection", collection, note)
	}

	return nil
}

// Delete removes an Collection from the database (hard delete)
func (service *Collection) Delete(session data.Session, collection *model.Collection, note string) error {

	const location = "service.Collection.Delete"

	// Delete CollectionItems that are part of this Collection
	if err := service.collectionItemService.DeleteByCollection(session, collection.UserID, collection.CollectionID, note); err != nil {
		return derp.Wrap(err, location, "Deleting CollectionItems for Collection", collection.CollectionID.Hex())
	}

	// Delete this Collection
	if err := service.collection(session).HardDelete(exp.Equal("_id", collection.CollectionID)); err != nil {
		return derp.Wrap(err, location, "Deleting Collection", collection)
	}

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *Collection) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Collection record, without applying any additional business rules
func (service *Collection) HardDeleteByID(session data.Session, userID primitive.ObjectID, collectionID primitive.ObjectID) error {

	const location = "service.Collection.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", collectionID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Collection", "userID: "+userID.Hex(), "collectionID: "+collectionID.Hex())
	}

	return nil
}

/******************************************
 * Generic Data Methods
******************************************/

// ObjectType returns the type of object that this service manages
func (service *Collection) ObjectType() string {
	return "Collection"
}

// New returns a fully initialized model.Collection as a data.Object.
func (service *Collection) ObjectNew() data.Object {
	result := model.NewCollection()
	return &result
}

func (service *Collection) ObjectID(object data.Object) primitive.ObjectID {

	if collection, ok := object.(*model.Collection); ok {
		return collection.CollectionID
	}

	return primitive.NilObjectID
}

func (service *Collection) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Collection) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewCollection()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Collection) ObjectSave(session data.Session, object data.Object, note string) error {

	if collection, ok := object.(*model.Collection); ok {
		return service.Save(session, collection, note)
	}
	return derp.Internal("service.Collection.ObjectSave", "Invalid object type", object)
}

func (service *Collection) ObjectDelete(session data.Session, object data.Object, note string) error {
	if collection, ok := object.(*model.Collection); ok {
		return service.Delete(session, collection, note)
	}
	return derp.Internal("service.Collection.ObjectDelete", "Invalid object type", object)
}

func (service *Collection) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Collection", "Not Authorized")
}

// Schema returns the validating schema for all Collection records
func (service *Collection) Schema() schema.Schema {
	return schema.New(model.CollectionSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID returns the single Collection that matches the provided CollectionID and UserID
func (service *Collection) LoadByID(session data.Session, userID primitive.ObjectID, collectionID primitive.ObjectID, collection *model.Collection) error {
	criteria := exp.Equal("userId", userID).AndEqual("_id", collectionID)
	return service.Load(session, criteria, collection)
}

// RangeByUserID returns a RangeFunc that yields all Collections owned by the provided UserID
func (service *Collection) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.Collection], error) {
	criteria := exp.Equal("userId", userID)
	return service.Range(session, criteria)
}

// RangeByCollectionID returns a RangeFunc that yields all Collections owned by the provided CollectionID
func (service *Collection) RangeByCollectionID(session data.Session, collectionID primitive.ObjectID) (iter.Seq[model.Collection], error) {
	criteria := exp.Equal("_id", collectionID)
	return service.Range(session, criteria)
}

func (service *Collection) DeleteByURL(session data.Session, url string) error {

	// Try to parse the Collection URL as a local collection
	userID, collectionID, err := service.locatorService.ParseCollection(url)

	// If it doesn't parse, then it's not a local collection
	if err != nil {
		return nil
	}

	// Try to load the Collection from the database
	collection := model.NewCollection()

	if err := service.LoadByID(session, userID, collectionID, &collection); err != nil {
		return derp.Wrap(err, "service.Collection.DeleteByURL", "Loading collection", "url: "+url)
	}

	// Delete the Collection
	if err := service.Delete(session, &collection, "Deleted by URL"); err != nil {
		return derp.Wrap(err, "service.Collection.DeleteByURL", "Deleting collection", "url: "+url)
	}

	// Success.
	return nil
}

// DeleteByUserID deletes all CollectionItems owned by the provided UserID
func (service *Collection) DeleteByUserID(session data.Session, userID primitive.ObjectID, note string) error {

	const location = "service.Collection.DeleteByUserID"

	// Retrieve all Collections
	collections, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to query collections by UserID", "userID: "+userID.Hex())
	}

	// Delete each collection
	for collection := range collections {
		if err := service.Delete(session, &collection, note); err != nil {
			return derp.Wrap(err, location, "Deleting Collection", collection)
		}
	}

	// Success
	return nil
}

/******************************************
 * Data Getters
 ******************************************/

func (service *Collection) ActivityPubURL(userID primitive.ObjectID, collectionID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex() + "/pub/collections/" + collectionID.Hex()
}

/******************************************
 * Other Methods
 ******************************************/

// LoadByParentAndType loads the single Collection identified by the (parentID, type) pair.
func (service *Collection) LoadByParentAndType(session data.Session, parentID primitive.ObjectID, collectionType string, collection *model.Collection) error {
	criteria := exp.Equal("parentId", parentID).AndEqual("type", collectionType)
	return service.Load(session, criteria, collection)
}

// LoadOrCreateByParent returns the Collection for the given (parentID, type), creating it just-in-time on first use.
func (service *Collection) LoadOrCreateByParent(session data.Session, ownerID primitive.ObjectID, parentID primitive.ObjectID, collectionType string, collection *model.Collection) error {

	const location = "service.Collection.LoadOrCreateByParent"

	// This is concurrency-safe: when two callers race to create the same collection,
	// the unique index on (parentId, type) rejects the loser's insert with a
	// duplicate-key error, and that caller re-loads the winner instead of duplicating.
	// See queries/sync/context.go for the index, and COLLECTIONS-REDESIGN.md (D2) for
	// why this replaces the racy load-then-save idiom used elsewhere.
	parentDetail := "parentID: " + parentID.Hex()
	typeDetail := "type: " + collectionType

	// Fast path: the collection almost always already exists.
	err := service.LoadByParentAndType(session, parentID, collectionType, collection)

	if err == nil {
		return nil
	}

	// Any error other than "not found" is a real failure.
	if !derp.IsNotFound(err) {
		return derp.Wrap(err, location, "Unable to load Collection", parentDetail, typeDetail)
	}

	// Slow path: no collection yet, so try to create one.
	*collection = model.NewCollection()
	collection.UserID = ownerID
	collection.ParentID = parentID
	collection.Type = collectionType

	saveErr := service.Save(session, collection, "Created just-in-time")

	if saveErr == nil {
		return nil
	}

	// If we lost a creation race, the unique index rejects our insert. Re-load
	// the winner's record (which is now guaranteed to exist) and return it.
	if mongo.IsDuplicateKeyError(saveErr) {

		if reloadErr := service.LoadByParentAndType(session, parentID, collectionType, collection); reloadErr != nil {
			return derp.Wrap(reloadErr, location, "Unable to re-load Collection after duplicate-key conflict", parentDetail, typeDetail)
		}

		return nil
	}

	// Any other save error is a real failure.
	return derp.Wrap(saveErr, location, "Unable to create Collection", parentDetail, typeDetail)
}

// AddItem adds a URI to the provided Collection as a new CollectionItem
func (service *Collection) AddItem(session data.Session, collection *model.Collection, itemURI string, inReplyTo string) error {

	const location = "service.Collection.AddItem"

	// Create a new CollectionItem record
	collectionItem := model.NewCollectionItem()
	collectionItem.UserID = collection.UserID
	collectionItem.CollectionID = collection.CollectionID
	collectionItem.URI = itemURI
	collectionItem.InReplyTo = inReplyTo

	// Save the CollectionItem to the database
	if err := service.collectionItemService.SaveUnique(session, &collectionItem, ""); err != nil {
		return derp.Wrap(err, location, "Saving collection item", collectionItem)
	}

	// Success
	return nil
}
