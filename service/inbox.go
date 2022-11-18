package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/microcosm-cc/bluemonday"
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
func (service *Inbox) New() model.InboxItem {
	return model.NewInboxItem()
}

// Query returns a slice of InboxItems that math the provided criteria
func (service *Inbox) Query(criteria exp.Expression, options ...option.Option) ([]model.InboxItem, error) {
	result := []model.InboxItem{}
	err := service.collection.Query(&result, criteria, options...)
	return result, err
}

// List returns an iterator containing all of the Inboxs that match the provided criteria
func (service *Inbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Inbox from the database
func (service *Inbox) Load(criteria exp.Expression, result *model.InboxItem) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Inbox.Load", "Error loading Inbox", criteria)
	}

	return nil
}

// Save adds/updates an Inbox in the database
func (service *Inbox) Save(inboxItem *model.InboxItem, note string) error {

	// Sanitize the input
	strictPolicy := bluemonday.StrictPolicy()
	inboxItem.Label = strictPolicy.Sanitize(inboxItem.Label)
	inboxItem.Summary = strictPolicy.Sanitize(inboxItem.Summary)
	inboxItem.Content = bluemonday.UGCPolicy().Sanitize(inboxItem.Content)

	if err := service.collection.Save(inboxItem, note); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error saving Inbox", inboxItem, note)
	}

	return nil
}

// Delete removes an Inbox from the database (virtual delete)
func (service *Inbox) Delete(inboxItem *model.InboxItem, note string) error {

	// Delete Inbox record last.
	if err := service.collection.Delete(inboxItem, note); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error deleting Inbox", inboxItem, note)
	}

	return nil
}

/*******************************************
 * Generic Data Methods
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *Inbox) ObjectNew() data.Object {
	result := model.NewInboxItem()
	return &result
}

func (service *Inbox) ObjectID(object data.Object) primitive.ObjectID {

	if inboxItem, ok := object.(*model.InboxItem); ok {
		return inboxItem.InboxItemID
	}

	return primitive.NilObjectID
}
func (service *Inbox) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Inbox) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewInboxItem()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Inbox) ObjectSave(object data.Object, note string) error {
	if inboxItem, ok := object.(*model.InboxItem); ok {
		return service.Save(inboxItem, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectSave", "Invalid Object Type", object)
}

func (service *Inbox) ObjectDelete(object data.Object, note string) error {
	if inboxItem, ok := object.(*model.InboxItem); ok {
		return service.Delete(inboxItem, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectDelete", "Invalid Object Type", object)
}

func (service *Inbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Inbox", "Not Authorized")
}

func (service *Inbox) Schema() schema.Element {
	return model.InboxItemSchema()
}

/*******************************************
 * Custom Queries
 *******************************************/

func (service *Inbox) LoadItemByID(userID primitive.ObjectID, inboxItemID primitive.ObjectID, result *model.InboxItem) error {

	const location = "service.Inbox.LoadItemByID"

	criteria := exp.
		Equal("userId", userID).
		AndEqual("_id", inboxItemID)

	return service.Load(criteria, result)
}

// LoadBySource locates a single stream that matches the provided OriginURL
func (service *Inbox) LoadByOriginURL(userID primitive.ObjectID, originURL string, result *model.InboxItem) error {

	criteria := exp.
		Equal("userId", userID).
		AndEqual("origin.url", originURL)

	return service.Load(criteria, result)
}

// SetReadDate updates the readDate for a single InboxItem IF it is not already read
func (service *Inbox) SetReadDate(userID primitive.ObjectID, token string, readDate int64) error {

	const location = "service.Inbox.SetReadDate"

	// Try to load the InboxItem from the database
	inboxItem := model.NewInboxItem()

	// Convert the string to an ObjectID
	inboxItemID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Cannot parse inboxItemID", token)
	}

	if err := service.LoadItemByID(userID, inboxItemID, &inboxItem); err != nil {
		return derp.Wrap(err, location, "Cannot load InboxItem", userID, token)
	}

	// RULE: If the InboxItem is already marked as read, then we don't need to update it.  Return success.
	if inboxItem.ReadDate > 0 {
		return nil
	}

	// Update the readDate and save the InboxItem
	inboxItem.ReadDate = readDate

	if err := service.Save(&inboxItem, "Mark Read"); err != nil {
		return derp.Wrap(err, location, "Cannot save InboxItem", inboxItem)
	}

	// Actual success here.
	return nil
}

// QueryPurgeable returns a list of InboxItems that are older than the purge date for this subscription
func (service *Inbox) QueryPurgeable(subscription *model.Subscription) ([]model.InboxItem, error) {

	// Purge date is X days before the current date
	purgeDuration := time.Duration(subscription.PurgeDuration) * 24 * time.Hour
	purgeDate := time.Now().Add(0 - purgeDuration).Unix()

	// InboxItems can be purged if they are READ and older than the purge date
	criteria := exp.GreaterThan("readDate", 0).AndLessThan("readDate", purgeDate)
	return service.Query(criteria)
}
