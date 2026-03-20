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
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NewsFeed manages the NewsItem records for a User.
type NewsFeed struct {
	importItemService *ImportItem
	folderService     *Folder
	ruleService       *Rule
	host              string
	counter           int
	mutex             *sync.Mutex
}

// NewNewsFeed returns a fully populated NewsFeed service
func NewNewsFeed() NewsFeed {
	return NewsFeed{
		mutex: &sync.Mutex{},
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *NewsFeed) Refresh(factory *Factory) {
	service.importItemService = factory.ImportItem()
	service.folderService = factory.Folder()
	service.ruleService = factory.Rule()
	service.host = factory.Host()
}

// Close stops any background processes controlled by this service
func (service *NewsFeed) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *NewsFeed) collection(session data.Session) data.Collection {
	return session.Collection("NewsFeed")
}

// New creates a newly initialized NewsFeed that is ready to use
func (service *NewsFeed) New() model.NewsItem {
	return model.NewNewsItem()
}

// Count returns the number of records that match the provided criteria
func (service *NewsFeed) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *NewsFeed) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.NewsItem, error) {
	result := make([]model.NewsItem, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Activities that match the provided criteria
func (service *NewsFeed) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the NewsItems that match the provided criteria
func (service *NewsFeed) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.NewsItem], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.NewsFeed.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewNewsItem), nil
}

// Load retrieves an NewsFeed from the database
func (service *NewsFeed) Load(session data.Session, criteria exp.Expression, result *model.NewsItem) error {

	const location = "service.NewsFeed.Load"

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, location, "Unable to load NewsFeed message", criteria)
	}

	return nil
}

// Save adds/updates an NewsFeed in the database
func (service *NewsFeed) Save(session data.Session, message *model.NewsItem, note string) error {

	const location = "service.NewsFeed.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(message); err != nil {
		return derp.Wrap(err, location, "Unable to validate NewsFeed", message)
	}

	// Calculate a (hopefully unique) rank for this message
	service.CalculateRank(message)

	// Save the value to the database
	if err := service.collection(session).Save(message, note); err != nil {
		return derp.Wrap(err, location, "Unable to save NewsFeed", message, note)
	}

	// Recalculate the unread count for the folder that owns this message.
	if err := service.folderService.CalculateUnreadCount(session, message.UserID, message.FolderID); err != nil {
		return derp.Wrap(err, location, "Unable to recalculate unread count", message)
	}

	// Wait 1 millisecond between each document to guarantee sorting by CreateDate
	time.Sleep(1 * time.Millisecond)

	return nil
}

// Delete removes an NewsFeed from the database (virtual delete)
func (service *NewsFeed) Delete(session data.Session, message *model.NewsItem, note string) error {

	// Delete NewsFeed record last.
	if err := service.collection(session).Delete(message, note); err != nil {
		return derp.Wrap(err, "service.NewsFeed.Delete", "Unable to delete NewsFeed", message, note)
	}

	return nil
}

// DeleteMany removes all child streams from the provided stream (virtual delete)
func (service *NewsFeed) DeleteMany(session data.Session, criteria exp.Expression, note string) error {

	rangeFunc, err := service.Range(session, criteria)

	if err != nil {
		return derp.Wrap(err, "service.NewsFeed.DeleteMany", "Unable to list streams to delete", criteria)
	}

	for message := range rangeFunc {
		if err := service.Delete(session, &message, note); err != nil {
			return derp.Wrap(err, "service.NewsFeed.DeleteMany", "Unable to delete message", message)
		}
	}

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *NewsFeed) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific NewsFeed record, without applying any additional business rules
func (service *NewsFeed) HardDeleteByID(session data.Session, userID primitive.ObjectID, messageID primitive.ObjectID) error {

	const location = "service.NewsFeed.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", messageID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete NewsFeed", "userID: "+userID.Hex(), "messageID: "+messageID.Hex())
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *NewsFeed) ObjectType() string {
	return "NewsFeed"
}

// New returns a fully initialized model.NewsFeed record as a data.Object.
func (service *NewsFeed) ObjectNew() data.Object {
	result := model.NewNewsItem()
	return &result
}

func (service *NewsFeed) ObjectID(object data.Object) primitive.ObjectID {

	if message, ok := object.(*model.NewsItem); ok {
		return message.NewsItemID
	}

	return primitive.NilObjectID
}

func (service *NewsFeed) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *NewsFeed) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewNewsItem()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *NewsFeed) ObjectSave(session data.Session, object data.Object, note string) error {
	if message, ok := object.(*model.NewsItem); ok {
		return service.Save(session, message, note)
	}
	return derp.Internal("service.NewsFeed.ObjectSave", "Invalid Object Type", object)
}

func (service *NewsFeed) ObjectDelete(session data.Session, object data.Object, note string) error {
	if message, ok := object.(*model.NewsItem); ok {
		return service.Delete(session, message, note)
	}
	return derp.Internal("service.NewsFeed.ObjectDelete", "Invalid Object Type", object)
}

func (service *NewsFeed) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.NewsFeed.ObjectUserCan", "Not Authorized")
}

func (service *NewsFeed) Schema() schema.Schema {
	return schema.New(model.NewsItemSchema())
}

/******************************************
 * Custom Query Methods
 ******************************************/

func (service *NewsFeed) QueryByUserID(session data.Session, userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.NewsItem, error) {
	criteria = criteria.AndEqual("userId", userID)
	return service.Query(session, criteria, options...)
}

func (service *NewsFeed) RangeByFolder(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) (iter.Seq[model.NewsItem], error) {
	criteria := exp.Equal("userId", userID).
		AndEqual("folderId", folderID)

	return service.Range(session, criteria)
}

func (service *NewsFeed) RangeByFollowingID(session data.Session, userID primitive.ObjectID, followingID primitive.ObjectID) (iter.Seq[model.NewsItem], error) {
	criteria := exp.Equal("userId", userID).
		AndEqual("origin.followingId", followingID)

	return service.Range(session, criteria)
}

func (service *NewsFeed) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.NewsItem], error) {
	return service.Range(session, exp.Equal("userId", userID))
}

func (service *NewsFeed) LoadByID(session data.Session, userID primitive.ObjectID, messageID primitive.ObjectID, result *model.NewsItem) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("_id", messageID)

	return service.Load(session, criteria, result)
}

// LoadByURL returns the first message that matches the provided UserID and URL
func (service *NewsFeed) LoadByURL(session data.Session, userID primitive.ObjectID, url string, result *model.NewsItem) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("url", url)

	return service.Load(session, criteria, result)
}

// LoadUnreadByURL returns the first UNREAD message that matches the provided UserID and URL
func (service *NewsFeed) LoadUnreadByURL(session data.Session, userID primitive.ObjectID, url string, result *model.NewsItem) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("url", url).
		AndEqual("readDate", math.MaxInt64)

	return service.Load(session, criteria, result)
}

// LoadSibling searches for the previous/next sibling to the provided message criteria.
func (service *NewsFeed) LoadSibling(session data.Session, folderID primitive.ObjectID, rank int64, following string, direction string) (model.NewsItem, error) {

	const location = "service.NewsFeed.LoadSibling"

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
		return model.NewsItem{}, derp.Wrap(err, location, "Unable to retrieve siblings")
	}

	// This *should* read the prev/next message into the pointer and be done.
	if result := model.NewNewsItem(); it.Next(&result) {
		return result, nil
	}

	// No results.  Shame! Shame!
	return model.NewsItem{}, derp.NotFound(location, "Sibling record not found")
}

func (service *NewsFeed) LoadOldestUnread(session data.Session, userID primitive.ObjectID, message *model.NewsItem) error {

	const location = "service.NewsFeed.LoadOldestUnread"

	criteria := exp.Equal("userId", userID)
	sort := option.SortAsc("createDate")

	it, err := service.List(session, criteria, option.FirstRow(), sort)

	if err != nil {
		return derp.Wrap(err, location, "Unable to list messages")
	}

	for it.Next(message) {
		return nil
	}

	return derp.NotFound(location, "No unread messages")
}

func (service *NewsFeed) MarkReadByDate(session data.Session, userID primitive.ObjectID, rank int64) error {

	const location = "service.NewsFeed.MarkReadByDate"

	criteria := exp.Equal("userId", userID).AndLessThan("rank", rank)
	sort := option.SortAsc("rank")

	it, err := service.List(session, criteria, sort)

	if err != nil {
		return derp.Wrap(err, location, "Unable to list messages")
	}

	// Loop through every message and mark it as read
	for message := model.NewNewsItem(); it.Next(&message); message = model.NewNewsItem() {
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
func (service *NewsFeed) MarkRead(session data.Session, message *model.NewsItem) error {

	const location = "service.NewsFeed.MarkRead"

	// Set status to READ.  If the message was not changed, then exit
	if isUpdated := message.MarkRead(); !isUpdated {
		return nil
	}

	// Save the message
	if err := service.Save(session, message, "Update StateID to "+message.StateID); err != nil {
		return derp.Wrap(err, location, "Unable to save message")
	}

	// Recalculate statistics
	if err := service.folderService.CalculateUnreadCount(session, message.UserID, message.FolderID); err != nil {
		return derp.Wrap(err, location, "Unable to set unread count")
	}

	// Lo hicimos!
	return nil
}

// MarkRead updates a message to "UNREAD" status and recalculates statistics
func (service *NewsFeed) MarkUnread(session data.Session, message *model.NewsItem) error {

	const location = "service.NewsFeed.MarkUnread"

	// Set status to UNREAD.  If the message was not changed, then exit
	if isUpdated := message.MarkUnread(); !isUpdated {
		return nil
	}

	// Save the message
	if err := service.Save(session, message, "Update StateID to "+message.StateID); err != nil {
		return derp.Wrap(err, location, "Unable to save message")
	}

	// Recalculate statistics
	if err := service.folderService.CalculateUnreadCount(session, message.UserID, message.FolderID); err != nil {
		return derp.Wrap(err, location, "Unable to set unread count")
	}

	// Success
	return nil
}

func (service *NewsFeed) MarkMuted(session data.Session, message *model.NewsItem) error {

	const location = "service.NewsFeed.MarkMuted"

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

func (service *NewsFeed) MarkUnmuted(session data.Session, message *model.NewsItem) error {

	const location = "service.NewsFeed.MarkUnmuted"

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

// SetResponse sets/clears a Response type from a NewsItem
func (service *NewsFeed) setResponse(session data.Session, userID primitive.ObjectID, url string, responseType string, responseID primitive.ObjectID) error {

	const location = "service.NewsFeed.setResponse"

	// Load the message that is being responded to
	message := model.NewNewsItem()
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

// CalculateRank generates a unique rank for the message based on the PublishDate and the number of messages
// that already exist in the database with this PublishDate.
func (service *NewsFeed) CalculateRank(message *model.NewsItem) {

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

// CountUnreadNewsItems counts the number of messages for a user/folder that are marked "unread".
func (service *NewsFeed) CountUnreadNewsItems(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) (int, error) {

	criteria := exp.Equal("userId", userID).
		AndEqual("folderId", folderID).
		AndEqual("readDate", math.MaxInt64).
		AndEqual("deleteDate", 0)

	count, err := service.collection(session).Count(criteria)
	return int(count), err
}

func (service *NewsFeed) UpdateNewsFeedFolders(session data.Session, userID primitive.ObjectID, followingID primitive.ObjectID, folderID primitive.ObjectID) error {

	rangeFunc, err := service.RangeByFollowingID(session, userID, followingID)

	if err != nil {
		return derp.Wrap(err, "service.NewsFeed", "Unable to list Activities by following", userID, followingID)
	}

	for message := range rangeFunc {
		message.FolderID = folderID
		if err := service.Save(session, &message, "UpdateNewsFeedFolders"); err != nil {
			return derp.Wrap(err, "service.NewsFeed", "Unable to save NewsFeed NewsItem", message)
		}
	}

	// Recalculate the "unread" count on the new folder
	if err := service.folderService.CalculateUnreadCount(session, userID, folderID); err != nil {
		return derp.Wrap(err, "service.NewsFeed", "Unable to calculate unread count for new folder", userID, folderID)
	}

	return nil
}

func (service *NewsFeed) DeleteByUserID(session data.Session, userID primitive.ObjectID, note string) error {
	return service.DeleteMany(session, exp.Equal("userId", userID), note)
}

func (service *NewsFeed) DeleteByOrigin(session data.Session, internalID primitive.ObjectID, note string) error {
	return service.DeleteMany(session, exp.Equal("origin.followingId", internalID), note)
}

func (service *NewsFeed) DeleteByFolder(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) error {

	rangeFunc, err := service.RangeByFolder(session, userID, folderID)

	if err != nil {
		return derp.Wrap(err, "service.NewsFeed", "Unable to list Activities by folder", userID, folderID)
	}

	for message := range rangeFunc {
		if err := service.Delete(session, &message, "DeleteByFolder"); err != nil {
			return derp.Wrap(err, "service.NewsFeed", "Unable to delete NewsFeed", message)
		}
	}

	return nil
}

// QueryPurgeable returns a list of NewsFeeds that are older than the purge date for this following
func (service *NewsFeed) RangePurgeable(session data.Session, following *model.Following) (iter.Seq[model.NewsItem], error) {

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
