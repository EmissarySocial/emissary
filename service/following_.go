package service

import (
	"iter"
	"math/rand"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// followingMimeStack lists the preferred mime types for follows
const followingMimeStack = "application/activity+json; q=1.0, text/html; q=0.9, application/feed+json; q=0.8, application/atom+xml; q=0.7, application/rss+xml; q=0.6, text/xml; q=0.5, */*; q=0.1"

// Following manages all interactions with the Following collection
type Following struct {
	collection      data.Collection
	streamService   *Stream
	userService     *User
	inboxService    *Inbox
	folderService   *Folder
	keyService      *EncryptionKey
	activityService *ActivityStream
	host            string
	closed          chan bool
}

// NewFollowing returns a fully populated Following service.
func NewFollowing() Following {
	return Following{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Following) Refresh(collection data.Collection, streamService *Stream, userService *User, inboxService *Inbox, folderService *Folder, keyService *EncryptionKey, activityService *ActivityStream, host string) {
	service.collection = collection
	service.streamService = streamService
	service.userService = userService
	service.inboxService = inboxService
	service.folderService = folderService
	service.keyService = keyService
	service.activityService = activityService
	service.host = host
}

// Close stops the following service watcher
func (service *Following) Close() {
	close(service.closed)
}

// Start begins the background scheduler that checks each following
// according to its own polling frequency
// TODO: HIGH: Need to make this configurable on a per-physical-server basis so that
// clusters can work together without hammering the Following collection.
func (service *Following) Start() {

	const location = "service.Following.Start"

	// Wait until the service has booted up correctly.
	for service.collection == nil {
		time.Sleep(1 * time.Minute)
	}

	// query the database every minute, looking for following that should be loaded from the web.
	for {

		// If (for some reason) the service collection is still nil, then
		// wait this one out.
		if service.collection == nil {
			continue
		}

		// Get a list of all following that can be polled
		it, err := service.ListPollable()

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error listing pollable following"))
			continue
		}

		following := model.NewFollowing()

		for it.Next(&following) {
			select {

			// If we're done, we're done.
			case <-service.closed:
				return

			default:

				// Poll each following for new items.
				if err := service.Connect(following); err != nil {
					derp.Report(derp.Wrap(err, location, "Error connecting to remote server"))
				}

				if err := service.PurgeInbox(following); err != nil {
					derp.Report(derp.Wrap(err, location, "Error purghing inbox"))
				}
			}

			following = model.NewFollowing()
		}

		// Poll randomly between 45 and 75 minutes
		time.Sleep(time.Duration(rand.Intn(30)+45) * time.Minute)
	}
}

/******************************************
 * Common Data Methods
 ******************************************/

// New creates a newly initialized Following that is ready to use
func (service *Following) New() model.Following {
	return model.NewFollowing()
}

// Count returns the number of records that match the provided criteria
func (service *Following) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an iterator containing all of the Following who match the provided criteria
func (service *Following) Query(criteria exp.Expression, options ...option.Option) ([]model.Following, error) {
	result := make([]model.Following, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Following who match the provided criteria
func (service *Following) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Following records that match the provided criteria
func (service *Following) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Following], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Following.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewFollowing), nil
}

// Load retrieves an Following from the database
func (service *Following) Load(criteria exp.Expression, result *model.Following) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Following.Load", "Error loading Following", criteria)
	}

	return nil
}

// Save adds/updates an Following in the database
func (service *Following) Save(following *model.Following, note string) error {

	const location = "service.Following.Save"

	// TODO: LOW: Add duplicate checks to this function?

	// RULE: Reset status and error counts when saving
	following.Method = model.FollowingMethodPoll
	following.Status = model.FollowingStatusNew
	following.StatusMessage = ""
	following.ErrorCount = 0

	if following.Behavior == "" {
		following.Behavior = model.FollowingBehaviorPostsAndReplies
	}

	if following.RuleAction == "" {
		following.RuleAction = model.RuleActionLabel
	}

	// Validate the value before saving
	if err := service.Schema().Validate(following); err != nil {
		return derp.Wrap(err, location, "Error validating Following", following)
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

	// Prevent duplicate following records
	if err := service.preventDuplicates(following); err != nil {
		return derp.Wrap(err, location, "Error preventing duplicate", following)
	}

	// RULE: Set the Folder Name
	folder := model.NewFolder()
	if err := service.folderService.LoadByID(following.UserID, following.FolderID, &folder); err == nil {
		following.Folder = folder.Label
	}

	// Save the following to the database
	if err := service.collection.Save(following, note); err != nil {
		return derp.Wrap(err, location, "Error saving Following", following, note)
	}

	// RULE: Update messages if requested by the UX
	go service.inboxService.UpdateInboxFolders(following.UserID, following.FollowingID, following.FolderID)

	// Recalculate the follower count for this user
	go service.userService.CalcFollowingCount(following.UserID)

	// Connect to external services and discover the best update method.
	// This will also update the status again, soon.
	go service.RefreshAndConnect(*following)

	// Win!
	return nil
}

// Delete removes an Following from the database (virtual delete)
func (service *Following) Delete(following *model.Following, note string) error {

	const location = "service.Following.Delete"

	if err := service.deleteNoStats(following, note); err != nil {
		return derp.Wrap(err, location, "Error deleting Following", following, note)
	}

	// Recalculate the follower count for this user
	go service.userService.CalcFollowingCount(following.UserID)

	// Recalculate the unread count for this folder
	go derp.Report(service.folderService.ReCalculateUnreadCountFromFolder(following.UserID, following.FolderID))

	return nil
}

// deleteNoStats removes an Following from the database (virtual delete)
// but DOES NOT recompute statistics for parent records.  This is useful when
// cascading deletes, because there's no reason to recompute statistics for
// records that will be deleted.
func (service *Following) deleteNoStats(following *model.Following, comment string) error {

	const location = "service.Following.deleteNoStats"

	// Remove the Following record
	if err := service.collection.Delete(following, comment); err != nil {
		return derp.Wrap(err, location, "Error deleting Following", following, comment)
	}

	// Remove any messages received from this Following
	if err := service.inboxService.DeleteByOrigin(following.FollowingID, "Parent record deleted"); err != nil {
		return derp.Wrap(err, location, "Error deleting streams for Following", following)
	}

	// Disconnect from external services (if necessary)
	service.Disconnect(following)

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Following) ObjectType() string {
	return "Following"
}

// New returns a fully initialized model.Stream as a data.Object.
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

func (service *Following) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Following) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewFollowing()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Following) ObjectSave(object data.Object, note string) error {
	if following, ok := object.(*model.Following); ok {
		return service.Save(following, note)
	}
	return derp.NewInternalError("service.Following.ObjectSave", "Invalid object type", object)
}

func (service *Following) ObjectDelete(object data.Object, note string) error {
	if following, ok := object.(*model.Following); ok {
		return service.Delete(following, note)
	}
	return derp.NewInternalError("service.Following.ObjectDelete", "Invalid object type", object)
}

func (service *Following) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Following.ObjectUserCan", "Not Authorized")
}

func (service *Following) Schema() schema.Schema {
	return schema.New(model.FollowingSchema())
}

/******************************************
 * ActivityPub Queries
 ******************************************/

func (service *Following) ListActivityPub(userID primitive.ObjectID, options ...option.Option) (data.Iterator, error) {
	criteria := exp.Equal("userId", userID).
		AndEqual("method", model.FollowingMethodActivityPub)

	return service.List(criteria, options...)
}

/******************************************
 * Custom Queries
 ******************************************/

// QueryByUser returns a slice of all following for a given user
func (service *Following) QueryByUser(userID primitive.ObjectID) ([]model.FollowingSummary, error) {
	result := make([]model.FollowingSummary, 0)
	criteria := exp.Equal("userId", userID)
	err := service.collection.Query(&result, notDeleted(criteria), option.Fields(model.FollowingSummaryFields()...), option.SortAsc("label"))
	return result, err
}

// QueryByFolder returns a slice of all following for a given user
func (service *Following) QueryByFolder(userID primitive.ObjectID, folderID primitive.ObjectID) ([]model.FollowingSummary, error) {
	result := make([]model.FollowingSummary, 0)
	criteria := exp.Equal("userId", userID).AndEqual("folderId", folderID)
	err := service.collection.Query(&result, notDeleted(criteria), option.Fields(model.FollowingSummaryFields()...), option.SortAsc("label"))
	return result, err
}

// QueryByFolderAndExp returns a slice of all following for a given user
func (service *Following) QueryByFolderAndExp(userID primitive.ObjectID, folderID primitive.ObjectID, criteria exp.Expression) ([]model.FollowingSummary, error) {

	result := make([]model.FollowingSummary, 0)
	criteria = criteria.
		AndEqual("userId", userID).
		AndEqual("folderId", folderID)

	err := service.collection.Query(&result, notDeleted(criteria), option.Fields(model.FollowingSummaryFields()...), option.SortAsc("label"))
	return result, err
}

// ListPollable returns an iterator of all following that are ready to be polled
func (service *Following) ListPollable() (data.Iterator, error) {
	criteria := exp.LessThan("nextPoll", time.Now().Unix()).
		AndNotEqual("method", model.FollowingMethodActivityPub) // Don't poll ActivityPub

	return service.List(criteria, option.SortAsc("lastPolled"))
}

// RangeByUserID returns an iterator of all following for a given userID
func (service *Following) RangeByUserID(userID primitive.ObjectID) (iter.Seq[model.Following], error) {
	criteria := exp.Equal("userId", userID)
	return service.Range(criteria, option.SortAsc("lastPolled"))
}

// LoadByID retrieves an Following from the database.  UserID is required to prevent
// people from snooping on other's following.
func (service *Following) LoadByID(userID primitive.ObjectID, followingID primitive.ObjectID, result *model.Following) error {

	criteria := exp.Equal("_id", followingID).
		AndEqual("userId", userID)

	if err := service.Load(criteria, result); err != nil {
		return derp.Wrap(err, "service.Following.LoadByID", "Error loading Following", criteria)
	}

	return nil
}

// LoadByToken loads an individual following using a string version of the following ID
func (service *Following) LoadByToken(userID primitive.ObjectID, token string, result *model.Following) error {

	if token == "new" {
		*result = model.NewFollowing()
		result.UserID = userID
		return nil
	}

	followingID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Following.LoadByToken", "Error parsing followingId", token)
	}

	return service.LoadByID(userID, followingID, result)
}

// LoadByURL loads an individual following using the target URL that is being followed
func (service *Following) LoadByURL(userID primitive.ObjectID, profileUrl string, result *model.Following) error {

	criteria := exp.Equal("userId", userID).
		AndEqual("profileUrl", profileUrl)

	return service.Load(criteria, result)
}

/******************************************
 * Custom Actions
 ******************************************/

func (service *Following) GetFollowingID(userID primitive.ObjectID, uri string) (string, error) {

	const location = "service.Following.IsFollowing"

	// Load the ActivityStream document
	document, err := service.activityService.Load(uri)

	if err != nil {
		return "", derp.Wrap(err, location, "Error loading ActivityStream document", uri)
	}

	// If this document is not an Actor, then get the Actor of the document
	if !document.IsActor() {
		document = document.Actor()
	}

	if document.IsNil() {
		return "", derp.NewBadRequestError(location, "Invalid ActivityStream document", uri)
	}

	// Look for the Actor in the Following collection
	following := model.NewFollowing()

	if err := service.LoadByURL(userID, document.ID(), &following); err == nil {
		return following.ID(), nil
	} else if derp.NotFound(err) {
		return "", nil
	} else {
		return "", derp.Wrap(err, location, "Error loading Following record", uri)
	}
}

// DeleteByUserID removes all Following records for the provided userID
func (service *Following) DeleteByUserID(userID primitive.ObjectID, comment string) error {

	const location = "service.Following.DeleteByUserID"

	// Load all Following for the provided userID
	rangeFunc, err := service.RangeByUserID(userID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading following", userID)
	}

	// Delete each Following record
	for following := range rangeFunc {
		if err := service.deleteNoStats(&following, comment); err != nil {
			return derp.Wrap(err, location, "Error deleting following", following)
		}
	}

	// No Cap.
	return nil
}

// PurgeInbox removes all inbox items that are past their expiration date.
// TODO: LOW: Should this be in the Inbox service?
func (service *Following) PurgeInbox(following model.Following) error {

	const location = "service.Following.PurgeFollowing"

	// Check each following for expired items.
	items, err := service.inboxService.QueryPurgeable(&following)

	// If there was an error querying for purgeable items, log it and exit.
	if err != nil {
		return derp.Wrap(err, location, "Error querying purgeable items", following)
	}

	// Purge each item that has expired
	for _, item := range items {
		if err := service.inboxService.Delete(&item, "Purged"); err != nil {
			return derp.Wrap(err, location, "Error purging item", item)
		}
	}

	return nil
}

/******************************************
 * Other Updates Methods
 ******************************************/

// SetStatusLoading updates a Following record with the "Loading" status
func (service *Following) SetStatusLoading(following *model.Following) error {

	// Update Following state
	following.Status = model.FollowingStatusLoading
	following.StatusMessage = ""
	following.LastPolled = time.Now().Unix()

	// Save the Following to the database
	return service.collection.Save(following, "Updating status")
}

// SetStatusSuccess updates a Following record with the "Success" status and
// resets the error count to zero.
func (service *Following) SetStatusSuccess(following *model.Following) error {

	// Update Following state
	following.Status = model.FollowingStatusSuccess
	following.StatusMessage = ""

	following.NextPoll = following.LastPolled + int64(following.PollDuration*60*60)
	following.ErrorCount = 0

	// Save the Following to the database
	return service.collection.Save(following, "Updating status")
}

// SetStatusFailure updates a Following record to the "Failure" status and
// increments the error count.
func (service *Following) SetStatusFailure(following *model.Following, statusMessage string) error {

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

	// Save the Following to the database
	return service.collection.Save(following, "Updating status")
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

func (service *Following) preventDuplicates(current *model.Following) error {

	// Search the database for the original record
	original := model.NewFollowing()
	if err := service.LoadByURL(current.UserID, current.URL, &original); err != nil {
		if derp.NotFound(err) {
			return nil
		}
		return derp.Wrap(err, "service.Following.preventDuplicate", "Error loading Following", current)
	}

	// Delete the original record
	if err := service.Delete(&original, "removing duplicate"); err != nil {
		return derp.Wrap(err, "service.Following.preventDuplicate", "Error deleting original", original)
	}

	return nil
}
