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

// Inbox manages all Inbox records for a User.  This includes Inbox and Outbox
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

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Inbox) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Inbox) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// New creates a newly initialized Inbox that is ready to use
func (service *Inbox) New() model.Message {
	return model.NewMessage()
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *Inbox) Query(criteria exp.Expression, options ...option.Option) ([]model.Message, error) {
	result := []model.Message{}
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Activities that match the provided criteria
func (service *Inbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Inbox from the database
func (service *Inbox) Load(criteria exp.Expression, result *model.Message) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error loading Inbox", criteria)
	}

	return nil
}

// Save adds/updates an Inbox in the database
func (service *Inbox) Save(message *model.Message, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(message); err != nil {
		return derp.Wrap(err, "service.Inbox.Save", "Error cleaning Inbox", message)
	}

	// TODO: In what circumstances should this trigger additional events?
	if message.Document.InternalID.IsZero() {
		switch message.Document.Type {
		case model.DocumentTypeArticle:
		case model.DocumentTypeNote:
		case model.DocumentTypeBlock:
		case model.DocumentTypeFollow:
		case model.DocumentTypeLike:
		}
	}

	// Save the value to the database
	if err := service.collection.Save(message, note); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error saving Inbox", message, note)
	}

	return nil
}

// Delete removes an Inbox from the database (virtual delete)
func (service *Inbox) Delete(message *model.Message, note string) error {

	// Delete Inbox record last.
	if err := service.collection.Delete(message, note); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error deleting Inbox", message, note)
	}

	return nil
}

// DeleteMany removes all child streams from the provided stream (virtual delete)
func (service *Inbox) DeleteMany(criteria exp.Expression, note string) error {

	it, err := service.List(criteria)

	if err != nil {
		return derp.Wrap(err, "service.Message.Delete", "Error listing streams to delete", criteria)
	}

	message := model.NewMessage()

	for it.Next(&message) {
		if err := service.Delete(&message, note); err != nil {
			return derp.Wrap(err, "service.Message.Delete", "Error deleting message", message)
		}
		message = model.NewMessage()
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Inbox) ObjectType() string {
	return "Inbox"
}

// New returns a fully initialized model.Stream as a data.Object.
func (service *Inbox) ObjectNew() data.Object {
	result := model.NewMessage()
	return &result
}

func (service *Inbox) ObjectID(object data.Object) primitive.ObjectID {

	if message, ok := object.(*model.Message); ok {
		return message.MessageID
	}

	return primitive.NilObjectID
}

func (service *Inbox) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Inbox) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Inbox) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewMessage()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Inbox) ObjectSave(object data.Object, note string) error {
	if message, ok := object.(*model.Message); ok {
		return service.Save(message, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectSave", "Invalid Object Type", object)
}

func (service *Inbox) ObjectDelete(object data.Object, note string) error {
	if message, ok := object.(*model.Message); ok {
		return service.Delete(message, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectDelete", "Invalid Object Type", object)
}

func (service *Inbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Inbox", "Not Authorized")
}

func (service *Inbox) Schema() schema.Schema {
	return schema.New(model.MessageSchema())
}

/******************************************
 * Custom Query Methods
 ******************************************/

func (service *Inbox) QueryByUserID(userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Message, error) {
	criteria = exp.Equal("userId", userID).And(criteria)
	return service.Query(criteria, options...)
}

func (service *Inbox) ListByFolder(userID primitive.ObjectID, folderID primitive.ObjectID) (data.Iterator, error) {
	criteria := exp.Equal("userId", userID).
		AndEqual("folderId", folderID)

	return service.List(criteria)
}

func (service *Inbox) ListByFollowingID(userID primitive.ObjectID, followingID primitive.ObjectID) (data.Iterator, error) {
	criteria := exp.Equal("userId", userID).
		AndEqual("origin.internalId", followingID)

	return service.List(criteria)
}

func (service *Inbox) LoadByID(userID primitive.ObjectID, messageID primitive.ObjectID, result *model.Message) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("_id", messageID)

	return service.Load(criteria, result)
}

func (service *Inbox) LoadByURL(userID primitive.ObjectID, url string, result *model.Message) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("document.url", url)

	return service.Load(criteria, result)
}

/******************************************
 * Custom Behaviors
 ******************************************/

func (service *Inbox) UpdateInboxFolders(userID primitive.ObjectID, followingID primitive.ObjectID, folderID primitive.ObjectID) {

	it, err := service.ListByFollowingID(userID, followingID)

	if err != nil {
		derp.Report(derp.Wrap(err, "service.Inbox", "Cannot list Activities by following", userID, followingID))
		return
	}

	message := model.NewMessage()
	for it.Next(&message) {
		message.FolderID = folderID
		if err := service.Save(&message, "UpdateInboxFolders"); err != nil {
			derp.Report(derp.Wrap(err, "service.Inbox", "Cannot save Inbox Message", message))
		}
		message = model.NewMessage()
	}
}

func (service *Inbox) DeleteByOrigin(internalID primitive.ObjectID, note string) error {
	return service.DeleteMany(exp.Equal("origin.internalId", internalID), note)
}

func (service *Inbox) DeleteByFolder(userID primitive.ObjectID, folderID primitive.ObjectID) error {

	it, err := service.ListByFolder(userID, folderID)

	if err != nil {
		return derp.Wrap(err, "service.Inbox", "Cannot list Activities by folder", userID, folderID)
	}

	message := model.NewMessage()
	for it.Next(&message) {
		if err := service.Delete(&message, "DeleteByFolder"); err != nil {
			return derp.Wrap(err, "service.Inbox", "Cannot delete Inbox", message)
		}
		message = model.NewMessage()
	}

	return nil
}

// SetReadDate updates the readDate for a single Inbox IF it is not already read
func (service *Inbox) SetReadDate(userID primitive.ObjectID, token string, readDate int64) error {

	const location = "service.Inbox.SetReadDate"

	// Convert the string to an ObjectID
	messageID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Cannot parse messageID", token)
	}

	// Try to load the Inbox from the database
	message := model.NewMessage()
	if err := service.LoadByID(userID, messageID, &message); err != nil {
		return derp.Wrap(err, location, "Cannot load Inbox", userID, token)
	}

	// RULE: If the Inbox is already marked as read, then we don't need to update it.  Return success.
	if message.ReadDate > 0 {
		return nil
	}

	// Update the readDate and save the Inbox
	message.ReadDate = readDate

	if err := service.Save(&message, "Mark Read"); err != nil {
		return derp.Wrap(err, location, "Cannot save Inbox", message)
	}

	// Actual success here.
	return nil
}

// QueryPurgeable returns a list of Inboxs that are older than the purge date for this following
func (service *Inbox) QueryPurgeable(following *model.Following) ([]model.Message, error) {

	// Purge date is X days before the current date
	purgeDuration := time.Duration(following.PurgeDuration) * 24 * time.Hour
	purgeDate := time.Now().Add(0 - purgeDuration).Unix()

	// Activities in the INBOX can be purged if they are READ and older than the purge date
	criteria := exp.GreaterThan("readDate", 0).
		AndLessThan("readDate", purgeDate)

	return service.Query(criteria)
}
