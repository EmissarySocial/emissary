package service

import (
	"math/rand"
	"strings"
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
const followingMimeStack = "application/feed+json; q=1.0, application/json; q=0.9, application/atom+xml; q=0.8, application/rss+xml; q=0.7, application/xml; q=0.6, text/xml; q=0.5, text/html; q=0.4, */*; q=0.1"

// Following manages all interactions with the Following collection
type Following struct {
	collection    data.Collection
	streamService *Stream
	userService   *User
	inboxService  *Inbox
	keyService    *EncryptionKey
	host          string
	closed        chan bool
}

// NewFollowing returns a fully populated Following service.
func NewFollowing(collection data.Collection, streamService *Stream, userService *User, inboxService *Inbox, keyService *EncryptionKey, host string) Following {

	service := Following{
		collection:    collection,
		streamService: streamService,
		userService:   userService,
		inboxService:  inboxService,
		keyService:    keyService,
		host:          host,
		closed:        make(chan bool),
	}

	service.Refresh(collection)

	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Following) Refresh(collection data.Collection) {
	service.collection = collection
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

	rand.Seed(time.Now().UnixNano())

	// query the database every minute, looking for following that should be loaded from the web.
	for {

		// Poll randomly between 1 and 5 minutes
		// time.Sleep(time.Duration(rand.Intn(5)+1) * time.Minute)
		time.Sleep(time.Duration(10 * time.Second))

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
				service.Connect(following)
				service.PurgeInbox(following)
			}

			following = model.NewFollowing()
		}

		// Poll randomly between 1 and 5 minutes
		time.Sleep(time.Duration(rand.Intn(5)+1) * time.Minute)
	}
}

/******************************************
 * Common Data Methods
 ******************************************/

// New creates a newly initialized Following that is ready to use
func (service *Following) New() model.Following {
	return model.NewFollowing()
}

// Query returns an iterator containing all of the Following who match the provided criteria
func (service *Following) Query(criteria exp.Expression, options ...option.Option) ([]model.Following, error) {
	result := make([]model.Following, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Following who match the provided criteria
func (service *Following) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
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

	// RULE: Reset status and error counts when saving
	following.Method = model.FollowMethodPoll
	following.Status = model.FollowingStatusNew
	following.StatusMessage = ""
	following.ErrorCount = 0

	// RULE: Hacky way to make the URL valid.  This should
	// probably be handled in the schema URL validation.
	switch {
	case strings.HasPrefix(following.URL, "https://"):
	case strings.HasPrefix(following.URL, "http://"):
	default:
		following.URL = "https://" + following.URL
	}

	// Clean the value before saving
	if err := service.Schema().Clean(following); err != nil {
		return derp.Wrap(err, "service.Following.Save", "Error cleaning Following", following)
	}

	// Save the following to the database
	if err := service.collection.Save(following, note); err != nil {
		return derp.Wrap(err, "service.Following.Save", "Error saving Following", following, note)
	}

	// RULE: Update messages if requested by the UX
	if following.DoMoveMessages {
		go service.inboxService.UpdateInboxFolders(following.UserID, following.FollowingID, following.FolderID)
	}

	// Recalculate the follower count for this user
	go service.userService.CalcFollowingCount(following.UserID)

	// Connect to external services and discover the best update method
	go service.Connect(*following)

	// Win!
	return nil
}

// Delete removes an Following from the database (virtual delete)
func (service *Following) Delete(following *model.Following, note string) error {

	if err := service.collection.Delete(following, note); err != nil {
		return derp.Wrap(err, "service.Following.Delete", "Error deleting Following", following, note)
	}

	if err := service.inboxService.DeleteByOrigin(following.FollowingID, "Parent record deleted"); err != nil {
		return derp.Wrap(err, "service.Following.Delete", "Error deleting streams for Following", following, note)
	}

	// Recalculate the follower count for this user
	go service.userService.CalcFollowingCount(following.UserID)

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

func (service *Following) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
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
		AndEqual("method", model.FollowMethodActivityPub)

	return service.List(criteria, options...)
}

/******************************************
 * Custom Queries
 ******************************************/

// QueryByUserID returns a slice of all following for a given user
func (service *Following) QueryByUserID(userID primitive.ObjectID) ([]model.FollowingSummary, error) {
	result := make([]model.FollowingSummary, 0)
	criteria := exp.Equal("userId", userID)
	err := service.collection.Query(&result, notDeleted(criteria))
	return result, err
}

// ListPollable returns an iterator of all following that are ready to be polled
func (service *Following) ListPollable() (data.Iterator, error) {
	criteria := exp.LessThan("nextPoll", time.Now().Unix())
	return service.List(criteria, option.SortAsc("lastPolled"))
}

// ListByUserID returns an iterator of all following for a given user
func (service *Following) ListByUserID(userID primitive.ObjectID) (data.Iterator, error) {
	criteria := exp.Equal("userId", userID)
	return service.List(criteria, option.SortAsc("lastPolled"))
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

// LoadByURL loads an infividual following using the target URL that is being followed
func (service *Following) LoadByURL(parentID primitive.ObjectID, url string, result *model.Following) error {

	criteria := exp.Equal("parentId", parentID).
		AndEqual("url", url)

	return service.Load(criteria, result)
}

/******************************************
 * Custom Actions
 ******************************************/

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

func (service *Following) CallbackURL() string {
	return service.host + "/.websub"
}

/******************************************
 * ActivityPub Methods
 ******************************************/

func (service *Following) ActivityPubID(following *model.Following) string {
	return service.host + "/@" + following.UserID.Hex() + "/pub/following/" + following.FollowingID.Hex()
}

func (service *Following) ActivityPubActorID(following *model.Following) string {
	return service.host + "/@" + following.UserID.Hex()
}

func (service *Following) AsJSONLD(following *model.Following) mapof.Any {

	return mapof.Any{
		"@context": vocab.ContextTypeActivityStreams,
		"id":       service.ActivityPubID(following),
		"type":     vocab.ActivityTypeFollow,
		"actor":    service.ActivityPubActorID(following),
		"object":   following.URL,
	}
}
