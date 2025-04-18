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

// Subscriber defines a service that manages all content subscribers created and imported by Users.
type Subscriber struct {
	collection data.Collection
}

// NewSubscriber returns a fully initialized Subscriber service
func NewSubscriber() Subscriber {
	return Subscriber{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Subscriber) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Subscriber) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Subscriber) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe Subscribers that match the provided criteria
func (service *Subscriber) Query(criteria exp.Expression, options ...option.Option) ([]model.Subscriber, error) {
	result := make([]model.Subscriber, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Subscribers that match the provided criteria
func (service *Subscriber) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Subscriber records that match the provided criteria
func (service *Subscriber) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Subscriber], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Subscriber.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewSubscriber), nil
}

// Load retrieves an Subscriber from the database
func (service *Subscriber) Load(criteria exp.Expression, subscriber *model.Subscriber) error {

	if err := service.collection.Load(notDeleted(criteria), subscriber); err != nil {
		return derp.Wrap(err, "service.Subscriber.Load", "Error loading Subscriber", criteria)
	}

	return nil
}

// Save adds/updates an Subscriber in the database
func (service *Subscriber) Save(subscriber *model.Subscriber, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(subscriber); err != nil {
		return derp.Wrap(err, "service.Subscriber.Save", "Error validating Subscriber", subscriber)
	}

	// Save the subscriber to the database
	if err := service.collection.Save(subscriber, note); err != nil {
		return derp.Wrap(err, "service.Subscriber.Save", "Error saving Subscriber", subscriber, note)
	}

	return nil
}

// Delete removes an Subscriber from the database (virtual delete)
func (service *Subscriber) Delete(subscriber *model.Subscriber, note string) error {

	// Delete this Subscriber
	if err := service.collection.Delete(subscriber, note); err != nil {
		return derp.Wrap(err, "service.Subscriber.Delete", "Error deleting Subscriber", subscriber, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Subscriber) ObjectType() string {
	return "Subscriber"
}

// New returns a fully initialized model.Subscriber as a data.Object.
func (service *Subscriber) ObjectNew() data.Object {
	result := model.NewSubscriber()
	return &result
}

func (service *Subscriber) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Subscriber); ok {
		return mention.SubscriberID
	}

	return primitive.NilObjectID
}

func (service *Subscriber) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Subscriber) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewSubscriber()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Subscriber) ObjectSave(object data.Object, comment string) error {
	if subscriber, ok := object.(*model.Subscriber); ok {
		return service.Save(subscriber, comment)
	}
	return derp.NewInternalError("service.Subscriber.ObjectSave", "Invalid Object Type", object)
}

func (service *Subscriber) ObjectDelete(object data.Object, comment string) error {
	if subscriber, ok := object.(*model.Subscriber); ok {
		return service.Delete(subscriber, comment)
	}
	return derp.NewInternalError("service.Subscriber.ObjectDelete", "Invalid Object Type", object)
}

func (service *Subscriber) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Subscriber.ObjectUserCan", "Not Authorized")
}

func (service *Subscriber) Schema() schema.Schema {
	return schema.New(model.SubscriberSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Subscriber) LoadByID(userID primitive.ObjectID, subscriberID primitive.ObjectID, subscriber *model.Subscriber) error {

	criteria := exp.Equal("_id", subscriberID).
		AndEqual("userId", userID)

	return service.Load(criteria, subscriber)
}

func (service *Subscriber) LoadByToken(userID primitive.ObjectID, token string, subscriber *model.Subscriber) error {

	if subscriberID, err := primitive.ObjectIDFromHex(token); err == nil {
		return service.LoadByID(userID, subscriberID, subscriber)
	} else {
		return derp.Wrap(err, "service.Subscriber.LoadByToken", "Invalid Token", token)
	}
}
