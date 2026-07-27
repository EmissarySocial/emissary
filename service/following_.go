package service

import (
	"iter"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/tools/asrules"
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
const followingMimeStack = "application/activity+json; q=1.0, text/html; q=0.9, application/feed+json; q=0.8, application/atom+xml; q=0.7, application/rss+xml; q=0.6, text/xml; q=0.5, */*; q=0.1" //nolint:unused // retained for future use

// Following manages all interactions with the Following collection
type Following struct {
	activityService   *ActivityStream
	folderService     *Folder
	host              string
	hostname          string
	importItemService *ImportItem
	keyService        *EncryptionKey
	newsFeedService   *NewsFeed
	outboxService     *Outbox
	ruleService       *Rule
	sseUpdateChannel  chan<- realtime.Message
	streamService     *Stream
	userService       *User
	queue             *queue.Queue
}

// NewFollowing returns a fully populated Following service.
func NewFollowing() Following {
	return Following{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Following) Refresh(factory *Factory) {
	service.activityService = factory.ActivityStream()
	service.folderService = factory.Folder()
	service.host = factory.Host()
	service.hostname = factory.Hostname()
	service.importItemService = factory.ImportItem()
	service.keyService = factory.EncryptionKey()
	service.newsFeedService = factory.NewsFeed()
	service.outboxService = factory.Outbox()
	service.ruleService = factory.Rule()
	service.queue = factory.Queue()
	service.sseUpdateChannel = factory.SSEUpdateChannel()
	service.streamService = factory.Stream()
	service.userService = factory.User()
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the data collection for Following records
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
		return nil, derp.Wrap(err, "service.Following.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewFollowing), nil
}

// Load retrieves an Following from the database
func (service *Following) Load(session data.Session, criteria exp.Expression, result *model.Following) error {

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Following.Load", "Loading Following", criteria)
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

	// RULE: Update Polling duration based on the transmission method
	switch following.Method {

	case model.FollowingMethodActivityPub:
		following.PollDuration = 24 * 7 * 30 // retry ActivityPub connections every 30 days

	default:
		following.PollDuration = 24
	}

	// Validate the value before saving
	if _, err := service.Schema().Validate(following); err != nil {
		return derp.Wrap(err, location, "Validating Following record", following)
	}

	// RULE: R11 -- following an actor this User has blocked is refused at creation. Existing
	// records stay editable, so block-driven pause/cleanup flows can still update them.
	if following.IsNew() {

		keys := append(model.ActorMatchKeys(following.URL), model.ActorMatchKeys(following.ProfileURL)...)
		disposition, err := service.ruleService.DispositionForKeys(session, following.UserID, keys, time.Now().Unix())

		if err != nil {
			return derp.Wrap(err, location, "Checking rules before following", following.URL)
		}

		if disposition.IsBlocked() {
			return derp.Validation("You have blocked this account. Remove the block rule before following it.")
		}
	}

	// RULE: The Folder MUST belong to this User
	if err := service.setFolder(session, following); err != nil {
		return derp.Wrap(err, location, "Setting Folder", following)
	}

	// RULE: IF the Folder changed, then move related inbox items into it
	if following.FolderID.IsChanged() {
		if err := service.newsFeedService.UpdateNewsFeedFolders(session, following.UserID, following.FollowingID, following.FolderID.Value()); err != nil {
			return derp.Wrap(err, location, "Updating NewsFeed Folders")
		}
	}

	// Save the following, folding onto any existing follow of the same actor and retrying once if
	// a concurrent create won the unique index (idx_Following_User_Profile_Unique).
	if err := service.reconcileAndSave(session, following, func() error {
		return service.collection(session).Save(following, note)
	}); err != nil {
		return derp.Wrap(err, location, "Saving Following", following, note)
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
		return derp.Wrap(err, location, "Counting `Following` records")
	}

	// Run follow-on tasks asynchronously
	if err := service.Connect(session, following); err != nil {
		return derp.Wrap(err, location, "Initiating external service connection")
	}

	// Win!
	return nil
}

// Delete removes an Following from the database (virtual delete)
func (service *Following) Delete(session data.Session, following *model.Following, note string) error {

	const location = "service.Following.Delete"

	// Remove the Following record from the database. deleteNoStats handles the external
	// disconnect (e.g. Undo/Follow), so we do NOT disconnect here — doing both double-sent
	// the Undo. (Previously this line spawned `go service.Disconnect(...)`, a goroutine that
	// used the request's session after the transaction returned — a use-after-free hazard. F4.)
	if err := service.deleteNoStats(session, following, note); err != nil {
		return derp.Wrap(err, location, "Deleting Following", following, note)
	}

	// Recalculate the follower count for this user
	if err := service.userService.CalcFollowingCount(session, following.UserID); err != nil {
		return derp.Wrap(err, location, "Calculating Following count")
	}

	// Recalculate the unread count for this folder
	if err := service.folderService.CalculateUnreadCount(session, following.UserID, following.FolderID.Value()); err != nil {
		return derp.Wrap(err, location, "Calculating Unread count")
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
		return derp.Wrap(err, location, "Deleting Following", following, comment)
	}

	// Remove any messages received from this Following
	if err := service.newsFeedService.DeleteByOrigin(session, following.FollowingID, "Parent record deleted"); err != nil {
		return derp.Wrap(err, location, "Deleting streams for Following", following)
	}

	// Disconnect from external services (e.g. Undo/Follow) if necessary. This only ENQUEUES a
	// post-commit task now (no blocking HTTP), so it runs synchronously on the caller's session
	// — no goroutine, no session escape. The task carries the Follow as payload, so it is safe
	// that the row above is already deleted. See POST-COMMIT-FEDERATION.md F4.
	service.Disconnect(session, following)

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
		return derp.Wrap(err, location, "Deleting Following", "userID: "+userID.Hex(), "followingID: "+followingID.Hex())
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
	return derp.Internal("service.Following.ObjectSave", "Invalid object type", object)
}

func (service *Following) ObjectDelete(session data.Session, object data.Object, note string) error {
	if following, ok := object.(*model.Following); ok {
		return service.Delete(session, following, note)
	}
	return derp.Internal("service.Following.ObjectDelete", "Invalid object type", object)
}

func (service *Following) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Following.ObjectUserCan", "Not Authorized")
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

	// RULE: a Following paused by a block rule is never polled (R8)
	criteria := exp.LessThan("nextPoll", time.Now().Unix()).
		AndNotEqual("status", model.FollowingStatusPaused)

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
		return derp.Wrap(err, location, "Loading Following", criteria)
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
		return derp.Wrap(err, location, "FollowingId must be a valid ObjectID", token, derp.WithNotFound())
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

	// Load the ActivityStream document.
	//
	// RULE: this answers "does the User already follow this actor?" -- a question about the User's OWN
	// Following records, not a request to show them the actor's content. Letting the viewer's rules
	// refuse the fetch would report a blocked or muted actor as "not followed" (or fail the caller
	// outright), so the reveal keeps the answer truthful. The verdict is still stamped on the document.
	document, err := service.activityService.UserClient(userID).Load(uri, asrules.WithReveal(true))

	if err != nil {
		return "", derp.Wrap(err, location, "Loading ActivityStream document", uri)
	}

	// If this document is not an Actor, then get the Actor of the document
	if !document.IsActor() {
		document = document.Actor()
	}

	// If this document is nil, then return an error
	if document.IsNil() {
		return "", derp.BadRequest(location, "Invalid ActivityStream document", uri)
	}

	// Look for the Actor in the Following collection
	following := model.NewFollowing()

	if err := service.LoadByURL(session, userID, document.ID(), &following); err != nil {
		if derp.IsNotFound(err) {
			return "", nil
		}
		return "", derp.Wrap(err, location, "Loading Following record", uri)
	}

	return following.ID(), nil
}

// DeleteByUserID removes all Following records for the provided userID
func (service *Following) DeleteByUserID(session data.Session, userID primitive.ObjectID, comment string) error {

	const location = "service.Following.DeleteByUserID"

	// Load all Following for the provided userID
	rangeFunc, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Loading Following", userID)
	}

	// Delete each Following record
	for following := range rangeFunc {
		if err := service.deleteNoStats(session, &following, comment); err != nil {
			return derp.Wrap(err, location, "Deleting following", following)
		}
	}

	// No Cap.
	return nil
}

// DeleteByFolder removes all Following records for the provided folderID
func (service *Following) DeleteByFolder(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID, comment string) error {

	rangeFunc, err := service.RangeByFolderID(session, userID, folderID)

	if err != nil {
		return derp.Wrap(err, "service.Folder.DeleteByFolder", "Listing folders", userID, folderID)
	}

	for folder := range rangeFunc {
		if err := service.Delete(session, &folder, comment); err != nil {
			return derp.Wrap(err, "service.Folder.DeleteByFolder", "Deleting folder", folder)
		}
	}

	// Skibidi.
	return nil
}

// PurgeNewsFeed removes all inbox items that are past their expiration date.
// TODO: HIGH: This should be rescheduled to run less frequently
func (service *Following) PurgeNewsFeed(session data.Session, following model.Following) error {

	const location = "service.Following.PurgeFollowing"

	// Check each following for expired items.
	messages, err := service.newsFeedService.RangePurgeable(session, &following)

	if err != nil {
		return derp.Wrap(err, location, "Querying purgeable items", following)
	}

	// Purge each item that has expired
	for message := range messages {
		if err := service.newsFeedService.Delete(session, &message, "Purged"); err != nil {
			return derp.Wrap(err, location, "Purging message", message)
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
		return derp.Wrap(err, location, "Saving moved Following", following)
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
		return derp.Wrap(err, "service.Following.SetStatusLoading", "Saving Following", following)
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
		return derp.Wrap(err, "service.Following.SetStatusPolling", "Saving Following", following)
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
		return derp.Wrap(err, "service.Following.SetStatusSuccess", "Saving Following", following)
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
		return derp.Wrap(err, "service.Following.SetStatusFailure", "Saving Following", following)
	}

	// Notify the user that their Following list has been changed
	service.sseUpdateChannel <- realtime.NewMessage_FollowingUpdated(following.UserID)

	return nil
}

// Pause suspends a Following that a BLOCK rule now covers (R8): it sends the Undo/Follow
// (enqueued, delivered post-commit), then marks the row PAUSED so polling stops. The row is
// kept, never auto-resumed -- the paused Following is the one-click re-follow affordance, and
// re-following runs the normal Connect flow.
func (service *Following) Pause(session data.Session, following *model.Following) error {

	const location = "service.Following.Pause"

	// A Following that is already paused has nothing more to pause
	if following.Status == model.FollowingStatusPaused {
		return nil
	}

	// Notify the remote actor that this relationship has ended
	service.Disconnect(session, following)

	// Update Following state
	following.Status = model.FollowingStatusPaused
	following.StatusMessage = "Paused by a block rule"

	// Save the Following to the database (no other business rules)
	if err := service.collection(session).Save(following, "Paused by block rule"); err != nil {
		return derp.Wrap(err, location, "Saving Following", following)
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

// setFolder guarantees that a Following points to an inbox Folder that the User owns,
// and syncs the cached Folder label.
func (service *Following) setFolder(session data.Session, following *model.Following) error {

	const location = "service.Following.setFolder"

	// RULE: An empty FolderID falls back to the User's first Folder.  Follows started from the
	// synthetic "all folders" News Feed view, and follows created via the Mastodon API, both
	// arrive without a Folder of their own.
	if following.FolderID.Value().IsZero() {

		folders, err := service.folderService.QueryByUserID(session, following.UserID)

		if err != nil {
			return derp.Wrap(err, location, "Loading Folders", following.UserID)
		}

		if len(folders) == 0 {
			return derp.BadRequest(location, "User must have at least one inbox Folder", following.UserID)
		}

		// QueryByUserID sorts by rank, so the first Folder is this User's default
		following.FolderID.Set(folders[0].FolderID)
		following.Folder = folders[0].Label

		return nil
	}

	// Search for the named Folder, scoped to this User
	folder := model.NewFolder()
	if err := service.folderService.LoadByID(session, following.UserID, following.FolderID.Value(), &folder); err != nil {

		// RULE: A missing Folder means that this User does not own it
		if derp.IsNotFound(err) {
			return derp.BadRequest(location, "Folder does not exist, or does not belong to this User", following.UserID, following.FolderID.Value())
		}

		// Any other error is a genuine failure to reach the database
		return derp.Wrap(err, location, "Loading Folder", following.UserID, following.FolderID.Value())
	}

	following.Folder = folder.Label

	// There is no Folder #2.  There is only Zuul.
	return nil
}

// reconcileAndSave folds the incoming Following onto any existing follow of the same actor, runs
// the provided save function, and -- if a concurrent create won the unique index race -- folds
// onto the winner's row and retries the save exactly once.
func (service *Following) reconcileAndSave(session data.Session, following *model.Following, save func() error) error {

	const location = "service.Following.reconcileAndSave"

	// Fold onto any existing active Following for this (userId, profileUrl) so an edit of an
	// already-followed actor updates that row instead of inserting a duplicate. (For a brand-new
	// follow, profileUrl is not resolved until Connect, so this is a no-op here.)
	if err := service.reconcileDuplicate(session, following); err != nil {
		return derp.Wrap(err, location, "Reconciling duplicate Following", following)
	}

	// First attempt
	if err := save(); err != nil {

		// RULE: a lost race trips idx_Following_User_Profile_Unique, surfaced by data-mongo as a
		// Conflict. Fold onto the winner's row and retry as an in-place update, so concurrent
		// saves of the same actor converge on one record instead of erroring. Anything else is a
		// genuine failure.
		if !derp.IsConflict(err) {
			return derp.Wrap(err, location, "Saving Following", following)
		}

		if err := service.reconcileDuplicate(session, following); err != nil {
			return derp.Wrap(err, location, "Reconciling duplicate Following after conflict", following)
		}

		if err := save(); err != nil {
			return derp.Wrap(err, location, "Saving Following after conflict", following)
		}
	}

	// Two follows enter, one row leaves.
	return nil
}

// reconcileDuplicate finds any OTHER live Following with the same (userId, profileUrl) and, if one
// exists, makes the incoming record adopt its identity -- so Save updates that row IN PLACE instead
// of inserting a second row that would violate idx_Following_User_Profile_Unique. The incoming
// settings win; the existing row's identity and creation Journal are retained. A record whose
// ProfileURL is not yet resolved (a brand-new follow before Connect) reconciles against nothing.
//
// IMPORTANT: this method MAY mutate the provided Following.
func (service *Following) reconcileDuplicate(session data.Session, following *model.Following) error {

	const location = "service.Following.reconcileDuplicate"

	// RULE: an unresolved ProfileURL has no identity to collide with (the unique index's partial
	// filter excludes it too)
	if following.ProfileURL == "" {
		return nil
	}

	criteria := exp.NotEqual("_id", following.FollowingID).
		AndEqual("userId", following.UserID).
		AndEqual("profileUrl", following.ProfileURL)

	existing := model.NewFollowing()

	if err := service.Load(session, criteria, &existing); err != nil {

		// No existing row with this actor -- nothing to reconcile
		if derp.IsNotFound(err) {
			return nil
		}

		// A genuine database error must surface, not be swallowed
		return derp.Wrap(err, location, "Loading possible duplicate", following.ProfileURL)
	}

	// If the incoming record was already persisted under its OWN id, retire that now-duplicate
	// row before adopting the winner's identity. This happens on the create path: Save inserts a
	// row before Connect resolves profileUrl, and if that resolved key already belongs to another
	// follow, the pre-inserted row would otherwise leak as an orphan. Deleted at the collection
	// level (not via Delete): it was never connected to a remote actor, so no ActivityPub Undo is
	// owed. A record that only lives in memory (IsNew, or a failed insert) has nothing to clean --
	// the LoadByID guard skips it.
	if !following.IsNew() {

		stale := model.NewFollowing()

		if loadErr := service.LoadByID(session, following.UserID, following.FollowingID, &stale); loadErr == nil {
			if err := service.collection(session).Delete(&stale, "Removing duplicate Following"); err != nil {
				return derp.Wrap(err, location, "Removing duplicate Following", stale.FollowingID)
			}
		}
	}

	// Adopt the existing row so the save becomes an in-place update carrying the incoming settings
	following.FollowingID = existing.FollowingID
	following.Journal = existing.Journal

	return nil
}
