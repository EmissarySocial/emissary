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
func (service *Inbox) Refresh(ruleService *Rule, folderService *Folder, host string) {
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

func (service *Inbox) collection(session data.Session) data.Collection {
	return session.Collection("Inbox")
}

// New creates a newly initialized Inbox that is ready to use
func (service *Inbox) New() model.Message {
	return model.NewMessage()
}

// Count returns the number of records that match the provided criteria
func (service *Inbox) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *Inbox) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Message, error) {
	result := []model.Message{}
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Activities that match the provided criteria
func (service *Inbox) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Messages that match the provided criteria
func (service *Inbox) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Message], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Inbox.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewMessage), nil
}

// Load retrieves an Inbox from the database
func (service *Inbox) Load(session data.Session, criteria exp.Expression, result *model.Message) error {

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Inbox.Load", "Unable to load Inbox message", criteria)
	}

	return nil
}

// Save adds/updates an Inbox in the database
func (service *Inbox) Save(session data.Session, message *model.Message, note string) error {

	const location = "service.Inbox.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(message); err != nil {
		return derp.Wrap(err, location, "Unable to validate Inbox", message)
	}

	// Calculate a (hopefully unique) rank for this message
	service.CalculateRank(message)

	// Save the value to the database
	if err := service.collection(session).Save(message, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Inbox", message, note)
	}

	// Recalculate the unread count for the folder that owns this message.
	if err := service.folderService.CalculateUnreadCount(session, message.UserID, message.FolderID); err != nil {
		return derp.Wrap(err, location, "Unable to recalculate unread count", message)
	}

	// Wait 1 millisecond between each document to guarantee sorting by CreateDate
	time.Sleep(1 * time.Millisecond)

	return nil
}

// Delete removes an Inbox from the database (virtual delete)
func (service *Inbox) Delete(session data.Session, message *model.Message, note string) error {

	// Delete Inbox record last.
	if err := service.collection(session).Delete(message, note); err != nil {
		return derp.Wrap(err, "service.Inbox.Delete", "Unable to delete Inbox", message, note)
	}

	return nil
}

// DeleteMany removes all child streams from the provided stream (virtual delete)
func (service *Inbox) DeleteMany(session data.Session, criteria exp.Expression, note string) error {

	rangeFunc, err := service.Range(session, criteria)

	if err != nil {
		return derp.Wrap(err, "service.Inbox.DeleteMany", "Unable to list streams to delete", criteria)
	}

	for message := range rangeFunc {
		if err := service.Delete(session, &message, note); err != nil {
			return derp.Wrap(err, "service.Inbox.DeleteMany", "Unable to delete message", message)
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

// New returns a fully initialized model.Inbox record as a data.Object.
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

func (service *Inbox) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Inbox) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewMessage()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Inbox) ObjectSave(session data.Session, object data.Object, note string) error {
	if message, ok := object.(*model.Message); ok {
		return service.Save(session, message, note)
	}
	return derp.InternalError("service.Inbox.ObjectSave", "Invalid Object Type", object)
}

func (service *Inbox) ObjectDelete(session data.Session, object data.Object, note string) error {
	if message, ok := object.(*model.Message); ok {
		return service.Delete(session, message, note)
	}
	return derp.InternalError("service.Inbox.ObjectDelete", "Invalid Object Type", object)
}

func (service *Inbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Inbox.ObjectUserCan", "Not Authorized")
}

func (service *Inbox) Schema() schema.Schema {
	return schema.New(model.MessageSchema())
}

/******************************************
 * Custom Query Methods
 ******************************************/

func (service *Inbox) QueryByUserID(session data.Session, userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Message, error) {
	criteria = criteria.AndEqual("userId", userID)
	return service.Query(session, criteria, options...)
}

func (service *Inbox) RangeByFolder(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) (iter.Seq[model.Message], error) {
	criteria := exp.Equal("userId", userID).
		AndEqual("folderId", folderID)

	return service.Range(session, criteria)
}

func (service *Inbox) RangeByFollowingID(session data.Session, userID primitive.ObjectID, followingID primitive.ObjectID) (iter.Seq[model.Message], error) {
	criteria := exp.Equal("userId", userID).
		AndEqual("origin.followingId", followingID)

	return service.Range(session, criteria)
}

func (service *Inbox) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.Message], error) {
	return service.Range(session, exp.Equal("userId", userID))
}

func (service *Inbox) LoadByID(session data.Session, userID primitive.ObjectID, messageID primitive.ObjectID, result *model.Message) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("_id", messageID)

	return service.Load(session, criteria, result)
}

// LoadByURL returns the first message that matches the provided UserID and URL
func (service *Inbox) LoadByURL(session data.Session, userID primitive.ObjectID, url string, result *model.Message) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("url", url)

	return service.Load(session, criteria, result)
}

// LoadUnreadByURL returns the first UNREAD message that matches the provided UserID and URL
func (service *Inbox) LoadUnreadByURL(session data.Session, userID primitive.ObjectID, url string, result *model.Message) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("url", url).
		AndEqual("readDate", math.MaxInt64)

	return service.Load(session, criteria, result)
}

// LoadSibling searches for the previous/next sibling to the provided message criteria.
func (service *Inbox) LoadSibling(session data.Session, folderID primitive.ObjectID, rank int64, following string, direction string) (model.Message, error) {

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
	it, err := service.List(session, criteria, option.FirstRow(), sort)

	if err != nil {
		return model.Message{}, derp.Wrap(err, location, "Unable to retrieve siblings")
	}

	// Try to read the results
	result := model.NewMessage()

	// This *should* read the prev/next message into the pointer and be done.
	if it.Next(&result) {
		return result, nil
	}

	// No results.  Shame! Shame!
	return model.Message{}, derp.NotFoundError(location, "Sibling record not found")
}

func (service *Inbox) LoadOldestUnread(session data.Session, userID primitive.ObjectID, message *model.Message) error {

	const location = "service.Inbox.LoadOldestUnread"

	criteria := exp.Equal("userId", userID)
	sort := option.SortAsc("createDate")

	it, err := service.List(session, criteria, option.FirstRow(), sort)

	if err != nil {
		return derp.Wrap(err, location, "Unable to list messages")
	}

	for it.Next(message) {
		return nil
	}

	return derp.NotFoundError(location, "No unread messages")
}

func (service *Inbox) MarkReadByDate(session data.Session, userID primitive.ObjectID, rank int64) error {

	const location = "service.Inbox.MarkReadByDate"

	criteria := exp.Equal("userId", userID).AndLessThan("rank", rank)
	sort := option.SortAsc("rank")

	it, err := service.List(session, criteria, sort)

	if err != nil {
		return derp.Wrap(err, location, "Unable to list messages")
	}

	message := model.NewMessage()
	for it.Next(&message) {
		if err := service.MarkRead(session, &message); err != nil {
			return derp.Wrap(err, location, "Unable to mark message as read")
		}
	}

	return nil
}

/******************************************
 * Custom Behaviors
 ******************************************/

// MarkRead updates a message to "READ" status and recalculates statistics
func (service *Inbox) MarkRead(session data.Session, message *model.Message) error {

	const location = "service.Inbox.MarkRead"

	// Set status to READ.  If the message was not changed, then exit
	if isUpdated := message.MarkRead(); !isUpdated {
		return nil
	}

	// Save the message
	if err := service.Save(session, message, "Update StateID to "+message.StateID); err != nil {
		return derp.Wrap(err, location, "Unable to save message")
	}

	// Recalculate statistics
	if err := service.recalculateUnreadCounts(session, message); err != nil {
		return derp.Wrap(err, location, "Unable to recalculate unread counts")
	}

	// Lo hicimos!
	return nil
}

// MarkRead updates a message to "UNREAD" status and recalculates statistics
func (service *Inbox) MarkUnread(session data.Session, message *model.Message) error {

	const location = "service.Inbox.MarkUnread"

	// Set status to UNREAD.  If the message was not changed, then exit
	if isUpdated := message.MarkUnread(); !isUpdated {
		return nil
	}

	// Save the message
	if err := service.Save(session, message, "Update StateID to "+message.StateID); err != nil {
		return derp.Wrap(err, location, "Unable to save message")
	}

	// Recalculate statistics
	if err := service.recalculateUnreadCounts(session, message); err != nil {
		return derp.Wrap(err, location, "Unable to recalculate unread counts")
	}

	// Success
	return nil
}

func (service *Inbox) MarkMuted(session data.Session, message *model.Message) error {

	const location = "service.Inbox.MarkMuted"

	// Set status to MUTED.  If the message is unchanged, then exit
	if isUpdated := message.MarkMuted(); !isUpdated {
		return nil
	}

	// Save the message
	if err := service.Save(session, message, "Set Status to MUTED"); err != nil {
		return derp.Wrap(err, location, "Unable to save message")
	}

	return nil
}

func (service *Inbox) MarkUnmuted(session data.Session, message *model.Message) error {

	const location = "service.Inbox.MarkUnmuted"

	// Set status to READ (unmuted).  If the message is unchanged, then exit
	if isUpdated := message.MarkRead(); !isUpdated {
		return nil
	}

	// Save the message
	if err := service.Save(session, message, "Set Status to MUTED"); err != nil {
		return derp.Wrap(err, location, "Unable to save message")
	}

	// Success
	return nil
}

// SetResponse sets/clears a Response type from a Message
func (service *Inbox) setResponse(session data.Session, userID primitive.ObjectID, url string, responseType string, responseID primitive.ObjectID) error {

	const location = "service.Inbox.setResponse"

	// Load the message that is being responded to
	message := model.NewMessage()
	if err := service.LoadByURL(session, userID, url, &message); err != nil {

		// Exceptional case: If there is no message to respond to, then do not return an error.
		if derp.IsNotFound(err) {
			return nil
		}

		// Failure and Shame!
		return derp.Wrap(err, location, "Unable to load message by URL", url)
	}

	// Set the response on the message
	if changed := message.Response.SetDelta(responseType, responseID); !changed {
		return nil
	}

	// Save the message
	if err := service.Save(session, &message, "Set Response"); err != nil {
		return derp.Wrap(err, location, "Unable to save message with response")
	}

	// Silence is GoLdEN.
	return nil
}

func (service *Inbox) recalculateUnreadCounts(session data.Session, message *model.Message) error {

	const location = "service.Inbox.recalculateUnreadCounts"

	// Recalculate the "unread" count on the corresponding folder
	unreadCount, err := service.CountUnreadMessages(session, message.UserID, message.FolderID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to count unread messages")
	}

	// Update the "unread" count for the Folder
	if err := service.folderService.SetUnreadCount(session, message.UserID, message.FolderID, unreadCount); err != nil {
		return derp.Wrap(err, location, "Unable to set unread count")
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
func (service *Inbox) CountUnreadMessages(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) (int, error) {

	criteria := exp.Equal("userId", userID).
		AndEqual("folderId", folderID).
		AndEqual("readDate", math.MaxInt64).
		AndEqual("deleteDate", 0)

	count, err := service.collection(session).Count(criteria)
	return int(count), err
}

func (service *Inbox) UpdateInboxFolders(session data.Session, userID primitive.ObjectID, followingID primitive.ObjectID, folderID primitive.ObjectID) error {

	rangeFunc, err := service.RangeByFollowingID(session, userID, followingID)

	if err != nil {
		return derp.Wrap(err, "service.Inbox", "Unable to list Activities by following", userID, followingID)
	}

	for message := range rangeFunc {
		message.FolderID = folderID
		if err := service.Save(session, &message, "UpdateInboxFolders"); err != nil {
			return derp.Wrap(err, "service.Inbox", "Unable to save Inbox Message", message)
		}
	}

	// Recalculate the "unread" count on the new folder
	if err := service.folderService.CalculateUnreadCount(session, userID, folderID); err != nil {
		return derp.Wrap(err, "service.Inbox", "Unable to calculate unread count for new folder", userID, folderID)
	}

	return nil
}

func (service *Inbox) DeleteByUserID(session data.Session, userID primitive.ObjectID, note string) error {
	return service.DeleteMany(session, exp.Equal("userId", userID), note)
}

func (service *Inbox) DeleteByOrigin(session data.Session, internalID primitive.ObjectID, note string) error {
	return service.DeleteMany(session, exp.Equal("origin.followingId", internalID), note)
}

func (service *Inbox) DeleteByFolder(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) error {

	rangeFunc, err := service.RangeByFolder(session, userID, folderID)

	if err != nil {
		return derp.Wrap(err, "service.Inbox", "Unable to list Activities by folder", userID, folderID)
	}

	for message := range rangeFunc {
		if err := service.Delete(session, &message, "DeleteByFolder"); err != nil {
			return derp.Wrap(err, "service.Inbox", "Unable to delete Inbox", message)
		}
	}

	return nil
}

// QueryPurgeable returns a list of Inboxs that are older than the purge date for this following
func (service *Inbox) RangePurgeable(session data.Session, following *model.Following) (iter.Seq[model.Message], error) {

	// Purge date is X days before the current date
	purgeDuration := time.Duration(following.PurgeDuration) * 24 * time.Hour
	purgeDate := time.Now().Add(0 - purgeDuration).Unix()

	// Activities in the INBOX can be purged if they are READ and older than the purge date
	criteria := exp.
		Equal("followingId", following.FollowingID).
		AndGreaterThan("readDate", 0).
		AndLessThan("readDate", purgeDate)

	return service.Range(session, criteria)
}
