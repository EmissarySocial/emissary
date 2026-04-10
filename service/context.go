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
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Context defines a service that can send and receive objectLink data
type Context struct {
	host string
}

// NewContext returns a fully initialized Context service
func NewContext() Context {
	return Context{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Context) Refresh(factory *Factory) {
	service.host = factory.Host()
}

// Close stops any background processes controlled by this service
func (service *Context) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Context) collection(session data.Session) data.Collection {
	return session.Collection("Context")
}

// Count returns the number of Contexts that match the provided criteria
func (service *Context) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Contexts that match the provided criteria
func (service *Context) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.ObjectLink, error) {
	result := make([]model.ObjectLink, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Contexts that match the provided criteria
func (service *Context) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns an iterator containing all of the ObjectLinks who match the provided criteria
func (service *Context) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.ObjectLink], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.ObjectLink.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewObjectLink), nil
}

// Load retrieves an ObjectLink from the database
func (service *Context) Load(session data.Session, criteria exp.Expression, objectLink *model.ObjectLink) error {

	if err := service.collection(session).Load(notDeleted(criteria), objectLink); err != nil {
		return derp.Wrap(err, "service.ObjectLink.Load", "Unable to load ObjectLink", criteria)
	}

	return nil
}

// Save adds/updates an ObjectLink in the database
func (service *Context) Save(session data.Session, objectLink *model.ObjectLink, note string) error {

	const location = "service.ObjectLink.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(objectLink); err != nil {
		return derp.Wrap(err, location, "Unable to validate ObjectLink", objectLink)
	}

	// Save the value to the database
	if err := service.collection(session).Save(objectLink, note); err != nil {
		return derp.Wrap(err, location, "Unable to save ObjectLink", objectLink, note)
	}

	return nil
}

// Delete removes an ObjectLink from the database (hard delete)
func (service *Context) Delete(session data.Session, objectLink *model.ObjectLink, note string) error {

	const location = "service.ObjectLink.Delete"

	// Delete this ObjectLink from the database
	if err := service.collection(session).HardDelete(exp.Equal("_id", objectLink.ObjectLinkID)); err != nil {
		return derp.Wrap(err, location, "Unable to delete ObjectLink", objectLink)
	}

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly documents that match the provided criteria
func (service *Context) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Context record, without applying any additional business rules
func (service *Context) HardDeleteByID(session data.Session, userID primitive.ObjectID, contextItemID primitive.ObjectID) error {

	const location = "service.ObjectLink.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", contextItemID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete ObjectLink", "userID: "+userID.Hex(), "contextItemID: "+contextItemID.Hex())
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Context) ObjectType() string {
	return "ObjectLink"
}

// New returns a fully initialized model.ObjectLink as a data.Object.
func (service *Context) ObjectNew() data.Object {
	result := model.NewObjectLink()
	return &result
}

func (service *Context) ObjectID(object data.Object) primitive.ObjectID {

	if objectLink, ok := object.(*model.ObjectLink); ok {
		return objectLink.ObjectLinkID
	}

	return primitive.NilObjectID
}

func (service *Context) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Context) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewObjectLink()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Context) ObjectSave(session data.Session, object data.Object, note string) error {

	if objectLink, ok := object.(*model.ObjectLink); ok {
		return service.Save(session, objectLink, note)
	}
	return derp.Internal("service.ObjectLink.ObjectSave", "Invalid object type", object)
}

func (service *Context) ObjectDelete(session data.Session, object data.Object, note string) error {
	if objectLink, ok := object.(*model.ObjectLink); ok {
		return service.Delete(session, objectLink, note)
	}
	return derp.Internal("service.ObjectLink.ObjectDelete", "Invalid object type", object)
}

func (service *Context) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.ObjectLink", "Not Authorized")
}

func (service *Context) Schema() schema.Schema {
	return schema.New(model.ObjectLinkSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Context) CountByContext(session data.Session, context string, criteria exp.Expression) (int64, error) {
	criteria = criteria.And(exp.Equal("context", context))
	return service.Count(session, criteria)
}

func (service *Context) RangeByContext(session data.Session, context string, criteria exp.Expression, options ...option.Option) (iter.Seq[model.ObjectLink], error) {
	criteria = criteria.And(exp.Equal("context", context))
	return service.Range(session, criteria, options...)
}

func (service *Context) QueryByInReplyTo(session data.Session, inReplyTo string, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.ObjectLink], error) {
	criteria = criteria.AndEqual("inReplyTo", inReplyTo)
	spew.Dump(criteria)
	return service.Query(session, criteria, options...)
}

func (service *Context) LoadByID(session data.Session, context string, object string, objectLink *model.ObjectLink) error {
	criteria := exp.Equal("context", context).AndEqual("object", object)
	return service.Load(session, criteria, objectLink)
}

func (service *Context) HardDeleteByObject(session data.Session, context string, object string) error {
	criteria := exp.Equal("context", context).AndEqual("object", object)

	// Delete this ObjectLink from the database
	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.ObjectLink.DeleteByObject", "Unable to delete ObjectLink", "context: "+context, "object: "+object)
	}
	return nil
}

/******************************************
 * Custom Behaviors
 ******************************************/

// SaveUnique guarantees that there is only one ObjectLink for a given context/object pair.  It does this
// by removing any existing ObjectLink that matches this pair before saving the new one.
func (service *Context) SaveUnique(session data.Session, objectLink *model.ObjectLink, note string) error {

	const location = "service.ObjectLink.SaveUnique"

	if err := service.HardDeleteByObject(session, objectLink.Context, objectLink.Object); err != nil {
		return derp.Wrap(err, location, "Unable to delete existing ObjectLink", "context: "+objectLink.Context, "object: "+objectLink.Object)
	}

	if err := service.Save(session, objectLink, note); err != nil {
		return derp.Wrap(err, location, "Unable to save ObjectLink", objectLink, note)
	}

	return nil
}

/******************************************
 * Collection Interface
 ******************************************/

func (service *Context) CollectionCount(session data.Session, context string, criteria exp.Expression) collection.CounterFunc {
	return func() (int64, error) {
		return service.CountByContext(session, context, criteria)
	}
}

// CollectionIterator returns the iterator function for this collection
func (service *Context) CollectionIterator(session data.Session, context string, criteria exp.Expression) collection.IteratorFunc {

	const location = "service.ObjectLink.CollectionIterator"

	return func(startAfter string) (iter.Seq[mapof.Any], error) {

		// Add the "startAfter" criteria (if applicable)
		if startAfter != "" {
			marker := model.NewObjectLink()
			if err := service.LoadByID(session, context, startAfter, &marker); err == nil {
				criteria = criteria.AndLessThan("_id", marker.ObjectLinkID)
			}
		}

		// Get Replies for this Context (sorted by insertion date)
		result, err := service.RangeByContext(
			session,
			context,
			criteria,
			option.Fields("_id"),
			option.SortDesc("_id"),
		)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unable to create iterator", "context", context)
		}

		// Map into a range of JSON-LD objects
		return ranges.Map(result, func(item model.ObjectLink) mapof.Any {
			return mapof.Any{
				vocab.PropertyID: item.ObjectLinkID.Hex(),
			}
		}), nil
	}
}
