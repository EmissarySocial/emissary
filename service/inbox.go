package service

import (
	"iter"
	"math"
	"sync"
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
	collection    data.Collection
	ruleService   *Rule
	folderService *Folder
	host          string
	counter       int
	mutex         *sync.Mutex
}

// NewInbox returns a fully populated Inbox service
func NewInbox() Inbox {
	return Inbox{
		mutex: &sync.Mutex{},
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Inbox) Refresh(collection data.Collection, ruleService *Rule, folderService *Folder, host string) {
	service.collection = collection
	service.ruleService = ruleService
	service.folderService = folderService
	service.host = host
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

// Count returns the number of records that match the provided criteria
func (service *Inbox) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *Inbox) Query(criteria exp.Expression, options ...option.Option) ([]model.Message, error) {
	result := []model.Message{}
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Activities that match the provided criteria
func (service *Inbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Messages that match the provided criteria
func (service *Inbox) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Message], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Inbox.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewMessage), nil
}

// Load retrieves an Inbox from the database
func (service *Inbox) Load(criteria exp.Expression, result *model.Message) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Inbox.Load", "Error loading Inbox message", criteria)
	}

	return nil
}

// Save adds/updates an Inbox in the database
func (service *Inbox) Save(message *model.Message, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(message); err != nil {
		return derp.Wrap(err, "service.Inbox.Save", "Error validating Inbox", message)
	}

	// Calculate a (hopefully unique) rank for this message
	service.CalculateRank(message)

	// Save the value to the database
	if err := service.collection.Save(message, note); err != nil {
		return derp.Wrap(err, "service.Inbox.Save", "Error saving Inbox", message, note)
	}

	// Recalculate the unread count for the folder that owns this message.
	if err := service.folderService.CalculateUnreadCount(message.UserID, message.FolderID); err != nil {
		return derp.Wrap(err, "service.Inbox.Save", "Error recalculating unread count", message)
	}

	// Wait 1 millisecond between each document to guarantee sorting by CreateDate
	time.Sleep(1 * time.Millisecond)

	return nil
}

// Delete removes an Inbox from the database (virtual delete)
func (service *Inbox) Delete(message *model.Message, note string) error {

	// Delete Inbox record last.
	if err := service.collection.Delete(message, note); err != nil {
		return derp.Wrap(err, "service.Inbox.Delete", "Error deleting Inbox", message, note)
	}

	return nil
}

// DeleteMany removes all child streams from the provided stream (virtual delete)
func (service *Inbox) DeleteMany(criteria exp.Expression, note string) error {

	rangeFunc, err := service.Range(criteria)

	if err != nil {
		return derp.Wrap(err, "service.Inbox.DeleteMany", "Error listing streams to delete", criteria)
	}

	for message := range rangeFunc {
		if err := service.Delete(&message, note); err != nil {
			return derp.Wrap(err, "service.Inbox.DeleteMany", "Error deleting message", message)
		}
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
	return derp.NewUnauthorizedError("service.Inbox.ObjectUserCan", "Not Authorized")
}

func (service *Inbox) Schema() schema.Schema {
	return schema.New(model.MessageSchema())
}

/******************************************
 * Custom Query Methods
 ******************************************/

func (service *Inbox) QueryByUserID(userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Message, error) {
	criteria = criteria.AndEqual("userId", userID)
	return service.Query(criteria, options...)
}

func (service *Inbox) RangeByFolder(userID primitive.ObjectID, folderID primitive.ObjectID) (iter.Seq[model.Message], error) {
	criteria := exp.Equal("userId", userID).
		AndEqual("folderId", folderID)

	return service.Range(criteria)
}

func (service *Inbox) RangeByFollowingID(userID primitive.ObjectID, followingID primitive.ObjectID) (iter.Seq[model.Message], error) {
	criteria := exp.Equal("userId", userID).
		AndEqual("origin.followingId", followingID)

	return service.Range(criteria)
}

func (service *Inbox) RangeByUserID(userID primitive.ObjectID) (iter.Seq[model.Message], error) {
	return service.Range(exp.Equal("userId", userID))
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
		return derp.Wrap(err, "service.Inbox.LoadByRank", "Error loading Inbox", userID, folderID, rankExpression)
	}

	for it.Next(result) {
		return nil
	}

	return derp.NewNotFoundError("service.Inbox.LoadByRank", "Inbox message not found", userID, folderID, rankExpression)
}

// LoadByURL returns the first message that matches the provided UserID and URL
func (service *Inbox) LoadByURL(userID primitive.ObjectID, url string, result *model.Message) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("url", url)

	return service.Load(criteria, result)
}

// LoadUnreadByURL returns the first UNREAD message that matches the provided UserID and URL
func (service *Inbox) LoadUnreadByURL(userID primitive.ObjectID, url string, result *model.Message) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("url", url).
		AndEqual("readDate", math.MaxInt64)

	return service.Load(criteria, result)
}

// LoadSibling searches for the previous/next sibling to the provided message criteria.
func (service *Inbox) LoadSibling(folderID primitive.ObjectID, rank int64, following string, direction string) (model.Message, error) {

	const location = "service.Inbox.LoadSibling"

	// Initialize query parameters
	var criteria exp.Expression = exp.Equal("folderId", folderID)
	var sort option.Option

	// Specific criteria for previous/next
	if direction == "prev" {
		criteria = criteria.AndLessThan("rank", rank)
		sort = option.SortDesc("rank")
	} else {
		criteria = criteria.AndGreaterThan("rank", rank)
		sort = option.SortAsc("rank")
	}

	// Limit further if a "followingId" is present
	if followingID, err := primitive.ObjectIDFromHex(following); err == nil {
		criteria = criteria.AndEqual("origin.followingId", followingID)
	}

	// Query the database.
	it, err := service.List(criteria, option.FirstRow(), sort)

	if err != nil {
		return model.Message{}, derp.Wrap(err, location, "Error retrieving siblings")
	}

	// Try to read the results
	result := model.NewMessage()

	// This *should* read the prev/next message into the pointer and be done.
	if it.Next(&result) {
		return result, nil
	}

	// No results.  Shame! Shame!
	return model.Message{}, derp.NewNotFoundError(location, "No record found")
}

func (service *Inbox) LoadOldestUnread(userID primitive.ObjectID, message *model.Message) error {

	const location = "service.Inbox.LoadOldestUnread"

	criteria := exp.Equal("userId", userID)
	sort := option.SortAsc("createDate")

	it, err := service.List(criteria, option.FirstRow(), sort)

	if err != nil {
		return derp.Wrap(err, location, "Error listing messages")
	}

	for it.Next(message) {
		return nil
	}

	return derp.NewNotFoundError(location, "No unread messages")
}

func (service *Inbox) MarkReadByDate(userID primitive.ObjectID, rank int64) error {

	const location = "service.Inbox.MarkReadByDate"

	criteria := exp.Equal("userId", userID).AndLessThan("rank", rank)
	sort := option.SortAsc("rank")

	it, err := service.List(criteria, sort)

	if err != nil {
		return derp.Wrap(err, location, "Error listing messages")
	}

	message := model.NewMessage()
	for it.Next(&message) {
		if err := service.MarkRead(&message); err != nil {
			return derp.Wrap(err, location, "Error marking message as read")
		}
	}

	return nil
}

/******************************************
 * Custom Behaviors
 ******************************************/

// MarkRead updates a message to "READ" status and recalculates statistics
func (service *Inbox) MarkRead(message *model.Message) error {

	const location = "service.Inbox.MarkRead"

	// Set status to READ.  If the message was not changed, then exit
	if isUpdated := message.MarkRead(); !isUpdated {
		return nil
	}

	// Save the message
	if err := service.Save(message, "Update StateID to "+message.StateID); err != nil {
		return derp.Wrap(err, location, "Error saving message")
	}

	// Recalculate statistics
	if err := service.recalculateUnreadCounts(message); err != nil {
		return derp.Wrap(err, location, "Error recalculating unread counts")
	}

	// Lo hicimos!
	return nil
}

// MarkRead updates a message to "UNREAD" status and recalculates statistics
func (service *Inbox) MarkUnread(message *model.Message) error {

	const location = "service.Inbox.MarkUnread"

	// Set status to UNREAD.  If the message was not changed, then exit
	if isUpdated := message.MarkUnread(); !isUpdated {
		return nil
	}

	// Save the message
	if err := service.Save(message, "Update StateID to "+message.StateID); err != nil {
		return derp.Wrap(err, location, "Error saving message")
	}

	// Recalculate statistics
	if err := service.recalculateUnreadCounts(message); err != nil {
		return derp.Wrap(err, location, "Error recalculating unread counts")
	}

	// Success
	return nil
}

func (service *Inbox) MarkMuted(message *model.Message) error {

	const location = "service.Inbox.MarkMuted"

	// Set status to MUTED.  If the message is unchanged, then exit
	if isUpdated := message.MarkMuted(); !isUpdated {
		return nil
	}

	// Save the message
	if err := service.Save(message, "Set Status to MUTED"); err != nil {
		return derp.Wrap(err, location, "Error saving message")
	}

	return nil
}

func (service *Inbox) MarkUnmuted(message *model.Message) error {

	const location = "service.Inbox.MarkMuted"

	// Set status to READ (unmuted).  If the message is unchanged, then exit
	if isUpdated := message.MarkRead(); !isUpdated {
		return nil
	}

	// Save the message
	if err := service.Save(message, "Set Status to MUTED"); err != nil {
		return derp.Wrap(err, location, "Error saving message")
	}

	// Success
	return nil
}

func (service *Inbox) recalculateUnreadCounts(message *model.Message) error {

	const location = "service.Inbox.recalculateUnreadCounts"

	// Recalculate the "unread" count on the corresponding folder
	unreadCount, err := service.CountUnreadMessages(message.UserID, message.FolderID)

	if err != nil {
		return derp.Wrap(err, location, "Error counting unread messages")
	}

	// Update the "unread" count for the Folder
	if err := service.folderService.SetUnreadCount(message.UserID, message.FolderID, unreadCount); err != nil {
		return derp.Wrap(err, location, "Error setting unread count")
	}

	// Lo hicimos! we did it.
	return nil
}

// CalculateRank generates a unique rank for the message based on the PublishDate and the number of messages
// that already exist in the database with this PublishDate.
func (service *Inbox) CalculateRank(message *model.Message) {

	// RULE: Do not reset the rank for items that already have one. This prevents
	// older messages from being re-queued to the top of the list
	if message.Rank > 0 {
		return
	}

	// Super-Quick™️ lock to use the service.counter variable
	service.mutex.Lock()
	defer service.mutex.Unlock()

	// Increment the counter (MOD 1000) so that we have precise ordering of messages
	service.counter = (service.counter + 1) % 1000

	// message.Rank = (time.Now().Unix() * 1000) + int64(service.counter)
	message.Rank = (message.PublishDate * 1000) + int64(service.counter)
}

// CountUnreadMessages counts the number of messages for a user/folder that are marked "unread".
func (service *Inbox) CountUnreadMessages(userID primitive.ObjectID, folderID primitive.ObjectID) (int, error) {

	criteria := exp.Equal("userId", userID).
		AndEqual("folderId", folderID).
		AndEqual("readDate", math.MaxInt64).
		AndEqual("deleteDate", 0)

	count, err := service.collection.Count(criteria)
	return int(count), err
}

func (service *Inbox) UpdateInboxFolders(userID primitive.ObjectID, followingID primitive.ObjectID, folderID primitive.ObjectID) {

	rangeFunc, err := service.RangeByFollowingID(userID, followingID)

	if err != nil {
		derp.Report(derp.Wrap(err, "service.Inbox", "Cannot list Activities by following", userID, followingID))
		return
	}

	for message := range rangeFunc {
		message.FolderID = folderID
		if err := service.Save(&message, "UpdateInboxFolders"); err != nil {
			derp.Report(derp.Wrap(err, "service.Inbox", "Cannot save Inbox Message", message))
		}
	}

	// Recalculate the "unread" count on the new folder
	if err := service.folderService.CalculateUnreadCount(userID, folderID); err != nil {
		derp.Report(derp.Wrap(err, "service.Inbox", "Cannot calculate unread count for new folder", userID, folderID))
	}
}

func (service *Inbox) DeleteByUserID(userID primitive.ObjectID, note string) error {
	return service.DeleteMany(exp.Equal("userId", userID), note)
}

func (service *Inbox) DeleteByOrigin(internalID primitive.ObjectID, note string) error {
	return service.DeleteMany(exp.Equal("origin.followingId", internalID), note)
}

func (service *Inbox) DeleteByFolder(userID primitive.ObjectID, folderID primitive.ObjectID) error {

	rangeFunc, err := service.RangeByFolder(userID, folderID)

	if err != nil {
		return derp.Wrap(err, "service.Inbox", "Cannot list Activities by folder", userID, folderID)
	}

	for message := range rangeFunc {
		if err := service.Delete(&message, "DeleteByFolder"); err != nil {
			return derp.Wrap(err, "service.Inbox", "Cannot delete Inbox", message)
		}
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
