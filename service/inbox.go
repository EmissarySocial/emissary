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

// Inbox manages all interactions with a user's Inbox
type Inbox struct {
	collection data.Collection
}

// NewInbox returns a fully populated Inbox service
func NewInbox(collection data.Collection) Inbox {
	service := Inbox{
		collection: collection,
	}

	service.Refresh(collection)
	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Inbox) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Inbox) Close() {

}

/*******************************************
 * Common Data Methods
 *******************************************/

// New creates a newly initialized Inbox that is ready to use
func (service *Inbox) New() model.Activity {
	return model.NewActivity()
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *Inbox) Query(criteria exp.Expression, options ...option.Option) ([]model.Activity, error) {
	result := []model.Activity{}
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Activities that match the provided criteria
func (service *Inbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Inbox from the database
func (service *Inbox) Load(criteria exp.Expression, result *model.Activity) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error loading Inbox", criteria)
	}

	return nil
}

// Save adds/updates an Inbox in the database
func (service *Inbox) Save(activity *model.Activity, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(activity); err != nil {
		return derp.Wrap(err, "service.Inbox.Save", "Error cleaning Inbox", activity)
	}

	// Save the value to the database
	if err := service.collection.Save(activity, note); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error saving Inbox", activity, note)
	}

	return nil
}

// Delete removes an Inbox from the database (virtual delete)
func (service *Inbox) Delete(activity *model.Activity, note string) error {

	// Delete Inbox record last.
	if err := service.collection.Delete(activity, note); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error deleting Inbox", activity, note)
	}

	return nil
}

/*******************************************
 * Generic Data Methods
 *******************************************/

// ObjectType returns the type of object that this service manages
func (service *Inbox) ObjectType() string {
	return "Activity"
}

// New returns a fully initialized model.Stream as a data.Object.
func (service *Inbox) ObjectNew() data.Object {
	result := model.NewActivity()
	return &result
}

func (service *Inbox) ObjectID(object data.Object) primitive.ObjectID {

	if activity, ok := object.(*model.Activity); ok {
		return activity.ActivityID
	}

	return primitive.NilObjectID
}

func (service *Inbox) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, criteria, options...)
}

func (service *Inbox) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Inbox) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewActivity()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Inbox) ObjectSave(object data.Object, note string) error {
	if activity, ok := object.(*model.Activity); ok {
		return service.Save(activity, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectSave", "Invalid Object Type", object)
}

func (service *Inbox) ObjectDelete(object data.Object, note string) error {
	if activity, ok := object.(*model.Activity); ok {
		return service.Delete(activity, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectDelete", "Invalid Object Type", object)
}

func (service *Inbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Inbox", "Not Authorized")
}

func (service *Inbox) Schema() schema.Schema {
	return schema.New(model.ActivitySchema())
}

/*******************************************
 * Custom Query Methods
 *******************************************/

func (service *Inbox) LoadItemByID(ownerID primitive.ObjectID, inboxItemID primitive.ObjectID, result *model.Activity) error {

	criteria := exp.
		Equal("_id", inboxItemID).
		AndEqual("ownerId", ownerID)

	return service.Load(criteria, result)
}

// LoadBySource locates a single stream that matches the provided OriginURL
func (service *Inbox) LoadByOriginURL(ownerID primitive.ObjectID, originURL string, result *model.Activity) error {

	criteria := exp.
		Equal("ownerId", ownerID).
		AndEqual("object.url", originURL)

	return service.Load(criteria, result)
}

// SetReadDate updates the readDate for a single Activity IF it is not already read
func (service *Inbox) SetReadDate(ownerID primitive.ObjectID, token string, readDate int64) error {

	const location = "service.Activity.SetReadDate"

	// Try to load the Activity from the database
	activity := model.NewActivity()

	// Convert the string to an ObjectID
	activityID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Cannot parse activityID", token)
	}

	if err := service.LoadItemByID(ownerID, activityID, &activity); err != nil {
		return derp.Wrap(err, location, "Cannot load Activity", ownerID, token)
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
func (service *Inbox) QueryPurgeable(subscription *model.Subscription) ([]model.Activity, error) {

	// Purge date is X days before the current date
	purgeDuration := time.Duration(subscription.PurgeDuration) * 24 * time.Hour
	purgeDate := time.Now().Add(0 - purgeDuration).Unix()

	// Activitys can be purged if they are READ and older than the purge date
	criteria := exp.GreaterThan("readDate", 0).AndLessThan("readDate", purgeDate)
	return service.Query(criteria)
}
