package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Inbox manages all Inbox records for a User.  This includes Inbox and Outbox
type Inbox struct {
	collection   data.Collection
	blockService *Block
}

// NewInbox returns a fully populated Inbox service
func NewInbox() Inbox {
	return Inbox{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Inbox) Refresh(collection data.Collection, blockService *Block) {
	service.collection = collection
	service.blockService = blockService
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

	// Apply block filters to this message
	if err := service.blockService.FilterMessage(message); err != nil {
		return derp.Wrap(err, "service.Inbox.Save", "Error filtering Inbox", message)
	}

	// Calculate the rank for this message, using the number of messages with an identical PublishDate
	if err := service.CalculateRank(message); err != nil {
		return derp.Wrap(err, "service.Inbox.Save", "Error calculating rank", message)
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

// loadByID is a private function because it does not limit results by User.  It should ONLY
// be available to trusted services in this package.
func (service *Inbox) loadByID(messageID primitive.ObjectID, result *model.Message) error {
	criteria := exp.Equal("_id", messageID)

	return service.Load(criteria, result)
}

func (service *Inbox) LoadByID(userID primitive.ObjectID, messageID primitive.ObjectID, result *model.Message) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("_id", messageID)

	return service.Load(criteria, result)
}

func (service *Inbox) LoadByRank(userID primitive.ObjectID, folderID primitive.ObjectID, rankExpression exp.Expression, result *model.Message, options ...option.Option) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("folderId", folderID).
		And(rankExpression)

	it, err := service.List(criteria, options...)

	if err != nil {
		return derp.Wrap(err, "service.Inbox", "Error loading Inbox", userID, folderID, rankExpression)
	}

	for it.Next(result) {
		return nil
	}

	return derp.NewNotFoundError("service.Inbox", "Inbox message not found", userID, folderID, rankExpression)
}

func (service *Inbox) LoadByURL(userID primitive.ObjectID, url string, result *model.Message) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("document.url", url)

	return service.Load(criteria, result)
}

func (service *Inbox) LoadOrCreate(userID primitive.ObjectID, url string, result *model.Message) error {

	err := service.LoadByURL(userID, url, result)

	if err == nil {
		return nil
	}

	if derp.NotFound(err) {
		*result = model.NewMessage()
		result.UserID = userID
		result.URL = url
		return nil
	}

	return derp.Wrap(err, "service.Inbox", "Error loading Inbox", userID, url)
}

/******************************************
 * Custom Behaviors
 ******************************************/

func (service *Inbox) SetResponse(userID primitive.ObjectID, messageID primitive.ObjectID, responseType string) error {

	message := model.NewMessage()

	if err := service.LoadByID(userID, messageID, &message); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error loading Inbox", userID, messageID)
	}

	message.SetMyResponse(responseType)

	if err := service.Save(&message, "SetResponse"); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error saving Inbox", userID, messageID, responseType)
	}

	return nil
}

func (service *Inbox) CalculateRank(message *model.Message) error {

	count, err := queries.CountMessages(service.collection, message.UserID, message.FolderID, message.PublishDate)

	if err != nil {
		return derp.Wrap(err, "service.Inbox", "Error calculating rank", message)
	}

	message.Rank = (message.PublishDate * 1000) + int64(count)
	return nil
}

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

// QueryPurgeable returns a list of Inboxs that are older than the purge date for this following
// TODO: HIGH: ReadDate is gone.  Need another way to purge messages.
func (service *Inbox) QueryPurgeable(following *model.Following) ([]model.Message, error) {

	// Purge date is X days before the current date
	purgeDuration := time.Duration(following.PurgeDuration) * 24 * time.Hour
	purgeDate := time.Now().Add(0 - purgeDuration).Unix()

	// Activities in the INBOX can be purged if they are READ and older than the purge date
	criteria := exp.GreaterThan("readDate", 0).
		AndLessThan("readDate", purgeDate)

	return service.Query(criteria)
}
