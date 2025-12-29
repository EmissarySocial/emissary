package service

import (
	"iter"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/turbine/queue"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// followingMimeStack lists the preferred mime types for follows
const followingMimeStack = "application/activity+json; q=1.0, text/html; q=0.9, application/feed+json; q=0.8, application/atom+xml; q=0.7, application/rss+xml; q=0.6, text/xml; q=0.5, */*; q=0.1"

// Following manages all interactions with the Following collection
type Following struct {
	factory           *Factory
	importItemService *ImportItem
	streamService     *Stream
	userService       *User
	inboxService      *Inbox
	folderService     *Folder
	keyService        *EncryptionKey
	queue             *queue.Queue
	sseUpdateChannel  chan<- realtime.Message
	host              string
	closed            chan bool
}

// NewFollowing returns a fully populated Following service.
func NewFollowing(factory *Factory) Following {
	return Following{
		factory: factory,
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Following) Refresh(folderService *Folder, keyService *EncryptionKey, importItemService *ImportItem, inboxService *Inbox, streamService *Stream, userService *User, queue *queue.Queue, sseUpdateChannel chan<- realtime.Message, host string) {
	service.folderService = folderService
	service.keyService = keyService
	service.importItemService = importItemService
	service.inboxService = inboxService
	service.streamService = streamService
	service.userService = userService
	service.queue = queue
	service.sseUpdateChannel = sseUpdateChannel
	service.host = host
}

// Close stops the following service watcher
func (service *Following) Close() {
	close(service.closed)
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Following) collection(session data.Session) data.Collection {
	return session.Collection("Following")
}

// New creates a newly initialized Following that is ready to use
func (service *Following) New() model.Following {
	return model.NewFollowing()
}

// Count returns the number of records that match the provided criteria
func (service *Following) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns an iterator containing all of the Following who match the provided criteria
func (service *Following) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Following, error) {
	result := make([]model.Following, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Following who match the provided criteria
func (service *Following) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Following records that match the provided criteria
func (service *Following) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Following], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Following.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewFollowing), nil
}

// Load retrieves an Following from the database
func (service *Following) Load(session data.Session, criteria exp.Expression, result *model.Following) error {

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Following.Load", "Unable to load Following", criteria)
	}

	return nil
}

// Save adds/updates an Following in the database
func (service *Following) Save(session data.Session, following *model.Following, note string) error {

	const location = "service.Following.Save"

	// Default following behavior
	if following.Behavior == "" {
		following.Behavior = model.FollowingBehaviorPostsAndReplies
	}

	// Default rule behavior
	if following.RuleAction == "" {
		following.RuleAction = model.RuleActionLabel
	}

	// RULE: Update Polling duration based on the transmission method
	switch following.Method {

	case model.FollowingMethodActivityPub:
		following.PollDuration = 24 * 7 * 30 // retry ActivityPub connections every 30 days

	case model.FollowingMethodWebSub:
		following.PollDuration = 24 * 7 // retry WebSub connections every 7 days

	default:
		following.PollDuration = 24
	}

	// Validate the value before saving
	if err := service.Schema().Validate(following); err != nil {
		return derp.Wrap(err, location, "Unable to validate Following record", following)
	}

	// Prevent duplicate following records
	if err := service.preventDuplicates(session, following); err != nil {
		return derp.Wrap(err, location, "Unable to prevent duplicate", following)
	}

	// RULE: IF changed, retrieve the Folder Name
	if following.FolderID.IsChanged() {

		// Get the new Folder label
		folder := model.NewFolder()
		if err := service.folderService.LoadByID(session, following.UserID, following.FolderID.Value(), &folder); err == nil {
			following.Folder = folder.Label
		}

		// Move related inbox items to the new folder
		if err := service.inboxService.UpdateInboxFolders(session, following.UserID, following.FollowingID, following.FolderID.Value()); err != nil {
			return derp.Wrap(err, location, "Unable to update Inbox Folders")
		}
	}

	// Save the following to the database
	if err := service.collection(session).Save(following, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Following", following, note)
	}

	// Notify the user that their Following list has been changed
	service.sseUpdateChannel <- realtime.NewMessage_FollowingUpdated(following.UserID)

	// Done.. UNLESS creating a new Following record
	if following.Status != model.FollowingStatusNew {
		return nil
	}

	// Fall through means we have to connect to external services
	// Reset status and error counts when saving
	following.StatusMessage = ""
	following.ErrorCount = 0

	// Recalculate the follower count for this user
	if err := service.userService.CalcFollowingCount(session, following.UserID); err != nil {
		return derp.Wrap(err, location, "Unable to count `Following` records")
	}

	// Run follow-on tasks asynchronously
	if err := service.Connect(session, following); err != nil {
		return derp.Wrap(err, location, "Unable to initiate external service connection")
	}

	// Win!
	return nil
}

// Delete removes an Following from the database (virtual delete)
func (service *Following) Delete(session data.Session, following *model.Following, note string) error {

	const location = "service.Following.Delete"

	if err := service.deleteNoStats(session, following, note); err != nil {
		return derp.Wrap(err, location, "Unable to delete Following", following, note)
	}

	// Recalculate the follower count for this user
	if err := service.userService.CalcFollowingCount(session, following.UserID); err != nil {
		return derp.Wrap(err, location, "Unable to calculate Following count")
	}

	// Recalculate the unread count for this folder
	if err := service.folderService.CalculateUnreadCount(session, following.UserID, following.FolderID.Value()); err != nil {
		return derp.Wrap(err, location, "Unable to calculate Unread count")
	}

	return nil
}

// deleteNoStats removes an Following from the database (virtual delete)
// but DOES NOT recompute statistics for parent records.  This is useful when
// cascading deletes, because there's no reason to recompute statistics for
// records that will be deleted.
func (service *Following) deleteNoStats(session data.Session, following *model.Following, comment string) error {

	const location = "service.Following.deleteNoStats"

	// Remove the Following record
	if err := service.collection(session).Delete(following, comment); err != nil {
		return derp.Wrap(err, location, "Unable to delete Following", following, comment)
	}

	// Remove any messages received from this Following
	if err := service.inboxService.DeleteByOrigin(session, following.FollowingID, "Parent record deleted"); err != nil {
		return derp.Wrap(err, location, "Unable to delete streams for Following", following)
	}

	// Disconnect from external services (if necessary)
	go service.Disconnect(session, following)

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *Following) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Following record, without applying any additional business rules
func (service *Following) HardDeleteByID(session data.Session, userID primitive.ObjectID, followingID primitive.ObjectID) error {

	const location = "service.Following.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", followingID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Following", "userID: "+userID.Hex(), "followingID: "+followingID.Hex())
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Following) ObjectType() string {
	return "Following"
}

// New returns a fully initialized model.Following as a data.Object.
func (service *Following) ObjectNew() data.Object {
	result := model.NewFollowing()
	return &result
}

// ObjectID returns the ID of a following object
func (service *Following) ObjectID(object data.Object) primitive.ObjectID {

	if following, ok := object.(*model.Following); ok {
		return following.FollowingID
	}

	return primitive.NilObjectID
}

func (service *Following) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Following) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewFollowing()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Following) ObjectSave(session data.Session, object data.Object, note string) error {
	if following, ok := object.(*model.Following); ok {
		return service.Save(session, following, note)
	}
	return derp.InternalError("service.Following.ObjectSave", "Invalid object type", object)
}

func (service *Following) ObjectDelete(session data.Session, object data.Object, note string) error {
	if following, ok := object.(*model.Following); ok {
		return service.Delete(session, following, note)
	}
	return derp.InternalError("service.Following.ObjectDelete", "Invalid object type", object)
}

func (service *Following) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Following.ObjectUserCan", "Not Authorized")
}

func (service *Following) Schema() schema.Schema {
	return schema.New(model.FollowingSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// QueryByUser returns a slice of all following for a given user
func (service *Following) QueryByUser(session data.Session, userID primitive.ObjectID) ([]model.FollowingSummary, error) {
	result := make([]model.FollowingSummary, 0)
	criteria := exp.Equal("userId", userID)
	err := service.collection(session).Query(&result, notDeleted(criteria), option.Fields(model.FollowingSummaryFields()...), option.SortAsc("label"))
	return result, err
}

// QueryByFolder returns a slice of all following for a given user
func (service *Following) QueryByFolder(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) ([]model.FollowingSummary, error) {
	result := make([]model.FollowingSummary, 0)
	criteria := exp.Equal("userId", userID).AndEqual("folderId", folderID)
	err := service.collection(session).Query(&result, notDeleted(criteria), option.Fields(model.FollowingSummaryFields()...), option.SortAsc("label"))
	return result, err
}

// QueryByFolderAndExp returns a slice of all following for a given user
func (service *Following) QueryByFolderAndExp(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID, criteria exp.Expression) ([]model.FollowingSummary, error) {

	result := make([]model.FollowingSummary, 0)
	criteria = criteria.
		AndEqual("userId", userID).
		AndEqual("folderId", folderID)

	err := service.collection(session).Query(&result, notDeleted(criteria), option.Fields(model.FollowingSummaryFields()...), option.SortAsc("label"))
	return result, err
}

// RangePollable returns an iterator of all following that are ready to be polled
func (service *Following) RangePollable(session data.Session) (iter.Seq[model.Following], error) {
	criteria := exp.LessThan("nextPoll", time.Now().Unix())

	return service.Range(session, criteria, option.SortAsc("lastPolled"))
}

// RangeByActorID returns an iterator of all following records that use the provided `ProfileURL`
func (service *Following) RangeByActorID(session data.Session, actorID string) (iter.Seq[model.Following], error) {
	criteria := exp.Equal("profileUrl", actorID)
	return service.Range(session, criteria)
}

// RangeByUserID returns an iterator of all following for a given userID
func (service *Following) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.Following], error) {
	criteria := exp.Equal("userId", userID)
	return service.Range(session, criteria)
}

// RangeByFolderID returns an iterator containing all of the Folders for a given user/folder
func (service *Following) RangeByFolderID(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) (iter.Seq[model.Following], error) {
	criteria := exp.Equal("userId", userID).AndEqual("_id", folderID)
	return service.Range(session, criteria)
}

// LoadByID retrieves an Following from the database.  UserID is required to prevent
// people from snooping on other's following.
func (service *Following) LoadByID(session data.Session, userID primitive.ObjectID, followingID primitive.ObjectID, result *model.Following) error {

	const location = "service.Following.LoadByID"

	criteria := exp.Equal("_id", followingID).
		AndEqual("userId", userID)

	if err := service.Load(session, criteria, result); err != nil {
		return derp.Wrap(err, location, "Unable to load Following", criteria)
	}

	return nil
}

// LoadByToken loads an individual following using a string version of the following ID
func (service *Following) LoadByToken(session data.Session, userID primitive.ObjectID, token string, result *model.Following) error {

	const location = "service.Following.LoadByToken"

	if token == "new" {
		*result = model.NewFollowing()
		result.UserID = userID
		return nil
	}

	followingID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "FollowingId must be a valid ObjectID", token)
	}

	return service.LoadByID(session, userID, followingID, result)
}

// LoadByURL loads an individual following using the target URL that is being followed
func (service *Following) LoadByURL(session data.Session, userID primitive.ObjectID, profileUrl string, result *model.Following) error {

	criteria := exp.Equal("userId", userID).
		AndEqual("profileUrl", profileUrl)

	return service.Load(session, criteria, result)
}

/******************************************
 * Custom Actions
 ******************************************/

func (service *Following) GetFollowingID(session data.Session, userID primitive.ObjectID, uri string) (string, error) {

	const location = "service.Following.IsFollowing"

	// Load the ActivityStream document
	activityService := service.factory.ActivityStream(model.ActorTypeUser, userID)
	document, err := activityService.Client().Load(uri)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to loadActivityStream document", uri)
	}

	// If this document is not an Actor, then get the Actor of the document
	if !document.IsActor() {
		document = document.Actor()
	}

	// If this document is nil, then return an error
	if document.IsNil() {
		return "", derp.BadRequestError(location, "Invalid ActivityStream document", uri)
	}

	// Look for the Actor in the Following collection
	following := model.NewFollowing()

	if err := service.LoadByURL(session, userID, document.ID(), &following); err != nil {
		if derp.IsNotFound(err) {
			return "", nil
		}
		return "", derp.Wrap(err, location, "Unable to load Following record", uri)
	}

	return following.ID(), nil
}

// DeleteByUserID removes all Following records for the provided userID
func (service *Following) DeleteByUserID(session data.Session, userID primitive.ObjectID, comment string) error {

	const location = "service.Following.DeleteByUserID"

	// Load all Following for the provided userID
	rangeFunc, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to loadfollowing", userID)
	}

	// Delete each Following record
	for following := range rangeFunc {
		if err := service.deleteNoStats(session, &following, comment); err != nil {
			return derp.Wrap(err, location, "Unable to delete following", following)
		}
	}

	// No Cap.
	return nil
}

// DeleteByFolder removes all Following records for the provided folderID
func (service *Following) DeleteByFolder(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID, comment string) error {

	rangeFunc, err := service.RangeByFolderID(session, userID, folderID)

	if err != nil {
		return derp.Wrap(err, "service.Folder.DeleteByFolder", "Unable to list folders", userID, folderID)
	}

	for folder := range rangeFunc {
		if err := service.Delete(session, &folder, comment); err != nil {
			return derp.Wrap(err, "service.Folder.DeleteByFolder", "Unable to delete folder", folder)
		}
	}

	// Skibidi.
	return nil
}

// PurgeInbox removes all inbox items that are past their expiration date.
// TODO: HIGH: This should be rescheduled to run less frequently
func (service *Following) PurgeInbox(session data.Session, following model.Following) error {

	const location = "service.Following.PurgeFollowing"

	// Check each following for expired items.
	messages, err := service.inboxService.RangePurgeable(session, &following)

	if err != nil {
		return derp.Wrap(err, location, "Error querying purgeable items", following)
	}

	// Purge each item that has expired
	for message := range messages {
		if err := service.inboxService.Delete(session, &message, "Purged"); err != nil {
			return derp.Wrap(err, location, "Error purging message", message)
		}
	}

	return nil
}

// Move updates a Following record to point to a new Profile URL
// and resets its status so that we will try to reconnect to the new URL.
func (service *Following) Move(session data.Session, following *model.Following, targetURL string) error {

	const location = "service.Following.Move"

	// If the target URL is the same as the current URL, do nothing
	if following.ProfileURL == targetURL {
		return nil
	}

	// Reset this Following record to connect to the new URL
	following.ProfileURL = targetURL
	following.Status = model.FollowingStatusNew
	following.Method = ""
	following.StatusMessage = ""
	following.ErrorCount = 0

	// Save the updated Following record
	if err := service.Save(session, following, "Moving to new URL"); err != nil {
		return derp.Wrap(err, location, "Unable to save moved Following", following)
	}

	// Win!
	return nil
}

/******************************************
 * Status Update Methods
 ******************************************/

// SetStatusLoading updates a Following record with the "Loading" status
func (service *Following) SetStatusLoading(session data.Session, following *model.Following) error {

	// Update Following state
	following.Status = model.FollowingStatusLoading
	following.StatusMessage = ""
	following.LastPolled = time.Now().Unix()

	// Save the Following to the database (no other busines rules)
	if err := service.collection(session).Save(following, "Updating status"); err != nil {
		return derp.Wrap(err, "service.Following.SetStatusLoading", "Unable to save Following", following)
	}

	// Notify the user that their Following list has been changed
	service.sseUpdateChannel <- realtime.NewMessage_FollowingUpdated(following.UserID)

	return nil
}

func (service *Following) SetStatusPolling(session data.Session, following *model.Following) error {

	// Update Following state
	following.Status = model.FollowingStatusSuccess
	following.Method = model.FollowingMethodPoll
	following.StatusMessage = ""

	// Save the Following to the database (no other busines rules)
	if err := service.collection(session).Save(following, "Updating status"); err != nil {
		return derp.Wrap(err, "service.Following.SetStatusPolling", "Unable to save Following", following)
	}

	// Notify the user that their Following list has been changed
	service.sseUpdateChannel <- realtime.NewMessage_FollowingUpdated(following.UserID)

	return nil
}

// SetStatusSuccess updates a Following record with the "Success" status and
// resets the error count to zero.
func (service *Following) SetStatusSuccess(session data.Session, following *model.Following) error {

	// Update Following state
	following.Status = model.FollowingStatusSuccess
	following.StatusMessage = ""

	following.NextPoll = following.LastPolled + int64(following.PollDuration*60*60)
	following.ErrorCount = 0

	// Save the Following to the database (no other busines rules)
	if err := service.collection(session).Save(following, "Updating status"); err != nil {
		return derp.Wrap(err, "service.Following.SetStatusSuccess", "Unable to save Following", following)
	}

	// Notify the user that their Following list has been changed
	service.sseUpdateChannel <- realtime.NewMessage_FollowingUpdated(following.UserID)

	return nil
}

// SetStatusFailure updates a Following record to the "Failure" status and
// increments the error count.
func (service *Following) SetStatusFailure(session data.Session, following *model.Following, statusMessage string) error {

	// Update Following state
	following.Status = model.FollowingStatusFailure
	following.StatusMessage = statusMessage
	following.ErrorCount = following.ErrorCount + 1

	// On failure, compute exponential backoff
	// Wait times are 1m, 2m, 4m, 8m, 16m, 32m, 64m, 128m, 256m (max ~4 hours)
	// But do not change "LastPolled" because that is the last time we were successful
	errorBackoff := following.ErrorCount

	if errorBackoff > 8 {
		errorBackoff = 8
	}

	errorBackoff = 2 ^ errorBackoff
	following.NextPoll = time.Now().Add(time.Duration(errorBackoff) * time.Minute).Unix()

	// Save the Following to the database (no other busines rules)
	if err := service.collection(session).Save(following, "Updating status"); err != nil {
		return derp.Wrap(err, "service.Following.SetStatusFailure", "Unable to save Following", following)
	}

	// Notify the user that their Following list has been changed
	service.sseUpdateChannel <- realtime.NewMessage_FollowingUpdated(following.UserID)

	return nil
}

/******************************************
 * ActivityPub Data Accessors
 ******************************************/

// ActivityPubID returns the public URL (ID) of a Following record
func (service *Following) ActivityPubID(following *model.Following) string {
	return service.host + "/@" + following.UserID.Hex() + "/pub/following/" + following.FollowingID.Hex()
}

// ActivityPubActorID returns the public URL (ID) of the actor being followed
func (service *Following) ActivityPubActorID(following *model.Following) string {
	return service.host + "/@" + following.UserID.Hex()
}

// AsJSONLD returns a Following record as a JSON-LD object
func (service *Following) AsJSONLD(following *model.Following) mapof.Any {

	return mapof.Any{
		"@context": vocab.ContextTypeActivityStreams,
		"id":       service.ActivityPubID(following),
		"type":     vocab.ActivityTypeFollow,
		"actor":    service.ActivityPubActorID(following),
		"object":   following.URL,
	}
}

/******************************************
 * Helper Methods
 ******************************************/

func (service *Following) preventDuplicates(session data.Session, current *model.Following) error {

	const location = "service.Following.preventDuplicate"

	// Search the database for the original record
	original := model.NewFollowing()
	if err := service.LoadByURL(session, current.UserID, current.URL, &original); err != nil {
		if derp.IsNotFound(err) {
			return nil
		}
		return derp.Wrap(err, location, "Unable to load Following", current)
	}

	// If the original and current are the same, then do nothing
	if original.FollowingID == current.FollowingID {
		return nil
	}

	// Otherwise, DELETE the original record
	if err := service.Delete(session, &original, "removing duplicate"); err != nil {
		return derp.Wrap(err, location, "Unable to delete original", original)
	}

	return nil
}
