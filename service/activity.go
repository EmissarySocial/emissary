package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Activity manages all interactions with a user's Activity
type Activity struct {
	collection data.Collection
}

// NewActivity returns a fully populated Activity service
func NewActivity(collection data.Collection) Activity {
	service := Activity{
		collection: collection,
	}

	service.Refresh(collection)
	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Activity) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Activity) Close() {

}

/*******************************************
 * Common Data Methods
 *******************************************/

// New creates a newly initialized Activity that is ready to use
func (service *Activity) New() model.Activity {
	return model.NewActivity()
}

// Query returns a slice of Activitys that math the provided criteria
func (service *Activity) Query(criteria exp.Expression, options ...option.Option) ([]model.Activity, error) {
	result := []model.Activity{}
	err := service.collection.Query(&result, criteria, options...)
	return result, err
}

// List returns an iterator containing all of the Activitys that match the provided criteria
func (service *Activity) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Activity from the database
func (service *Activity) Load(criteria exp.Expression, result *model.Activity) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Activity.Load", "Error loading Activity", criteria)
	}

	return nil
}

// Save adds/updates an Activity in the database
func (service *Activity) Save(activity *model.Activity, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(activity); err != nil {
		return derp.Wrap(err, "service.Activity.Save", "Error cleaning Activity", activity)
	}

	// Save to the database
	if err := service.collection.Save(activity, note); err != nil {
		return derp.Wrap(err, "service.Activity", "Error saving Activity", activity, note)
	}

	return nil
}

// Delete removes an Activity from the database (virtual delete)
func (service *Activity) Delete(activity *model.Activity, note string) error {

	// Delete Activity record last.
	if err := service.collection.Delete(activity, note); err != nil {
		return derp.Wrap(err, "service.Activity", "Error deleting Activity", activity, note)
	}

	return nil
}

/*******************************************
 * Generic Data Methods
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *Activity) ObjectNew() data.Object {
	result := model.NewActivity()
	return &result
}

func (service *Activity) ObjectID(object data.Object) primitive.ObjectID {

	if activity, ok := object.(*model.Activity); ok {
		return activity.ActivityID
	}

	return primitive.NilObjectID
}
func (service *Activity) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Activity) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewActivity()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Activity) ObjectSave(object data.Object, note string) error {
	if activity, ok := object.(*model.Activity); ok {
		return service.Save(activity, note)
	}
	return derp.NewInternalError("service.Activity.ObjectSave", "Invalid Object Type", object)
}

func (service *Activity) ObjectDelete(object data.Object, note string) error {
	if activity, ok := object.(*model.Activity); ok {
		return service.Delete(activity, note)
	}
	return derp.NewInternalError("service.Activity.ObjectDelete", "Invalid Object Type", object)
}

func (service *Activity) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Activity", "Not Authorized")
}

func (service *Activity) Schema() schema.Schema {
	return schema.New(model.ActivitySchema())
}

/*******************************************
 * Custom Queries
 *******************************************/

func (service *Activity) LoadItemByID(userID primitive.ObjectID, activityID primitive.ObjectID, result *model.Activity) error {

	criteria := exp.
		Equal("userId", userID).
		AndEqual("_id", activityID)

	return service.Load(criteria, result)
}

// LoadBySource locates a single stream that matches the provided OriginURL
func (service *Activity) LoadByOriginURL(userID primitive.ObjectID, originURL string, result *model.Activity) error {

	criteria := exp.
		Equal("userId", userID).
		AndEqual("origin.url", originURL)

	return service.Load(criteria, result)
}

// SetReadDate updates the readDate for a single Activity IF it is not already read
func (service *Activity) SetReadDate(userID primitive.ObjectID, token string, readDate int64) error {

	const location = "service.Activity.SetReadDate"

	// Try to load the Activity from the database
	activity := model.NewActivity()

	// Convert the string to an ObjectID
	activityID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Cannot parse activityID", token)
	}

	if err := service.LoadItemByID(userID, activityID, &activity); err != nil {
		return derp.Wrap(err, location, "Cannot load Activity", userID, token)
	}

	// RULE: If the Activity is already marked as read, then we don't need to update it.  Return success.
	if activity.ReadDate > 0 {
		return nil
	}

	// Update the readDate and save the Activity
	activity.ReadDate = readDate

	if err := service.Save(&activity, "Mark Read"); err != nil {
		return derp.Wrap(err, location, "Cannot save Activity", activity)
	}

	// Actual success here.
	return nil
}

// QueryPurgeable returns a list of Activitys that are older than the purge date for this subscription
func (service *Activity) QueryPurgeable(subscription *model.Subscription) ([]model.Activity, error) {

	// Purge date is X days before the current date
	purgeDuration := time.Duration(subscription.PurgeDuration) * 24 * time.Hour
	purgeDate := time.Now().Add(0 - purgeDuration).Unix()

	// Activitys can be purged if they are READ and older than the purge date
	criteria := exp.GreaterThan("readDate", 0).AndLessThan("readDate", purgeDate)
	return service.Query(criteria)
}
