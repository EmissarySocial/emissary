package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

// collection returns the Collection collection for the provided database session
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

	// Delete CollectionItems that are part of this Collection. DeleteByCollection filters on
	// parentId, which items inherit from collection.ParentID (NOT UserID) via AddItem.
	if err := service.collectionItemService.DeleteByCollection(session, collection.ParentID, collection.CollectionID, note); err != nil {
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
		return derp.Wrap(err, location, "Deleting Collection", "userID: "+userID.Hex(), "collectionID: "+collectionID.Hex())
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

// ObjectID returns the unique ID of the provided Collection. Implements the ModelService interface.
func (service *Collection) ObjectID(object data.Object) primitive.ObjectID {

	if collection, ok := object.(*model.Collection); ok {
		return collection.CollectionID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every Collection that matches the provided criteria. Implements the ModelService interface.
func (service *Collection) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single Collection as a data.Object. Implements the ModelService interface.
func (service *Collection) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewCollection()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds or updates a Collection in the database. Implements the ModelService interface.
func (service *Collection) ObjectSave(session data.Session, object data.Object, note string) error {

	if collection, ok := object.(*model.Collection); ok {
		return service.Save(session, collection, note)
	}
	return derp.Internal("service.Collection.ObjectSave", "Invalid object type", object)
}

// ObjectDelete marks a Collection as deleted. Implements the ModelService interface.
func (service *Collection) ObjectDelete(session data.Session, object data.Object, note string) error {
	if collection, ok := object.(*model.Collection); ok {
		return service.Delete(session, collection, note)
	}
	return derp.Internal("service.Collection.ObjectDelete", "Invalid object type", object)
}

// ObjectUserCan reports whether the provided Authorization may run an action on a Collection. Implements the ModelService interface.
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

// LoadByType returns the single Collection that matches the provided ParentID and Type
func (service *Collection) LoadByType(session data.Session, parentID primitive.ObjectID, collectionType string, collection *model.Collection) error {
	criteria := exp.Equal("parentId", parentID).AndEqual("collectionType", collectionType)
	return service.Load(session, criteria, collection)
}

// RangeByUserID returns a RangeFunc that yields all Collections owned by the provided UserID
func (service *Collection) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.Collection], error) {
	criteria := exp.Equal("userId", userID)
	return service.Range(session, criteria)
}

// DeleteByURL marks the Collection at the provided URL as deleted, ignoring URLs that are not local
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
		return derp.Wrap(err, location, "Querying Collections by UserID", "userID: "+userID.Hex())
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

// ActivityPubURL returns the ActivityPub URL of the provided Collection
func (service *Collection) ActivityPubURL(userID primitive.ObjectID, collectionID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex() + "/pub/collections/" + collectionID.Hex()
}

/******************************************
 * Other Methods
 ******************************************/

// LoadOrCreateByStream retrieves a Stream's Collection of the provided type, creating it if it does not exist
func (service *Collection) LoadOrCreateByStream(session data.Session, stream *model.Stream, collectionType string) (model.Collection, error) {

	return service.loadOrCreateByParent(
		session,
		stream.AttributedTo.UserID,
		model.CollectionParentTypeStream,
		stream.StreamID,
		collectionType,
		sliceof.String{vocab.NamespacePublic},
		sliceof.String{vocab.NamespacePublic},
	)
}

// LoadOrCreateByUser retrieves a User's Collection of the provided type, creating it if it does not exist
func (service *Collection) LoadOrCreateByUser(session data.Session, user *model.User, collectionType string) (model.Collection, error) {

	return service.loadOrCreateByParent(
		session,
		user.UserID,
		model.CollectionParentTypeUser,
		user.UserID,
		collectionType,
		sliceof.String{vocab.NamespacePublic},
		sliceof.String{vocab.NamespacePublic},
	)
}

// loadOrCreateByParent returns the Collection for the given (parentID, collectionType), creating it
// just-in-time on first use. The read/write permission lists are applied ONLY when the collection is
// created; an existing collection keeps its own. Callers should use the LoadOrCreateByUser /
// LoadOrCreateByStream wrappers rather than calling this directly.
func (service *Collection) loadOrCreateByParent(session data.Session, userID primitive.ObjectID, parentType string, parentID primitive.ObjectID, collectionType string, read sliceof.String, write sliceof.String) (model.Collection, error) {

	const location = "service.Collection.loadOrCreateByParent"

	// Concurrency-safe: when two callers race to create the same collection, the unique index on
	// (parentId, collectionType) rejects the loser's insert and that caller re-loads the winner.
	// See queries/sync/collection.go for the index, and COLLECTIONS-REDESIGN.md (D2).
	collection := model.NewCollection()

	// Fast path: the collection almost always already exists
	if err := service.LoadByType(session, parentID, collectionType, &collection); err != nil {

		// Anything other than NotFound is a real failure; a NotFound falls through to create one
		if !derp.IsNotFound(err) {
			return collection, derp.Wrap(err, location, "Loading Collection", parentType, parentID, collectionType)
		}
	} else {
		return collection, nil
	}

	// Slow path: no collection yet, so try to create one
	collection.UserID = userID
	collection.ParentType = parentType
	collection.ParentID = parentID
	collection.CollectionType = collectionType
	collection.Read = read
	collection.Write = write

	if err := service.Save(session, &collection, ""); err != nil {

		// A lost creation race trips the unique index, which data-mongo reports as a Conflict.
		// Re-load the winner's record, which is now guaranteed to exist.
		if derp.IsConflict(err) {

			if err := service.LoadByType(session, parentID, collectionType, &collection); err != nil {
				return collection, derp.Wrap(err, location, "Re-loading Collection after duplicate-key conflict", parentType, parentID, collectionType)
			}

			return collection, nil
		}

		return collection, derp.Wrap(err, location, "Creating Collection", parentType, parentID, collectionType)
	}

	return collection, nil
}

// AddItem adds a URI to the provided Collection as a new CollectionItem
func (service *Collection) AddItem(session data.Session, collection *model.Collection, itemURI string, inReplyTo string) error {

	const location = "service.Collection.AddItem"

	// Create a new CollectionItem record
	collectionItem := model.NewCollectionItem()
	collectionItem.CollectionID = collection.CollectionID
	collectionItem.UserID = collection.UserID
	collectionItem.ParentID = collection.ParentID
	collectionItem.CollectionType = collection.CollectionType
	collectionItem.URI = itemURI

	// Save the CollectionItem to the database
	if err := service.collectionItemService.SaveUnique(session, &collectionItem, ""); err != nil {
		return derp.Wrap(err, location, "Saving collection item", collectionItem)
	}

	// Keep the W3C-required `totalItems` accurate (D9).
	if err := service.refreshTotalItems(session, collection); err != nil {
		return derp.Wrap(err, location, "Refreshing collection totalItems", collection.CollectionID.Hex())
	}

	// Success
	return nil
}

// ProjectResponse adds or removes itemURI in the given Stream's Like/Dislike/Share collection, and
// returns the affected collection plus its collection type
func (service *Collection) ProjectResponse(session data.Session, stream *model.Stream, responseType string, itemURI string, add bool) (model.Collection, string, error) {

	const location = "service.Collection.ProjectResponse"

	// This is the side-agnostic projection primitive shared by BOTH the actor-side funnel
	// (service.Response, for local reactions) and the object-side handlers (inbound remote ones).
	// Everything around this call differs; the projection itself is identical, so it lives here once.

	// itemURI is the caller's choice of stable key: local reactions pass response.ActivityPubURL(),
	// remote ones pass the inbound activity's own ID.  The count field lives on the Stream, so the
	// caller refreshes it from the returned collection (see Stream.refreshResponseCount).

	// RULE: Only Like/Dislike/Announce project into a per-Stream collection.
	collectionType := model.CollectionTypeForResponse(responseType)

	if collectionType == "" {
		return model.Collection{}, "", nil
	}

	// RULE: Without an item URL there is nothing to project.
	if itemURI == "" {
		return model.Collection{}, "", nil
	}

	// On ADD, JIT the collection (concurrency-safe via the unique index) and add the item.
	if add {

		collection, err := service.LoadOrCreateByStream(session, stream, collectionType)

		if err != nil {
			return collection, collectionType, derp.Wrap(err, location, "Loading/Creating response collection", "type: "+collectionType, "streamID: "+stream.StreamID.Hex())
		}

		if err := service.AddItem(session, &collection, itemURI, stream.URL); err != nil {
			return collection, collectionType, derp.Wrap(err, location, "Adding response to collection", "itemURI: "+itemURI)
		}

		return collection, collectionType, nil
	}

	// On REMOVE, load the existing collection. If none exists, there is nothing to remove.
	collection := model.NewCollection()

	if err := service.LoadByType(session, stream.StreamID, collectionType, &collection); err != nil {

		if derp.IsNotFound(err) {
			return model.Collection{}, "", nil
		}

		return collection, collectionType, derp.Wrap(err, location, "Loading response collection", "type: "+collectionType, "streamID: "+stream.StreamID.Hex())
	}

	if err := service.RemoveItem(session, &collection, itemURI); err != nil {
		return collection, collectionType, derp.Wrap(err, location, "Removing response from collection", "itemURI: "+itemURI)
	}

	return collection, collectionType, nil
}

// RemoveItem removes the CollectionItem identified by itemURI from the provided Collection.
// It is a no-op when no such item exists.
func (service *Collection) RemoveItem(session data.Session, collection *model.Collection, itemURI string) error {

	const location = "service.Collection.RemoveItem"

	collectionItem := model.NewCollectionItem()

	if err := service.collectionItemService.LoadByURI(session, collection.CollectionID, itemURI, &collectionItem); err != nil {

		// An item that is already gone needs no removing
		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading CollectionItem", "uri: "+itemURI)
	}

	if err := service.collectionItemService.Delete(session, &collectionItem, ""); err != nil {
		return derp.Wrap(err, location, "Deleting CollectionItem", collectionItem)
	}

	// Keep the W3C-required `totalItems` accurate (D9).
	if err := service.refreshTotalItems(session, collection); err != nil {
		return derp.Wrap(err, location, "Refreshing collection totalItems", collection.CollectionID.Hex())
	}

	return nil
}

// CountItems returns the number of (live) CollectionItems in the provided Collection.
func (service *Collection) CountItems(session data.Session, collection *model.Collection) (int64, error) {
	// CountByCollection filters on parentId, which items inherit from collection.ParentID (NOT UserID) via AddItem.
	return service.collectionItemService.CountByCollection(session, collection.ParentID, collection.CollectionID, exp.All())
}

// refreshTotalItems recomputes the Collection's `totalItems` from the live CollectionItem rows and
// Saves it (D9). Uses recompute-and-Save (never increment) because there is no atomic $inc — a stale
// overwrite self-heals on the next add/remove (D4). The in-memory `collection.TotalItems` is updated
// too, so callers holding the pointer see the fresh value. `totalItems` is part of the W3C collection
// definition and MUST stay accurate for every add/remove on every collection type.
func (service *Collection) refreshTotalItems(session data.Session, collection *model.Collection) error {

	const location = "service.Collection.refreshTotalItems"

	count, err := service.CountItems(session, collection)

	if err != nil {
		return derp.Wrap(err, location, "Counting CollectionItems", collection.CollectionID.Hex())
	}

	collection.TotalItems = int(count)

	if err := service.Save(session, collection, "Refresh totalItems"); err != nil {
		return derp.Wrap(err, location, "Saving Collection", collection.CollectionID.Hex())
	}

	return nil
}
